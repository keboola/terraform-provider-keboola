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
  # TODO: Branch support
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

resource "keboola_component_configuration" "telemetryScheduler" {
  name = "Telemetry Scheduler"
  component_id = "keboola.scheduler"
  description = "Example configuration for telemetry data collection"
  configuration = jsonencode({
    "schedule": {
        "cronTab": "*/15 * * * *",
        "timezone": "UTC",
        "state": "enabled"
    },
    "target": {
        "componentId": "keboola.orchestrator",
        "configurationId": "11183691",
        "mode": "run"
    }
})
}

resource "keboola_scheduler" "telemetry_schedule_activate" {
  config_id = keboola_component_configuration.telemetry_scheduler.configuration_id
}

resource "keboola_component_configuration" "orchestration1" {
  name        = "Orchestration12"
  description = ""
  component_id = "keboola.orchestrator"
  configuration = jsonencode({
    phases = [
      {
        id   = random_string.random.result
        name = "Step 1"
        dependsOn = []
      }
    ],
    tasks = [
      {
        id = random_string.random.result
        name = "keboola.ex-aws-s3-${keboola_component_configuration.telemetry_extractor.configuration_id}"
        phase = random_string.random.result
        task = {
          componentId = "keboola.ex-aws-s3"
          configId = keboola_component_configuration.telemetry_extractor.configuration_id
          mode = "run"
        }
        continueOnFailure = false
        enabled = true
      }
    ]
  })
}

resource "keboola_component_configuration" "telemetry_scheduler" {
  name        = "Telemetry Scheduler"
  description = ""
  component_id = "keboola.scheduler"
  configuration = jsonencode({
    schedule = {
        cronTab = "*/15 * * * *"
        timezone = "UTC"
        state = "enabled"
    }
    target = {
        componentId = "keboola.orchestrator"
        configurationId = keboola_component_configuration.orchestration1.configuration_id
        mode = "run"
    }
  })
}

