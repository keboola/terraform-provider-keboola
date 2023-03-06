resource "keboola_encryption" "encryption_test" {
  value        = "valuetoencrypt"
  component_id = "ex-generic-v2"
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
            "apiKey": "${keboola_encryption.encryption_test.encrypted_value}"
        }
    }
}
EOT
}
