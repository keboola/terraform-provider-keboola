terraform {
  required_providers {
    keboola = {
      source = "keboola/keboola"
    }
  }
}

provider "keboola" {
  # Optionally set hostname_suffix and token via environment variables
  # hostname_suffix = "keboola.com"
  # token = "..."
}
