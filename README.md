# Terraform Provider Keboola

Implementation of a Terraform provider managing resources of [Keboola Storage API](https://keboola.docs.apiary.io/).
The provider is built with [**Terraform Plugin Framework**](https://developer.hashicorp.com/terraform/plugin/framework).
The provider repository: https://registry.terraform.io/providers/keboola/keboola


## Usage Example
```hcl
# 1. Specify the version of the Keboola Provider to use
terraform {
  required_providers {
    keboola = {
      source = "keboola/keboola"
      version = "x.x.x"
    }
  }
}

# 2. Configure the Keboola Provider
provider "keboola" {
  host = "https://connection.keboola.com"
  # Token can be specified via env KBC_TOKEN
  # token = ""
}

# 3. Create a resource group
resource "keboola_component_configuration" "example" {
  name          = "My extractor configuration"
  component_id  = "ex-generic-v2"
  description   = "extracts data from my external source"
  configuration = <<EOT
  {
    "somevalue": "foo",
    "othervalue": "bar"
  }
  EOT
}
```
* Additional examples can be found in the [`./examples`](./examples/) folder within this repository.

## Developing & Contributing to the Provider

### Learn about plugin development

#### Must read
- Design principles - https://developer.hashicorp.com/terraform/plugin/hashicorp-provider-design-principles
- Custom provider development tutorial (especially all the "Implement *" articles) - https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework

#### Worth read
- General docs on plugin developmnet - https://developer.hashicorp.com/terraform/plugin
- Terraform plugin framework docs - https://developer.hashicorp.com/terraform/plugin/framework
- Example plugin code - https://github.com/hashicorp/terraform-provider-scaffolding-framework

### Setup Environment

#### Install developer tools
* [Terraform (Core)](https://www.terraform.io/downloads.html) - version v1.0.3 and later
* [Go](https://golang.org/doc/install) version 1.18.x and later (to build the provider plugin)
#### Install Keboola terraform provider locally
Create `.terraformrc` file and place it into your home directory (see [docs](https://developer.hashicorp.com/terraform/tutorials/providers-plugin-framework/providers-plugin-framework-provider#prepare-terraform-for-local-provider-install)) as follows:

```hcl
provider_installation {

  # Use /home/developer/go/bin (or value of "go env GOBIN" command) as an overridden package directory
  # for the keboola/keboola provider. This disables the version and checksum
  # verifications for this provider and forces Terraform to look for the
  # keboola provider plugin in the given directory.
  dev_overrides {
    "registry.terraform.io/keboola/keboola" = "<FILL_GOBIN_PATH>"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```

Verfiy the correct setup by calling **`make test-install`** - this should succesfully finish with `Warning: Provider development overrides are in effect`.

#### Setup test envs for local developmment
Define envs `TEST_KBC_HOST` and `TEST_KBC_TOKEN` and run **`make testacc`** - this should succesfully run acceptance tests.
Alternatively you can run Terraform CLI commands (terraform plan, terraform apply) on the terraform files with "keboola/keboola" provider resources (e.g. see `examples` directory).

### Develop
To develop new resource follow these stops:
- Create sample example resource in the `examples` directory.
- Implement resource or data_source in the `keboola` directory.
- Implement acceptance tests.
- Generate docs through `make generate-docs` or `go generate ./...` command.
- Create a pull request and have it reviewed and released.
