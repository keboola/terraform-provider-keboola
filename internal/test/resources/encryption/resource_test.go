package encryption_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/keboola/terraform-provider-keboola/internal/test"
)

// For running the tests, we'll need a provider factory setup which will be defined in the provider package
// This is a placeholder that should be implemented correctly when running the actual tests

func TestAccEncryptionResource(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: test.AccProtoV6ProviderFactories(),
		PreCheck:                 test.AccPreCheck,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: test.ProviderConfig() + testEncryptionResourceConfig("valuetoencrypt"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keboola_encryption.test", "value", "valuetoencrypt"),
					resource.TestMatchResourceAttr("keboola_encryption.test", "encrypted_value", regexp.MustCompile(`KBC::ProjectSecure.*::.+`)),
					resource.TestCheckResourceAttr("keboola_encryption.test", "id", "none"),
					resource.TestCheckResourceAttr("keboola_encryption.test", "component_id", "ex-generic-v2"),
				),
			},
			// Update and Read testing
			{
				Config: test.ProviderConfig() + testEncryptionResourceConfig(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keboola_encryption.test", "value", ""),
					resource.TestMatchResourceAttr("keboola_encryption.test", "encrypted_value", regexp.MustCompile(`KBC::ProjectSecure.*::.+`)),
					resource.TestCheckResourceAttr("keboola_encryption.test", "id", "none"),
					resource.TestCheckResourceAttr("keboola_encryption.test", "component_id", "ex-generic-v2"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testEncryptionResourceConfig(value string) string {
	return fmt.Sprintf(`
resource "keboola_encryption" "test" {
  value = %[1]q
  component_id = "ex-generic-v2"
}
`, value)
}
