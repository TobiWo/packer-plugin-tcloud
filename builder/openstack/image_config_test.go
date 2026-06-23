// Copyright IBM Corp. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package openstack

import (
	"strings"
	"testing"
)

func testImageConfig() *ImageConfig {
	return &ImageConfig{
		ImageName: "foo",
	}
}

func TestImageConfigPrepare_Region(t *testing.T) {
	c := testImageConfig()
	if err := c.Prepare(nil); err != nil {
		t.Fatalf("shouldn't have err: %s", err)
	}

	c.ImageName = ""
	if err := c.Prepare(nil); err == nil {
		t.Fatal("should have error")
	}
}

func TestImageConfigPrepare_ImageCreationMethod(t *testing.T) {
	c := testImageConfig()
	if err := c.Prepare(nil); err != nil {
		t.Fatalf("shouldn't have err: %s", err)
	}
	if c.ImageCreationMethod != imageCreationMethodOpenStack {
		t.Fatalf("expected default image creation method %q, got %q", imageCreationMethodOpenStack, c.ImageCreationMethod)
	}
	if c.ImageMetadata["image_type"] != "image" {
		t.Fatalf("expected default image_type metadata, got %#v", c.ImageMetadata)
	}

	c = testImageConfig()
	c.ImageCreationMethod = imageCreationMethodOTC
	if err := c.Prepare(nil); err != nil {
		t.Fatalf("should accept otc: %s", err)
	}
	if len(c.ImageMetadata) != 0 {
		t.Fatalf("expected no default metadata for otc, got %#v", c.ImageMetadata)
	}

	c = testImageConfig()
	c.ImageCreationMethod = imageCreationMethodOTC
	c.ImageMetadata = map[string]string{"purpose": "test"}
	errs := c.Prepare(nil)
	if len(errs) == 0 || !strings.Contains(errs[0].Error(), "metadata") || !strings.Contains(errs[0].Error(), imageCreationMethodOTC) {
		t.Fatalf("expected otc metadata error, got %v", errs)
	}

	c = testImageConfig()
	c.ImageCreationMethod = "otc_ims"
	errs = c.Prepare(nil)
	if len(errs) == 0 {
		t.Fatal("should reject legacy otc_ims image creation method")
	}

	c = testImageConfig()
	c.ImageCreationMethod = "bad"
	errs = c.Prepare(nil)
	if len(errs) == 0 {
		t.Fatal("should reject unknown image creation method")
	}
}

func TestPrepareImageCreationConfig_OTCIMS(t *testing.T) {
	c := &Config{}
	c.ImageCreationMethod = imageCreationMethodOTC
	c.Region = "eu-de"
	c.TenantID = "project-id"
	if err := c.PrepareImageCreationConfig(); err != nil {
		t.Fatalf("shouldn't have err: %s", err)
	}
	if c.OTCIMSEndpoint != defaultOTCIMSEndpoint("eu-de") {
		t.Fatalf("expected default endpoint, got %q", c.OTCIMSEndpoint)
	}

	c = &Config{}
	c.ImageCreationMethod = imageCreationMethodOTC
	c.UseBlockStorageVolume = true
	if err := c.PrepareImageCreationConfig(); err == nil || !strings.Contains(err.Error(), "use_blockstorage_volume") {
		t.Fatalf("expected use_blockstorage_volume error, got %v", err)
	}

	c = &Config{}
	c.ImageCreationMethod = imageCreationMethodOTC
	c.Region = "eu-de"
	if err := c.PrepareImageCreationConfig(); err == nil || !strings.Contains(err.Error(), "tenant_id") {
		t.Fatalf("expected tenant_id error, got %v", err)
	}
}
