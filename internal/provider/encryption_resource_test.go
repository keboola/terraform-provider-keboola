package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccEncryptionResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		//PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testEncryptionResourceConfig("valuetoencrypt"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keboola_encryption.test", "value", "valuetoencrypt"),
					resource.TestMatchResourceAttr("keboola_encryption.test", "encrypted_value", regexp.MustCompile(`KBC::ProjectSecureKV::.+`)),
					resource.TestCheckResourceAttr("keboola_encryption.test", "id", "none"),
					resource.TestCheckResourceAttr("keboola_encryption.test", "component_id", "ex-generic-v2"),
				),
			},
			// Update and Read testing
			{
				Config: providerConfig + testEncryptionResourceConfig(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("keboola_encryption.test", "value", ""),
					resource.TestMatchResourceAttr("keboola_encryption.test", "encrypted_value", regexp.MustCompile(`KBC::ProjectSecureKV::.+`)),
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
