terraform {
  required_providers {
    keboola = {
      source = "keboola/keboola"
    }
  }
}

provider "keboola" {
  host  = "https://connection.north-europe.azure.keboola.com"
  token = ""
}

//data "keboola_example" "example" {}
