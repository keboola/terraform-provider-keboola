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
