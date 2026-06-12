// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package openstack

import (
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	tokens3 "github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
)

const otcAKSKIdentityMethod = "hw_ak_sk"

type otcAKSKAuthOptions struct {
	AccessKey   string
	SecretKey   string
	ProjectID   string
	AllowReauth bool
}

func (opts otcAKSKAuthOptions) ToTokenV3CreateMap(scope map[string]interface{}) (map[string]interface{}, error) {
	if opts.AccessKey == "" {
		return nil, fmt.Errorf("otc_access_key must be specified when using OTC AK/SK authentication")
	}
	if opts.SecretKey == "" {
		return nil, fmt.Errorf("otc_secret_key must be specified when using OTC AK/SK authentication")
	}

	auth := map[string]interface{}{
		"identity": map[string]interface{}{
			"methods": []string{otcAKSKIdentityMethod},
			otcAKSKIdentityMethod: map[string]interface{}{
				"access": map[string]interface{}{
					"key": opts.AccessKey,
				},
				"secret": map[string]interface{}{
					"key": opts.SecretKey,
				},
			},
		},
	}
	if len(scope) != 0 {
		auth["scope"] = scope
	}

	return map[string]interface{}{"auth": auth}, nil
}

func (opts otcAKSKAuthOptions) ToTokenV3ScopeMap() (map[string]interface{}, error) {
	if opts.ProjectID == "" {
		return nil, fmt.Errorf("tenant_id must be specified when using OTC AK/SK authentication")
	}

	return map[string]interface{}{
		"project": map[string]interface{}{
			"id": opts.ProjectID,
		},
	}, nil
}

func (opts otcAKSKAuthOptions) ToTokenV3HeadersMap(map[string]interface{}) (map[string]string, error) {
	return nil, nil
}

func (opts otcAKSKAuthOptions) CanReauth() bool {
	return opts.AllowReauth
}

func (c *AccessConfig) authenticateOTCAKSK(client *gophercloud.ProviderClient) error {
	return openstack.AuthenticateV3(client, tokens3.AuthOptionsBuilder(otcAKSKAuthOptions{
		AccessKey:   c.OTCAccessKey,
		SecretKey:   c.OTCSecretKey,
		ProjectID:   c.TenantID,
		AllowReauth: true,
	}), gophercloud.EndpointOpts{})
}
