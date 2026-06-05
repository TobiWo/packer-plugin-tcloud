// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package openstack

import (
	"context"
	"fmt"
	"time"

	"github.com/gophercloud/gophercloud"
)

var otcIMSJobPollInterval = 2 * time.Second

type otcIMSCreateImageRequest struct {
	Name                string   `json:"name"`
	InstanceID          string   `json:"instance_id"`
	Type                string   `json:"type"`
	Tags                []string `json:"tags,omitempty"`
	EnterpriseProjectID string   `json:"enterprise_project_id,omitempty"`
}

type otcIMSCreateImageResponse struct {
	JobID string `json:"job_id"`
}

type otcIMSJobResponse struct {
	Status     string            `json:"status"`
	JobID      string            `json:"job_id"`
	JobType    string            `json:"job_type"`
	ErrorCode  string            `json:"error_code"`
	FailReason string            `json:"fail_reason"`
	Entities   otcIMSJobEntities `json:"entities"`
}

type otcIMSJobEntities struct {
	ImageID   string `json:"image_id"`
	ImageName string `json:"image_name"`
}

func (c *AccessConfig) otcIMSClient(ctx context.Context, endpoint string) *gophercloud.ServiceClient {
	provider := *c.osClient
	provider.Context = ctx
	return &gophercloud.ServiceClient{
		ProviderClient: &provider,
		Endpoint:       gophercloud.NormalizeURL(endpoint),
		Type:           "image",
	}
}

func createOTCIMSImage(ctx context.Context, client *gophercloud.ServiceClient, opts otcIMSCreateImageRequest, projectID string) (string, error) {
	var createResp otcIMSCreateImageResponse
	_, err := client.Post(
		client.ServiceURL("v2", "cloudimages", "action"),
		opts,
		&createResp,
		&gophercloud.RequestOpts{OkCodes: []int{200}},
	)
	if err != nil {
		return "", err
	}
	if createResp.JobID == "" {
		return "", fmt.Errorf("OTC IMS create image response did not include job_id")
	}

	return waitForOTCIMSImageJob(ctx, client, projectID, createResp.JobID)
}

func waitForOTCIMSImageJob(ctx context.Context, client *gophercloud.ServiceClient, projectID, jobID string) (string, error) {
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}

		var job otcIMSJobResponse
		_, err := client.Get(
			client.ServiceURL("v1", projectID, "jobs", jobID),
			&job,
			&gophercloud.RequestOpts{OkCodes: []int{200}},
		)
		if err != nil {
			return "", err
		}

		switch job.Status {
		case "SUCCESS":
			if job.Entities.ImageID == "" {
				return "", fmt.Errorf("OTC IMS job %s succeeded without entities.image_id", jobID)
			}
			return job.Entities.ImageID, nil
		case "FAIL":
			return "", fmt.Errorf("OTC IMS job %s failed: %s %s", jobID, job.ErrorCode, job.FailReason)
		case "INIT", "RUNNING":
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(otcIMSJobPollInterval):
			}
		default:
			return "", fmt.Errorf("OTC IMS job %s returned unexpected status %q", jobID, job.Status)
		}
	}
}
