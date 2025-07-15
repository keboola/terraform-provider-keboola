# Manage example configuration.
resource "keboola_component_configuration" "ex_generic_test" {
  name          = "My generic extractor configuration"
  component_id  = "ex-generic-v2"
  description   = "pulls users from my external source"
  is_disabled   = false
  configuration = <<EOT
{
    "parameters": {
        "api": {
            "baseUrl": "http://myexternalresource.com"
        },
        "config": {
            "outputBucket": "output",
            "jobs": [
                {
                    "endpoint": "users",
                    "children": [
                        {
                            "endpoint": "user/{user-id}",
                            "dataField": ".",
                            "placeholders": {
                                "user-id": "id"
                            }
                        }
                    ]
                }
            ]
        }
    }
}
EOT
}

# Example: Create a configuration row using the new resource
resource "keboola_component_configuration_row" "example_row" {
  # You can use either the configuration_fqn or the branch_id/component_id/configuration_id triplet
  configuration_fqn = keboola_component_configuration.ex_generic_test.configuration_fqn

  # Optionally, you can provide additional fields as needed
  # id = "row-id" # Uncomment to provide a custom row ID
  # ... add other attributes as needed ...
}
