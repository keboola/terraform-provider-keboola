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

const hclBlockEnd = `
}
`

// buildKeboolaConfigurationRowHCL is a helper function to generate HCL for keboola_configuration_row using FQN.
func buildKeboolaConfigurationRowHCL(resourceID, fqn string, resourceDefinition map[string]any) string {
	result := fmt.Sprintf(`resource "keboola_configuration_row" "%s" {
  fqn = "%s"`, resourceID, fqn)

	for attribute, value := range resourceDefinition {
		var pair string
		switch v := value.(type) {
		case string:
			pair = fmt.Sprintf("%s = %v ", attribute, strconv.Quote(v))
		default:
			pair = fmt.Sprintf("%s = %v ", attribute, v)
		}

		result += "\n" + pair
	}

	result += hclBlockEnd

	return result
}

// buildKeboolaConfigurationRowHCLWithIDs is a helper function to generate HCL for keboola_configuration_row using explicit IDs.
func buildKeboolaConfigurationRowHCLWithIDs(resourceID, branchID, componentID, configID string, resourceDefinition map[string]any) string {
	result := fmt.Sprintf(`resource "keboola_configuration_row" "%s" {
  branch_id = "%s"
  component_id = "%s"
  configuration_id = "%s"`, resourceID, branchID, componentID, configID)

	for attribute, value := range resourceDefinition {
		var pair string
		switch v := value.(type) {
		case string:
			pair = fmt.Sprintf("%s = %v ", attribute, strconv.Quote(v))
		default:
			pair = fmt.Sprintf("%s = %v ", attribute, v)
		}

		result += "\n" + pair
	}

	result += hclBlockEnd

	return result
}

// buildKeboolaComponentConfigurationHCL is a helper function to generate HCL for keboola_component_configuration.
func buildKeboolaComponentConfigurationHCL(resourceID, componentID string, resourceDefinition map[string]any) string {
	result := fmt.Sprintf(`resource "keboola_component_configuration" "%s" {
  component_id = "%s"`, resourceID, componentID)

	for attribute, value := range resourceDefinition {
		var pair string
		switch v := value.(type) {
		case string:
			pair = fmt.Sprintf("%s = %v ", attribute, strconv.Quote(v))
		default:
			pair = fmt.Sprintf("%s = %v ", attribute, v)
		}

		result += "\n" + pair
	}

	result += hclBlockEnd

	return result
}

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

	// nolint:godox // TODO: updating both config and config row is not supported yet
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
