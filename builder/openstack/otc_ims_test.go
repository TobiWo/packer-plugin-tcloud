// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package openstack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud"
)

func testOTCIMSClient(endpoint string) *gophercloud.ServiceClient {
	return &gophercloud.ServiceClient{
		ProviderClient: &gophercloud.ProviderClient{
			TokenID:    "test-token",
			HTTPClient: http.Client{},
		},
		Endpoint: gophercloud.NormalizeURL(endpoint),
		Type:     "image",
	}
}

func TestCreateOTCIMSImage(t *testing.T) {
	oldPollInterval := otcIMSJobPollInterval
	otcIMSJobPollInterval = time.Millisecond
	defer func() { otcIMSJobPollInterval = oldPollInterval }()

	var createSeen bool
	jobRequests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-Token") != "test-token" {
			t.Fatalf("expected auth token header, got %q", r.Header.Get("X-Auth-Token"))
		}

		switch r.URL.Path {
		case "/v2/cloudimages/action":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST, got %s", r.Method)
			}
			createSeen = true
			var req otcIMSCreateImageRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("failed to decode request: %s", err)
			}
			if req.Name != "rhel-image" || req.InstanceID != "server-id" || req.Type != "ECS" {
				t.Fatalf("unexpected create request: %#v", req)
			}
			if strings.Join(req.Tags, ",") != "custom.rhel,source.packer" {
				t.Fatalf("unexpected tags: %#v", req.Tags)
			}
			if req.EnterpriseProjectID != "0" {
				t.Fatalf("unexpected enterprise project ID: %q", req.EnterpriseProjectID)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"job_id":"job-id"}`))
		case "/v1/project-id/jobs/job-id":
			if r.Method != http.MethodGet {
				t.Fatalf("expected GET, got %s", r.Method)
			}
			jobRequests++
			w.WriteHeader(http.StatusOK)
			if jobRequests == 1 {
				_, _ = w.Write([]byte(`{"status":"RUNNING","job_id":"job-id"}`))
				return
			}
			_, _ = w.Write([]byte(`{"status":"SUCCESS","job_id":"job-id","entities":{"image_id":"image-id"}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	imageID, err := createOTCIMSImage(context.Background(), testOTCIMSClient(srv.URL), otcIMSCreateImageRequest{
		Name:                "rhel-image",
		InstanceID:          "server-id",
		Type:                "ECS",
		Tags:                []string{"custom.rhel", "source.packer"},
		EnterpriseProjectID: "0",
	}, "project-id")
	if err != nil {
		t.Fatalf("shouldn't have err: %s", err)
	}
	if imageID != "image-id" {
		t.Fatalf("expected image-id, got %q", imageID)
	}
	if !createSeen {
		t.Fatal("expected create request")
	}
	if jobRequests != 2 {
		t.Fatalf("expected two job polling requests, got %d", jobRequests)
	}
}

func TestWaitForOTCIMSImageJob_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"FAIL","error_code":"IMG.0001","fail_reason":"bad image"}`))
	}))
	defer srv.Close()

	_, err := waitForOTCIMSImageJob(context.Background(), testOTCIMSClient(srv.URL), "project-id", "job-id")
	if err == nil || !strings.Contains(err.Error(), "IMG.0001") || !strings.Contains(err.Error(), "bad image") {
		t.Fatalf("expected failure details, got %v", err)
	}
}

func TestWaitForOTCIMSImageJob_MissingImageID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"SUCCESS","entities":{}}`))
	}))
	defer srv.Close()

	_, err := waitForOTCIMSImageJob(context.Background(), testOTCIMSClient(srv.URL), "project-id", "job-id")
	if err == nil || !strings.Contains(err.Error(), "entities.image_id") {
		t.Fatalf("expected missing image ID error, got %v", err)
	}
}

func TestWaitForOTCIMSImageJob_Cancel(t *testing.T) {
	oldPollInterval := otcIMSJobPollInterval
	otcIMSJobPollInterval = time.Hour
	defer func() { otcIMSJobPollInterval = oldPollInterval }()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"RUNNING"}`))
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := waitForOTCIMSImageJob(ctx, testOTCIMSClient(srv.URL), "project-id", "job-id")
	if err == nil || err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
