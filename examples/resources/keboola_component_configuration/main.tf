terraform {
  required_providers {
    keboola = {
      source = "keboola/keboola"
    }
  }
}

provider "keboola" {
  # can be defined via KBC_HOST env
  # host =
  # can be defined via KBC_TOKEN env
  # token =
}
