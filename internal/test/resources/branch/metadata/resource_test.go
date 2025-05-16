package metadata_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/keboola/terraform-provider-keboola/internal/test"
)

func testResource(name, resourceID string, resourceDefinition map[string]any) string {
	result := `resource "` + name + `" "` + resourceID + `" {`
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

func TestAccBranchMetadataResource(t *testing.T) {
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
				Config: test.ProviderConfig() +
					testResource("keboola_branch", "test1", map[string]any{
						"name": "test1",
					}) +
					testResource("keboola_branch_metadata", "description", map[string]any{
						"branch_id": "${keboola_branch.test1.id}",
						"key":       "description",
						"value":     "Test Branch Description",
					}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keboola_branch.test1", "id"),
					resource.TestCheckResourceAttrSet("keboola_branch.test1", "name"),
					resource.TestCheckResourceAttrSet("keboola_branch_metadata.description", "key"),
					resource.TestCheckResourceAttrSet("keboola_branch_metadata.description", "value"),
				),
			},
			// Attempt to update the branch metadata
			{
				Config: test.ProviderConfig() + testResource("keboola_branch", "test1", map[string]any{
					"name": "test1",
				}) + testResource("keboola_branch_metadata", "description", map[string]any{
					"branch_id": "${keboola_branch.test1.id}",
					"key":       "description",
					"value":     "Test Branch New Description",
				}),
				Destroy: true,
			},
		},
	})
}
