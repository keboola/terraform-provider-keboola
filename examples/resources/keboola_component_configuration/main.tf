terraform {
  required_providers {
    keboola = {
      source = "keboola/keboola"
    }
  }
}

provider "keboola" {
  hostname_suffix = var.hostname_suffix
  token           = var.token
}

variable "hostname_suffix" {
  type = string
}

variable "token" {
  type = string
}