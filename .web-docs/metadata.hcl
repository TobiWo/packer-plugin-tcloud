# Copyright IBM Corp. 2013, 2026
# SPDX-License-Identifier: MPL-2.0

# For full specification on the configuration of this file visit:
# https://github.com/hashicorp/integration-template#metadata-configuration
integration {
  name = "T-Cloud"
  description = "The T-Cloud Packer plugin can be used with HashiCorp Packer to create custom images for OpenStack-compatible T-Cloud environments."
  identifier = "packer/opentelekomcloud-community/tcloud"
  flags = ["hcp-ready"]
  component {
    type = "builder"
    name = "T-Cloud"
    slug = "tcloud"
  }
}
