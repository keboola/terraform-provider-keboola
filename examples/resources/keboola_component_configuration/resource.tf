# Manage example configuration.
resource "keboola_component_configuration" "ex_generic_test" {
  name          = "test ex generic new2"
  component_id  = "ex-generic-v2"
  description   = "testing creation of configuration via terraform"
  configuration = <<EOT
  {
    "number": 3331,
    "somevalue": "foobar",
    "othervalue": "blabla"
  }
  EOT
}
