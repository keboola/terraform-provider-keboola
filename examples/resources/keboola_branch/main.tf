terraform {
  required_providers {
    keboola = {
      source = "keboola/keboola"
    }
  }
}

provider "keboola" {
  # can be defined via KBC_HOSTNAME_SUFFIX env
  # hostname_suffix = "keboola.com"
  # can be defined via KBC_TOKEN env
  # token =
}
