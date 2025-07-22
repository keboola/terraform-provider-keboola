terraform {
  required_providers {
    keboola = {
      source = "keboola/keboola"
    }
  }
}

provider "keboola" {
  hostname_suffix = "north-europe.azure.keboola.com"
  token           = "xxx"
}

//data "keboola_example" "example" {}
