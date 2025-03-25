terraform {
  required_providers {
    keboola = {
      source = "keboola/keboola"
    }
  }
}

provider "keboola" {
  # Configuration can be provided via environment variables:
  # KBC_HOST - Keboola Stack API host
  # KBC_TOKEN - Storage API token
}

# Create a generic extractor configuration
resource "keboola_component_configuration" "telemetry_extractor" {
  name         = "Telemetry Extractor v2"
  component_id = "keboola.ex-aws-s3"
  description  = "Example configuration for collecting telemetry data"

  # Main configuration for the generic extractor
  configuration = jsonencode({
    parameters = {
      api = {
        baseUrl = "https://api.example.com/v1"
        authentication = {
          type = "basic"
        }
      }
      config = {
        debug        = true
        outputBucket = "telemetry-data"
        jobs = [
          {
            endpoint  = "metrics"
            dataField = "data"
            method    = "GET"
          }
        ]
      }
    }
  })
  # Row-specific configuration
  rows = [
    {
      name        = "Daily Metrics Collectiona"
      description = "Collects telemetry metricsa"
      configuration_row = jsonencode({
        parameters = {
          api = {
            query = {
              period = "daily"
              format = "json"
            }
          }
          processor = {
            after = {
              filter_empty = true
            }
          }
        }
      })
    },
    {
      name        = "Test"
      description = "Test"
      configuration_row = jsonencode({
        parameters = {
          api = {
            query = {
              format = "json"
            }
          }
        }
      })
    },
    {
      name        = "Metricsa"
      description = "Collects telemetrya"
      # Row-specific configuration
      configuration_row = jsonencode({
        parameters = {
          api = {
            query = {
              format = "json"
            }
          }
          processor = {
            after = {
              filter_empty = true
            }
          }
        }
      })
    },
  ]
}

