# Packer Plugin Openstack

The `Openstack` multi-component plugin can be used with HashiCorp [Packer](https://www.packer.io)
to create custom images. For the full list of available features for this plugin see [docs](docs).

## About this fork

This fork keeps the upstream OpenStack builder mostly intact, but adds
targeted changes for T-Cloud Public, formerly Open Telekom Cloud or OTC.
The goal is to keep OTC-specific behavior opt-in while leaving the default
OpenStack behavior unchanged.

| Change | Why it was added |
| --- | --- |
| OTC image creation method | Adds `image_creation_method = "otc"` to create the final image through OTC IMS instead of the generic Nova/Cinder/Glance flow. The stock flow uses Nova snapshots, Cinder upload-to-image, and Glance follow-up calls. On OTC, that can hit endpoint or behavior differences and may fail or produce images that do not boot cleanly. This addition also includes IMS endpoint handling, Enterprise Project support, OTC tag forwarding, metadata validation, and the required block-storage guard. |

This fork starts at v1.2.0 and is based on the upstream OpenStack plugin. Upstream changes will be merged in as needed.

## Installation

### Using pre-built releases

#### Using the `packer init` command

Starting from version 1.7, Packer supports a new `packer init` command allowing
automatic installation of Packer plugins. Read the
[Packer documentation](https://www.packer.io/docs/commands/init) for more information.

To install this plugin, copy and paste this code into your Packer configuration .
Then, run [`packer init`](https://www.packer.io/docs/commands/init).

```hcl
packer {
  required_plugins {
    openstack-otc = {
      version = ">= 1.1.3"
      source  = "github.com/tobiwo/openstack-otc"
    }
  }
}
```

#### Manual installation

You can find pre-built binary releases of the plugin [here](https://github.com/tobiwo/packer-plugin-openstack-otc/releases).
Once you have downloaded the latest archive corresponding to your target OS,
uncompress it to retrieve the plugin binary file corresponding to your platform.
To install the plugin, please follow the Packer documentation on
[installing a plugin](https://www.packer.io/docs/extending/plugins/#installing-plugins).

### From Sources

If you prefer to build the plugin from sources, clone the GitHub repository
locally and run the command `go build` from the root
directory. Upon successful compilation, a `packer-plugin-openstack-otc` plugin
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
