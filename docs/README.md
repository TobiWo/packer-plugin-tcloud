The Openstack Packer plugin provides a builder that is able to create new images
for use with OpenStack. The builder takes a source image, runs any provisioning
necessary on the image after launching it, then creates a new reusable image.
This reusable image can then be used as the foundation of new servers that are
launched within OpenStack. The builder will create temporary keypairs that
provide temporary access to the server while the image is being created. This
simplifies configuration quite a bit.

###  Installation

To install this plugin, copy and paste this code into your Packer configuration .
Then, run [`packer init`](https://www.packer.io/docs/commands/init).

```hcl
packer {
  required_plugins {
    openstack-otc = {
      version = "~> 1"
      source  = "github.com/tobiwo/openstack-otc"
    }
  }
}
```

Alternatively, you can use `packer plugins install` to manage installation of this plugin.

```sh
$ packer plugins install github.com/tobiwo/openstack-otc
```

### Components

#### Builder

- [builder](/packer/integrations/tobiwo/openstack-otc/latest/components/builder/openstack-otc) - The OpenStack Packer builder is able to create new images for use with OpenStack.
