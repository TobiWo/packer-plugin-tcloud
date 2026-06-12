// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MPL-2.0

package openstack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

func TestOTCAKSKAuthOptionsBuildsTokenRequest(t *testing.T) {
	opts := otcAKSKAuthOptions{
		AccessKey:   "access-key",
		SecretKey:   "secret-key",
		ProjectID:   "project-id",
		AllowReauth: true,
	}

	scope, err := opts.ToTokenV3ScopeMap()
	if err != nil {
		t.Fatalf("scope failed: %s", err)
	}

	body, err := opts.ToTokenV3CreateMap(scope)
	if err != nil {
		t.Fatalf("create map failed: %s", err)
	}

	auth := body["auth"].(map[string]interface{})
	identity := auth["identity"].(map[string]interface{})
	methods := identity["methods"].([]string)
	if len(methods) != 1 || methods[0] != "hw_ak_sk" {
		t.Fatalf("expected hw_ak_sk method, got %#v", methods)
	}

	aksk := identity["hw_ak_sk"].(map[string]interface{})
	access := aksk["access"].(map[string]interface{})
	secret := aksk["secret"].(map[string]interface{})
	if access["key"] != "access-key" {
		t.Fatalf("expected access key in request, got %#v", access["key"])
	}
	if secret["key"] != "secret-key" {
		t.Fatalf("expected secret key in request, got %#v", secret["key"])
	}

	project := auth["scope"].(map[string]interface{})["project"].(map[string]interface{})
	if project["id"] != "project-id" {
		t.Fatalf("expected project scope, got %#v", project)
	}
	if !opts.CanReauth() {
		t.Fatalf("expected reauth to be enabled")
	}
}

func TestOTCAKSKAuthValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      AccessConfig
		errContains string
	}{
		{
			name: "missing secret key",
			config: AccessConfig{
				OTCAccessKey: "access-key",
				TenantID:     "project-id",
			},
			errContains: "otc_access_key and otc_secret_key must be provided together",
		},
		{
			name: "missing access key",
			config: AccessConfig{
				OTCSecretKey: "secret-key",
				TenantID:     "project-id",
			},
			errContains: "otc_access_key and otc_secret_key must be provided together",
		},
		{
			name: "missing tenant id",
			config: AccessConfig{
				OTCAccessKey: "access-key",
				OTCSecretKey: "secret-key",
			},
			errContains: "tenant_id must be specified when using OTC AK/SK authentication",
		},
		{
			name: "token mixed with ak sk",
			config: AccessConfig{
				OTCAccessKey: "access-key",
				OTCSecretKey: "secret-key",
				TenantID:     "project-id",
				Token:        "token",
			},
			errContains: "OTC AK/SK authentication cannot be combined with token",
		},
		{
			name: "password mixed with ak sk",
			config: AccessConfig{
				OTCAccessKey: "access-key",
				OTCSecretKey: "secret-key",
				TenantID:     "project-id",
				Username:     "user",
				Password:     "password",
			},
			errContains: "OTC AK/SK authentication cannot be combined with username/password",
		},
		{
			name: "application credential mixed with ak sk",
			config: AccessConfig{
				OTCAccessKey:                "access-key",
				OTCSecretKey:                "secret-key",
				TenantID:                    "project-id",
				ApplicationCredentialID:     "credential-id",
				ApplicationCredentialSecret: "credential-secret",
			},
			errContains: "OTC AK/SK authentication cannot be combined with application credentials",
		},
		{
			name: "cloud mixed with ak sk",
			config: AccessConfig{
				OTCAccessKey: "access-key",
				OTCSecretKey: "secret-key",
				TenantID:     "project-id",
				Cloud:        "cloud",
			},
			errContains: "OTC AK/SK authentication cannot be combined with cloud",
		},
		{
			name: "valid ak sk",
			config: AccessConfig{
				OTCAccessKey: "access-key",
				OTCSecretKey: "secret-key",
				TenantID:     "project-id",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validateOTCAKSKAuth()
			if tt.errContains == "" {
				if err != nil {
					t.Fatalf("expected no error, got %s", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("expected error containing %q, got %v", tt.errContains, err)
			}
		})
	}
}

func TestOTCAKSKAuthenticateUsesTokenEndpointAndReauth(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/auth/tokens" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("X-Auth-Token") != "" {
			t.Fatalf("expected empty X-Auth-Token header, got %q", r.Header.Get("X-Auth-Token"))
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %s", err)
		}
		auth := body["auth"].(map[string]interface{})
		identity := auth["identity"].(map[string]interface{})
		if _, ok := identity["hw_ak_sk"]; !ok {
			t.Fatalf("expected hw_ak_sk identity in request: %#v", identity)
		}

		requests++
		w.Header().Set("X-Subject-Token", "issued-token")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"token": {
				"catalog": [{
					"type": "compute",
					"name": "nova",
					"endpoints": [{
						"interface": "public",
						"region": "eu-de",
						"url": "https://compute.example/v2.1"
					}]
				}]
			}
		}`))
	}))
	defer server.Close()

	provider, err := openstack.NewClient(server.URL + "/v3/")
	if err != nil {
		t.Fatalf("client failed: %s", err)
	}
	config := AccessConfig{
		OTCAccessKey: "access-key",
		OTCSecretKey: "secret-key",
		TenantID:     "project-id",
	}

	if err := config.authenticateOTCAKSK(provider); err != nil {
		t.Fatalf("authenticate failed: %s", err)
	}
	if token := provider.Token(); token != "issued-token" {
		t.Fatalf("expected issued token, got %q", token)
	}
	if provider.ReauthFunc == nil {
		t.Fatalf("expected reauth function")
	}
	if requests != 1 {
		t.Fatalf("expected one auth request, got %d", requests)
	}

	if err := provider.ReauthFunc(); err != nil {
		t.Fatalf("reauth failed: %s", err)
	}
	if requests != 2 {
		t.Fatalf("expected reauth request, got %d requests", requests)
	}
}

func TestAccessConfigPrepareUsesOTCAKSKAuthentication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/auth/tokens" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}

		w.Header().Set("X-Subject-Token", "issued-token")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"token": {
				"catalog": [{
					"type": "compute",
					"name": "nova",
					"endpoints": [{
						"interface": "public",
						"region": "eu-de",
						"url": "https://compute.example/v2.1"
					}]
				}]
			}
		}`))
	}))
	defer server.Close()

	config := AccessConfig{
		IdentityEndpoint: server.URL + "/v3/",
		OTCAccessKey:     "access-key",
		OTCSecretKey:     "secret-key",
		TenantID:         "project-id",
	}

	if errs := config.Prepare(nil); len(errs) != 0 {
		t.Fatalf("expected prepare to succeed, got %v", errs)
	}
	if token := config.osClient.Token(); token != "issued-token" {
		t.Fatalf("expected issued token, got %q", token)
	}
}

func TestAccessConfigRegisterSecretsFiltersOTCAKSKValues(t *testing.T) {
	config := AccessConfig{
		Password:     "unique-password-for-aksk-filter-test",
		Token:        "unique-config-token-for-aksk-filter-test",
		OTCSecretKey: "unique-secret-key-for-aksk-filter-test",
		osClient: &gophercloud.ProviderClient{
			TokenID: "unique-issued-token-for-aksk-filter-test",
		},
	}

	config.registerSecrets()

	message := packersdk.LogSecretFilter.FilterString(strings.Join([]string{
		config.Password,
		config.Token,
		config.OTCSecretKey,
		config.osClient.Token(),
	}, " "))
	if strings.Contains(message, config.Password) ||
		strings.Contains(message, config.Token) ||
		strings.Contains(message, config.OTCSecretKey) ||
		strings.Contains(message, config.osClient.Token()) {
		t.Fatalf("expected auth secrets to be filtered, got %q", message)
	}
}

func TestAccessConfigLoadsOTCAKSKFromEnvironment(t *testing.T) {
	t.Setenv("OTC_ACCESS_KEY", "env-access-key")
	t.Setenv("OTC_SECRET_KEY", "env-secret-key")

	config := AccessConfig{}
	config.loadOTCAKSKFromEnv()

	if config.OTCAccessKey != "env-access-key" {
		t.Fatalf("expected access key from environment, got %q", config.OTCAccessKey)
	}
	if config.OTCSecretKey != "env-secret-key" {
		t.Fatalf("expected secret key from environment, got %q", config.OTCSecretKey)
	}
}

func TestAccessConfigKeepsExplicitOTCAKSKValues(t *testing.T) {
	t.Setenv("OTC_ACCESS_KEY", "env-access-key")
	t.Setenv("OTC_SECRET_KEY", "env-secret-key")

	config := AccessConfig{
		OTCAccessKey: "config-access-key",
		OTCSecretKey: "config-secret-key",
	}
	config.loadOTCAKSKFromEnv()

	if config.OTCAccessKey != "config-access-key" {
		t.Fatalf("expected configured access key, got %q", config.OTCAccessKey)
	}
	if config.OTCSecretKey != "config-secret-key" {
		t.Fatalf("expected configured secret key, got %q", config.OTCSecretKey)
	}
}
