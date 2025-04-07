terraform {
  required_providers {
    keboola = {
      source = "keboola/keboola"
    }
    random = {
      source = "hashicorp/random"
    }
  }
}

provider "keboola" {
  # Configuration can be provided via environment variables:
  # KBC_HOST - Keboola Stack API host
  # KBC_TOKEN - Storage API token
}

resource "random_string" "random" {
  length           = 16
  special          = true
  override_special = "/@£$\\"
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

  ]
}


resource "keboola_component_configuration" "sleep_60s_hourly" {
  name         = "sleep-60-s-hourly"
  component_id = "keboola.orchestrator"
  description  = "Orchestration that runs the sleep-60s application hourly"

  configuration = jsonencode({
    phases = [
      {
        id        = random_string.random.result,
        name      = "Step 1",
        dependsOn = []
      }
    ],
    tasks = [
      {
        id    = random_string.random.result,
        name  = "app-command-60s",
        phase = random_string.random.result,
        task = {
          mode        = "run",
          componentId = "${keboola_component_configuration.telemetry_extractor.component_id}",
          configId    = "${keboola_component_configuration.telemetry_extractor.configuration_id}"
        },
        continueOnFailure = false,
        enabled           = true
      }
    ]
  })
}

resource "keboola_component_configuration" "telemetry_extractor2" {
  name          = "Telemetry Extractor v23"
  component_id  = "keboola.ex-aws-s3"
  description   = "Example configuration for collecting telemetry data"
  configuration = "{\"parameters\":{\"api\":{\"baseUrl\":\"http://myexternalresource.com\"},\"config\":{\"outputBucket\":\"outputs\",\"jobs\":[{\"endpoint\":\"users\",\"children\":[{\"endpoint\":\"user/{user-id}\",\"dataField\":\".\",\"placeholders\":{\"user-id\":\"id\"}}]}]}}}"
  rows = [
    {
      name              = "Test"
      description       = "Test"
      configuration_row = "{\"parameters\":{\"api\":{\"baseUrl\":\"http://myexternalresource.com\"},\"config\":{\"outputBucket\":\"outputs\",\"jobs\":[{\"endpoint\":\"users\",\"children\":[{\"endpoint\":\"user/{user-id}\",\"dataField\":\".\",\"placeholders\":{\"user-id\":\"id\"}}]}]}}}"
    },
    {
      name              = "Test2"
      description       = "Test2"
      configuration_row = "{\"parameters\":{\"api\":{\"baseUrl\":\"http://myexternalresource2.com\"}}}"
    },
  ]
}

resource "keboola_component_configuration" "telemetryScheduler" {
  name         = "Telemetry Scheduler"
  component_id = "keboola.scheduler"
  description  = "Example configuration for telemetry data collection"
  configuration = jsonencode({
    "schedule" : {
      "cronTab" : "*/15 * * * *",
      "timezone" : "UTC",
      "state" : "enabled"
    },
    "target" : {
      "componentId" : "keboola.orchestrator",
      "configurationId" : "${keboola_component_configuration.sleep_60s_hourly.configuration_id}",
      "mode" : "run"
    }
  })
}

resource "keboola_scheduler" "telemetry_scheduler" {
  configuration_id = keboola_component_configuration.telemetryScheduler.configuration_id
}