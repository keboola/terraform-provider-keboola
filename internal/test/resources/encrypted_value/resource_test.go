package encrypted_value_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/keboola/terraform-provider-keboola/internal/test"
)

// For running the tests, we'll need a provider factory setup which will be defined in the provider package
// This is a placeholder that should be implemented correctly when running the actual tests

func TestAccEncryptedValueResource(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: test.AccProtoV6ProviderFactories(),
		PreCheck:                 test.AccPreCheck,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: test.ProviderConfig() + testEncryptedValueResourceConfig("valuetoencrypt"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keboola_encrypted_value.test", "value", "valuetoencrypt"),
					resource.TestMatchResourceAttr("keboola_encrypted_value.test", "encrypted_value", regexp.MustCompile(`KBC::ProjectSecure.*::.+`)),
					resource.TestCheckResourceAttr("keboola_encrypted_value.test", "id", "none"),
					resource.TestCheckResourceAttr("keboola_encrypted_value.test", "component_id", "ex-generic-v2"),
				),
			},
			// ImportState testing — import ID is the encrypted value itself
			{
				ResourceName:      "keboola_encrypted_value.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["keboola_encrypted_value.test"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return rs.Primary.Attributes["encrypted_value"], nil
				},
				// value/component_id/config_id/branch_type are not restored during import
				// because this resource is stateless and has no Read implementation.
				ImportStateVerifyIgnore: []string{"value", "component_id", "config_id", "branch_type"},
			},
			// Update and Read testing
			{
				Config: test.ProviderConfig() + testEncryptedValueResourceConfig(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keboola_encrypted_value.test", "value", ""),
					resource.TestMatchResourceAttr("keboola_encrypted_value.test", "encrypted_value", regexp.MustCompile(`KBC::ProjectSecure.*::.+`)),
					resource.TestCheckResourceAttr("keboola_encrypted_value.test", "id", "none"),
					resource.TestCheckResourceAttr("keboola_encrypted_value.test", "component_id", "ex-generic-v2"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testEncryptedValueResourceConfig(value string) string {
	return fmt.Sprintf(`
resource "keboola_encrypted_value" "test" {
  value = %[1]q
  component_id = "ex-generic-v2"
}
`, value)
}
