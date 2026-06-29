// Copyright IBM Corp. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

//go:generate packer-sdc struct-markdown

package openstack

import (
	"fmt"
	"strings"

	imageservice "github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

const (
	imageCreationMethodOpenStack = "openstack"
	imageCreationMethodOTC       = "otc"
)

// ImageConfig is for common configuration related to creating Images.
type ImageConfig struct {
	// The name of the resulting image.
	ImageName string `mapstructure:"image_name" required:"true"`
	// Glance metadata applied to OpenStack-created images. Not supported with
	// image_creation_method "otc".
	ImageMetadata map[string]string `mapstructure:"metadata" required:"false"`
	// One of "public", "private", "shared", or "community".
	ImageVisibility imageservice.ImageVisibility `mapstructure:"image_visibility" required:"false"`
	// List of members to add to the image after creation. An image member is
	// usually a project (also called the "tenant") with whom the image is
	// shared.
	ImageMembers []string `mapstructure:"image_members" required:"false"`
	// When true, perform the image accept so the members can see the image in their
	// project. This requires a user with privileges both in the build project and
	// in the members provided. Defaults to false.
	ImageAutoAcceptMembers bool `mapstructure:"image_auto_accept_members" required:"false"`
	// Disk format of the resulting image. This option works if
	// use_blockstorage_volume is true.
	ImageDiskFormat string `mapstructure:"image_disk_format" required:"false"`
	// List of tags to add to the image after creation.
	ImageTags []string `mapstructure:"image_tags" required:"false"`
	// Minimum disk size needed to boot image, in gigabytes.
	ImageMinDisk int `mapstructure:"image_min_disk" required:"false"`
	// Image creation method. Defaults to "openstack". Set to "otc" to create
	// an OTC/T-Cloud Public IMS system disk image from the stopped build ECS.
	ImageCreationMethod string `mapstructure:"image_creation_method" required:"false"`
	// OTC IMS API endpoint. Defaults to https://ims.&lt;region&gt;.otc.t-systems.com
	// when omitted.
	OTCIMSEndpoint string `mapstructure:"otc_ims_endpoint" required:"false"`
	// OTC enterprise project ID assigned to the IMS image. Omitted when empty.
	OTCEnterpriseProjectID string `mapstructure:"otc_enterprise_project_id" required:"false"`
	// Skip creating the image. Useful for setting to `true` during a build test stage. Defaults to `false`.
	SkipCreateImage bool `mapstructure:"skip_create_image" required:"false"`
}

func (c *ImageConfig) Prepare(ctx *interpolate.Context) []error {
	errs := make([]error, 0)
	if c.ImageName == "" {
		errs = append(errs, fmt.Errorf("An image_name must be specified"))
	}
	if c.ImageCreationMethod == "" {
		c.ImageCreationMethod = imageCreationMethodOpenStack
	}
	if c.ImageCreationMethod != imageCreationMethodOpenStack && c.ImageCreationMethod != imageCreationMethodOTC {
		errs = append(errs, fmt.Errorf("Unknown image_creation_method value %s", c.ImageCreationMethod))
	}
	if c.ImageCreationMethod == imageCreationMethodOTC && len(c.ImageMetadata) > 0 {
		errs = append(errs, fmt.Errorf("metadata is not supported when image_creation_method is %q", imageCreationMethodOTC))
	}

	// By default, OpenStack seems to create the image with an image_type of
	// "snapshot", since it came from snapshotting a VM. A "snapshot" looks
	// slightly different in the OpenStack UI and OpenStack won't show
	// "snapshot" images as a choice in the list of images to boot from for a
	// new instance. See https://github.com/hashicorp/packer/issues/3038
	if c.ImageCreationMethod == imageCreationMethodOpenStack {
		if c.ImageMetadata == nil {
			c.ImageMetadata = map[string]string{"image_type": "image"}
		} else if c.ImageMetadata["image_type"] == "" {
			c.ImageMetadata["image_type"] = "image"
		}
	}

	// ImageVisibility values
	// https://wiki.openstack.org/wiki/Glance-v2-community-image-visibility-design
	if c.ImageVisibility != "" {
		validVals := []imageservice.ImageVisibility{"public", "private", "shared", "community"}
		valid := false
		for _, val := range validVals {
			if strings.EqualFold(string(c.ImageVisibility), string(val)) {
				valid = true
				c.ImageVisibility = val
				break
			}
		}
		if !valid {
			errs = append(errs, fmt.Errorf("Unknown visibility value %s", c.ImageVisibility))
		}
	}

	if c.ImageMinDisk < 0 {
		errs = append(errs, fmt.Errorf("An image min disk size must be greater than or equal to 0"))
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func defaultOTCIMSEndpoint(region string) string {
	return fmt.Sprintf("https://ims.%s.otc.t-systems.com", region)
}

func (c *Config) PrepareImageCreationConfig() error {
	if c.ImageCreationMethod != imageCreationMethodOTC {
		return nil
	}

	if c.UseBlockStorageVolume {
		return fmt.Errorf("image_creation_method %q requires use_blockstorage_volume to be false", imageCreationMethodOTC)
	}

	if c.TenantID == "" {
		return fmt.Errorf("tenant_id must be specified when image_creation_method is %q because IMS job polling requires the OpenStack project ID", imageCreationMethodOTC)
	}

	if c.OTCIMSEndpoint == "" {
		if c.Region == "" {
			return fmt.Errorf("region or otc_ims_endpoint must be specified when image_creation_method is %q", imageCreationMethodOTC)
		}
		c.OTCIMSEndpoint = defaultOTCIMSEndpoint(c.Region)
	}

	return nil
}
