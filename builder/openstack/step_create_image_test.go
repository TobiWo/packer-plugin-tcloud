// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package openstack

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud"
	gophercloudopenstack "github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

func TestStepCreateImageValidatesGlanceBeforeComputeCreate(t *testing.T) {
	createRequests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		createRequests++
		w.Header().Set("Location", "http://example.com/v2/images/image-id")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	config := testCreateImageConfigWithMissingImageEndpoint(t, srv.URL)
	config.ImageCreationMethod = imageCreationMethodOpenStack
	config.ImageName = "test-image"

	state := testCreateImageState(config)
	action := (&stepCreateImage{}).Run(context.Background(), state)

	assertCreateImageHaltedBeforeCreate(t, action, state, createRequests)
}

func TestStepCreateImageValidatesGlanceBeforeBlockStorageUpload(t *testing.T) {
	createRequests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		createRequests++
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	config := testCreateImageConfigWithMissingImageEndpoint(t, srv.URL)
	config.ImageCreationMethod = imageCreationMethodOpenStack
	config.ImageMetadata = map[string]string{"image_type": "image"}
	config.ImageName = "test-image"

	state := testCreateImageState(config)
	state.Put("volume_id", "volume-id")
	action := (&stepCreateImage{UseBlockStorageVolume: true}).Run(context.Background(), state)

	assertCreateImageHaltedBeforeCreate(t, action, state, createRequests)
}

func TestStepCreateImageValidatesGlanceBeforeOTCIMSCreate(t *testing.T) {
	createRequests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		createRequests++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"job_id":"job-id"}`))
	}))
	defer srv.Close()

	config := testCreateImageConfigWithMissingImageEndpoint(t, srv.URL)
	config.ImageCreationMethod = imageCreationMethodOTC
	config.ImageName = "test-image"
	config.OTCIMSEndpoint = srv.URL
	config.TenantID = "project-id"

	state := testCreateImageState(config)
	action := (&stepCreateImage{}).Run(context.Background(), state)

	assertCreateImageHaltedBeforeCreate(t, action, state, createRequests)
}

func testCreateImageConfigWithMissingImageEndpoint(t *testing.T, endpoint string) *Config {
	t.Helper()

	provider, err := gophercloudopenstack.NewClient("http://example.com/v3")
	if err != nil {
		t.Fatalf("failed to create provider client: %s", err)
	}
	provider.SetToken("test-token")
	provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		switch opts.Type {
		case "compute", "volumev3":
			return gophercloud.NormalizeURL(endpoint), nil
		case "image":
			return "", errors.New("missing image endpoint")
		default:
			return "", fmt.Errorf("unexpected endpoint type %q", opts.Type)
		}
	}

	return &Config{
		AccessConfig: AccessConfig{
			osClient: provider,
		},
	}
}

func testCreateImageState(config *Config) *multistep.BasicStateBag {
	state := &multistep.BasicStateBag{}
	state.Put("config", config)
	state.Put("server", &servers.Server{ID: "server-id"})
	state.Put("ui", &packersdk.BasicUi{
		Writer:      io.Discard,
		ErrorWriter: io.Discard,
	})
	return state
}

func assertCreateImageHaltedBeforeCreate(t *testing.T, action multistep.StepAction, state multistep.StateBag, createRequests int) {
	t.Helper()

	if action != multistep.ActionHalt {
		t.Fatalf("expected halt, got %s", action)
	}
	if createRequests != 0 {
		t.Fatalf("expected no create requests before Glance validation, got %d", createRequests)
	}
	if _, ok := state.GetOk("image"); ok {
		t.Fatalf("expected no image in state, got %v", state.Get("image"))
	}
	err, ok := state.Get("error").(error)
	if !ok || !strings.Contains(err.Error(), "image service client") {
		t.Fatalf("expected image service client error, got %v", state.Get("error"))
	}
}
