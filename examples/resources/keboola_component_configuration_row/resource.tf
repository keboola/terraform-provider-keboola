# This example demonstrates how to define a configuration row for a table in the SQL extractor.
# - The parent configuration must exist and be referenced via configuration_fqn.
# - Adjust parameters as needed for your use case. 

# Example: Create a component configuration for an SQL extractor
resource "keboola_component_configuration" "extractor_sql" {
  name         = "SQL Extractor Example"
  component_id = "keboola.ex-db-snowflake"
  description  = "Example configuration for SQL extractor."
  configuration = jsonencode({
    parameters = {
      # Add extractor-specific parameters here
    }
  })
}

# Example: Create a configuration row for the SQL extractor
resource "keboola_component_configuration_row" "extractor_sql_opportunity" {
  # Reference the parent configuration using configuration_fqn
  configuration_fqn = keboola_component_configuration.extractor_sql.fqn
  name              = "OPPORTUNITY"
  configuration = jsonencode({
    parameters = {
      columns     = []
      primaryKey  = []
      incremental = false
      outputTable = "in.c-keboola-ex-db-snowflake-${keboola_component_configuration.extractor_sql.id}.opportunity"
      table = {
        schema    = "HELP_TUTORIAL"
        tableName = "OPPORTUNITY"
      }
    }
  })
}
