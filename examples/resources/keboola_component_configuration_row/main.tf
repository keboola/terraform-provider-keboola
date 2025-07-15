terraform {
  required_providers {
    keboola = {
      source = "keboola/keboola"
    }
  }
}

provider "keboola" {
  # Optionally set host and token via environment variables
  # host  = "..."
  # token = "..."
}
