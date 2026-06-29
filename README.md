# Packer Plugin T-Cloud

The `T-Cloud` plugin can be used with HashiCorp [Packer](https://www.packer.io)
to create custom images. For the full list of available features for this plugin see [docs](docs).

## About this fork

This repo forks the official [Packer OpenStack plugin](https://github.com/hashicorp/packer-plugin-openstack). It keeps the upstream OpenStack builder mostly intact, but adds
targeted changes for T-Cloud Public, formerly Open Telekom Cloud or OTC. The goal is to keep OTC-specific behavior opt-in while leaving the default OpenStack behavior unchanged.

| Change | Why it was added |
| --- | --- |
| OTC image creation method | Adds `image_creation_method = "otc"` to create the final image through OTC IMS instead of the generic Nova/Cinder/Glance flow. The stock flow uses Nova snapshots, Cinder upload-to-image, and Glance follow-up calls. On OTC, that can hit endpoint or behavior differences and may fail or produce images that do not boot cleanly. This addition also includes IMS endpoint handling, Enterprise Project support, OTC tag forwarding, metadata validation, and the required block-storage guard. |
| OTC AK/SK authentication | Adds `otc_access_key` and `otc_secret_key` as an opt-in OTC authentication method. The plugin exchanges the keys for a project-scoped Keystone token through OTC's `hw_ak_sk` identity method, then continues to use normal OpenStack service clients with `X-Auth-Token`. This removes the need for wrapper scripts that fetch a token before running Packer while keeping token, password, application credential, and `clouds.yaml` authentication unchanged. |

This fork starts at v1.2.0 and is based on the upstream OpenStack plugin. Upstream changes will be merged in as needed.

## Installation

### Automatic installation with `packer init`

Starting from version 1.7, Packer supports a new `packer init` command allowing
automatic installation of Packer plugins. Read the
[Packer documentation](https://www.packer.io/docs/commands/init) for more information.

Automatic installation is not available for this plugin yet because it is not
registered for release-based plugin discovery. Use the pre-built release
workflow below for now.

### Using pre-built releases

Download the latest release archive for your platform from the
[releases page](https://github.com/opentelekomcloud-community/packer-plugin-tcloud/releases)
and unzip it in a location of your choice. Then install the extracted plugin
binary into Packer's plugin directory:

```sh
packer plugins install --path <PATH_TO_EXTRACTED_PLUGIN_BINARY> github.com/opentelekomcloud-community/tcloud
```

The `--path` command copies the binary into Packer's plugin directory. The
original extracted path is only needed during installation; the binary does not
need to be on your `PATH`.

Add the plugin reference to your Packer configuration:

```hcl
packer {
  required_plugins {
    tcloud = {
      version = ">= 1.2.0"
      source  = "github.com/opentelekomcloud-community/tcloud"
    }
  }
}
```

Then run [`packer init .`](https://www.packer.io/docs/commands/init) to verify
the installed plugin satisfies the template requirement.

### From Sources

If you prefer to build the plugin from sources, clone the GitHub repository
locally and run `go build -o packer-plugin-tcloud` from the root
directory. Upon successful compilation, a `packer-plugin-tcloud` plugin
binary file can be found in the root directory.
To install the compiled plugin, please follow the official Packer documentation
on [installing a plugin](https://www.packer.io/docs/extending/plugins/#installing-plugins).

### Configuration

For more information on how to configure the plugin, please read the
documentation located in the [`docs/`](docs) directory.

## Contributing

* If you think you've found a bug in the code or you have a question regarding
  the usage of this software, please reach out to us by opening an issue in
  this GitHub repository.
* Contributions to this project are welcome: if you want to add a feature or a
  fix a bug, please do so by opening a Pull Request in this GitHub repository.
  In case of feature contribution, we kindly ask you to open an issue to
  discuss it beforehand.
