package branch_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/keboola/terraform-provider-keboola/internal/test"
)

func testBranchResource(resourceID string, resourceDefinition map[string]any) string {
	result := `resource "keboola_branch" "` + resourceID + `" {`
	for attribute, value := range resourceDefinition {
		var pair string
		switch v := value.(type) {
		case string:
			pair = fmt.Sprintf("%s = %v ", attribute, strconv.Quote(v))
		default:
			pair = fmt.Sprintf("%s = %v ", attribute, v)
		}
		result = result + "\n" + pair
	}
	result = result + "\n" + "}\n"

	return result
}

func TestAccBranchResource(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: test.AccProtoV6ProviderFactories(),
		PreCheck:                 test.AccPreCheck,
		ExternalProviders: map[string]resource.ExternalProvider{
			"random": {
				Source:            "hashicorp/random",
				VersionConstraint: "3.1.0",
			},
		},
		Steps: []resource.TestStep{
			// Create a branch
			{
				Config: test.ProviderConfig() + testBranchResource("test", map[string]any{
					"name": "test",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keboola_branch.test", "id"),
					resource.TestCheckResourceAttrSet("keboola_branch.test", "name"),
				),
			},
			// Attempt to update the branch
			{
				Config: test.ProviderConfig() + testBranchResource("test", map[string]any{
					"name": "test2",
				}),
				Destroy: true,
			},
		},
	})
}
