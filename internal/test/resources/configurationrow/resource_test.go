// Package configurationrow contains acceptance tests for the keboola_configuration_row resource.
package configurationrow_test

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	test "github.com/keboola/terraform-provider-keboola/internal/test"
)

// buildHCLBlock is a generic helper to generate HCL for a resource block.
// It takes the resource type, resource name, required attributes (as a string),
// and a map of additional attributes.
func buildHCLBlock(resourceType, resourceName, requiredAttrs string, attrs map[string]any) string {
	result := fmt.Sprintf("resource \"%s\" \"%s\" {\n%s", resourceType, resourceName, requiredAttrs)

	// Iterate over attributes and add them to the result string.
	for attribute, value := range attrs {
		var pair string
		switch v := value.(type) {
		case string:
			pair = fmt.Sprintf("%s = %v ", attribute, strconv.Quote(v))
		default:
			pair = fmt.Sprintf("%s = %v ", attribute, v)
		}

		result += "\n" + pair
	}

	// Add an extra newline after the closing brace to ensure valid HCL when concatenating multiple blocks.
	result += "\n}\n"

	return result
}

// buildKeboolaConfigurationRowHCL generates HCL for keboola_configuration_row using FQN.
func buildKeboolaConfigurationRowHCL(resourceID, fqn string, attrs map[string]any) string {
	required := fmt.Sprintf("  fqn = \"%s\"", fqn)

	return buildHCLBlock("keboola_configuration_row", resourceID, required, attrs)
}

// buildKeboolaConfigurationRowHCLWithIDs generates HCL for keboola_configuration_row using explicit IDs.
func buildKeboolaConfigurationRowHCLWithIDs(resourceID, branchID, componentID, configID string, attrs map[string]any) string {
	required := fmt.Sprintf(
		"  branch_id = \"%s\"\n  component_id = \"%s\"\n  configuration_id = \"%s\"",
		branchID, componentID, configID,
	)

	return buildHCLBlock("keboola_configuration_row", resourceID, required, attrs)
}

// buildKeboolaComponentConfigurationHCL generates HCL for keboola_component_configuration.
func buildKeboolaComponentConfigurationHCL(resourceID, componentID string, attrs map[string]any) string {
	required := fmt.Sprintf("  component_id = \"%s\"", componentID)

	return buildHCLBlock("keboola_component_configuration", resourceID, required, attrs)
}

// TestAccConfigurationRowResource_basic tests basic creation of a configuration row using FQN.
func TestAccConfigurationRowResource_basic(t *testing.T) {
	t.Parallel()

	resourceName := "keboola_configuration_row.example_row"
	config := test.ProviderConfig() +
		buildKeboolaComponentConfigurationHCL("ex_generic_test", "ex-generic-v2", map[string]any{
			"name": "Test Config",
			"configuration": `{
						"parameters": {}
					}`,
		}) +
		buildKeboolaConfigurationRowHCL("example_row", "${keboola_component_configuration.ex_generic_test.fqn}", map[string]any{
			"name": "Test Row",
		})

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: test.AccProtoV6ProviderFactories(),
		PreCheck:                 func() { test.AccPreCheck() },
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					// The fqn will have the row ID appended by the API, so you may want to use a regex or prefix check here.
				),
			},
		},
	})
}

// TestAccConfigurationRowResource_CRUD tests create and update of a configuration row using explicit IDs.
func TestAccConfigurationRowResource_CRUD(t *testing.T) { //nolint:paralleltest
	resourceName := "keboola_configuration_row.example_row"
	configBase := map[string]any{
		"name": "Test Config",
	}
	configCreate := test.ProviderConfig() +
		buildKeboolaComponentConfigurationHCL("ex_mysql_test", "keboola.ex-db-mysql", configBase) +
		buildKeboolaConfigurationRowHCLWithIDs(
			"example_row",
			"${keboola_component_configuration.ex_mysql_test.branch_id}",
			"${keboola_component_configuration.ex_mysql_test.component_id}",
			"${keboola_component_configuration.ex_mysql_test.configuration_id}",
			map[string]any{
				"name": "Test Row",
			},
		)

	// Update only the description field in the configuration resource
	// Updating both config and config row is not supported yet
	configUpdate := test.ProviderConfig() +
		buildKeboolaComponentConfigurationHCL("ex_mysql_test", "keboola.ex-db-mysql", map[string]any{
			"name": "Test Config",
		}) +
		buildKeboolaConfigurationRowHCLWithIDs(
			"example_row",
			"${keboola_component_configuration.ex_mysql_test.branch_id}",
			"${keboola_component_configuration.ex_mysql_test.component_id}",
			"${keboola_component_configuration.ex_mysql_test.configuration_id}",
			map[string]any{
				"name":        "Test Row",
				"description": "Updated",
			},
		)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: test.AccProtoV6ProviderFactories(),
		PreCheck:                 func() { test.AccPreCheck() },
		Steps: []resource.TestStep{
			{
				Config: configCreate,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "branch_id"),
					resource.TestCheckResourceAttrSet(resourceName, "component_id"),
					resource.TestCheckResourceAttrSet(resourceName, "configuration_id"),
				),
			},
			{
				Config: configUpdate,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "branch_id"),
					resource.TestCheckResourceAttrSet(resourceName, "component_id"),
					resource.TestCheckResourceAttrSet(resourceName, "configuration_id"),
					// Add checks for updated attributes here
				),
			},
		},
	})
}

// TestAccConfigurationRowResource_missingIdentifiers tests error handling when required identifiers are missing.
func TestAccConfigurationRowResource_missingIdentifiers(t *testing.T) {
	t.Parallel()

	config := test.ProviderConfig() + `
resource "keboola_configuration_row" "invalid_row" {
  name = "Should Fail"
}`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: test.AccProtoV6ProviderFactories(),
		PreCheck:                 func() { test.AccPreCheck() },
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`(?i)Invalid Attribute Combination|No attribute specified when one \(and only one\) of`),
			},
		},
	})
}
