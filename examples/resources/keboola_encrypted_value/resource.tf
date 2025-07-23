resource "keboola_encrypted_value" "encryption_test" {
  value        = "valuetoencrypt"
  component_id = "ex-generic-v2"
  # config_id and branch_type are optional and can be set as needed
}

# Example with new optional attributes
resource "keboola_encrypted_value" "encryption_with_options" {
  value        = "anothersecret"
  component_id = "ex-generic-v2"
  config_id    = "123456789"
  branch_type  = "main"
}

resource "keboola_component_configuration" "ex_generic_test_with_encryption" {
  name          = "Extractor configuration with encrypted value"
  component_id  = "ex-generic-v2"
  is_disabled   = false
  configuration = <<EOT
{
    "parameters": {
        "api": {
            "baseUrl": "http://myexternalresource.com",
            "apiKey": "${keboola_encrypted_value.encryption_test.encrypted_value}"
        }
    }
}
EOT
}
