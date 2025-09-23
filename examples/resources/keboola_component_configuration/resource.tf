# Manage example configuration.
resource "keboola_component_configuration" "ex_generic_test" {
  name          = "Google BigQuery Data Extractor"
  component_id  = "keboola.ex-google-bigquery-v2"
  description   = "Extracts data from Google BigQuery tables and datasets"
  is_disabled   = false
  configuration = jsonencode({
    parameters = {
        google = {
            storage = "save"
            location = "us-west4"
        }
        service_account = {
            project_id = "keboola-test-398718"
            client_email = "keboola-test-398718@appspot.gserviceaccount.com"
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
              tableId = "testtable"
              datasetId = ""
            }
          }
        }
      })
    },
  ]
}


