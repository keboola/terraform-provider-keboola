package configuration_test

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/keboola/go-utils/pkg/orderedmap"
	"github.com/keboola/keboola-sdk-go/v2/pkg/keboola"
	"github.com/stretchr/testify/require"

	"github.com/keboola/terraform-provider-keboola/internal/provider_test"
)

// buildKeboolaConfigurationHCL is a helper function to generate HCL for keboola_component_configuration.
func buildKeboolaConfigurationHCL(resourceID, componentID string, resourceDefinition map[string]any) string {
	result := fmt.Sprintf(`resource "keboola_component_configuration" "%s" {
		component_id = "%s"`, resourceID, componentID)
	for attribute, value := range resourceDefinition {
		var pair string
		switch v := value.(type) {
		case string:
			pair = fmt.Sprintf("%s = %v ", attribute, strconv.Quote(v))
		case []map[string]any:
			// Special handling for rows which is a slice of maps
			rowsStr := "rows = ["
			for i, row := range v {
				rowsStr += "\n{"
				for rowKey, rowVal := range row {
					switch rv := rowVal.(type) {
					case string:
						rowsStr += fmt.Sprintf("\n\t%s = %v", rowKey, strconv.Quote(rv))
					default:
						rowsStr += fmt.Sprintf("\n\t%s = %v", rowKey, rv)
					}
				}
				rowsStr += "\n}"
				if i < len(v)-1 {
					rowsStr += ","
				}
			}
			rowsStr += "\n]"
			pair = rowsStr
		default:
			pair = fmt.Sprintf("%s = %v ", attribute, v)
		}
		result = result + "\n" + pair
	}
	result = result + `
	` + " }\n"

	return result
}

func notFoundComponentResource(resourceID string, resourceDefinition map[string]any) string {
	return buildKeboolaConfigurationHCL(resourceID, "not-existing-component-id", resourceDefinition)
}

func exGenericResource(resourceID string, resourceDefinition map[string]any) string {
	return buildKeboolaConfigurationHCL(resourceID, "ex-generic-v2", resourceDefinition)
}

func checkAllAttributesSet(resourceID string) resource.TestCheckFunc {
	fullResourceID := "keboola_component_configuration." + resourceID

	return resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttrSet(fullResourceID, "id"),
		resource.TestCheckResourceAttrSet(fullResourceID, "configuration_id"),
		resource.TestCheckResourceAttrSet(fullResourceID, "component_id"),
		resource.TestCheckResourceAttrSet(fullResourceID, "branch_id"),
		resource.TestCheckResourceAttrSet(fullResourceID, "name"),
		// resource.TestCheckResourceAttrSet(fullResourceID, "description"),
		resource.TestCheckResourceAttrSet(fullResourceID, "change_description"),
		resource.TestCheckResourceAttrSet(fullResourceID, "is_deleted"),
		resource.TestCheckResourceAttrSet(fullResourceID, "created"),
		resource.TestCheckResourceAttrSet(fullResourceID, "is_disabled"),
		resource.TestCheckResourceAttrSet(fullResourceID, "configuration"),
	)
}

func checkAttribute(attributeName, actualValue, expectedValue string) error {
	if actualValue != expectedValue {
		return provider_test.NewAttributeMismatchError(attributeName, expectedValue, actualValue)
	}

	return nil
}

// loads configuration from host and compares to the terraform state.
func testAccCheckExampleConfigMatchesReality(t *testing.T, resourceName string) resource.TestCheckFunc {
	t.Helper()

	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return provider_test.NewResourceNotFoundError(resourceName)
		}

		host := os.Getenv("TEST_KBC_HOST")   //nolint: forbidigo
		token := os.Getenv("TEST_KBC_TOKEN") //nolint: forbidigo

		attributes := rs.Primary.Attributes
		branchID, err := strconv.Atoi(attributes["branch_id"])
		if err != nil {
			return fmt.Errorf("Could not parse string %s to int: %w", attributes["branch_id"], err)
		}
		key := keboola.ConfigKey{
			ID:          keboola.ConfigID(attributes["configuration_id"]),
			BranchID:    keboola.BranchID(branchID),
			ComponentID: keboola.ComponentID(attributes["component_id"]),
		}
		ctx := t.Context()
		sapiClient, err := keboola.NewAuthorizedAPI(ctx, host, token)
		require.NoError(t, err)

		storedConfig, err := sapiClient.GetConfigRequest(key).Send(ctx)
		require.NoError(t, err)

		// Get rows separately
		rowKey := keboola.ConfigRowKey{
			ConfigID:    key.ID,
			BranchID:    key.BranchID,
			ComponentID: key.ComponentID,
		}

		rows, err := sapiClient.ListConfigRowRequest(rowKey).Send(ctx)
		require.NoError(t, err)

		// Combine into ConfigWithRows
		storedConfigWithRows := &keboola.ConfigWithRows{
			Config: storedConfig,
			Rows:   *rows,
		}

		err = checkAttribute("change_description", storedConfigWithRows.ChangeDescription, attributes["change_description"])
		if err != nil {
			return err
		}
		err = checkAttribute("description", storedConfigWithRows.Description, attributes["description"])
		if err != nil {
			return err
		}
		err = checkAttribute("is_disabled", strconv.FormatBool(storedConfigWithRows.IsDisabled), attributes["is_disabled"])
		if err != nil {
			return err
		}
		err = checkAttribute("is_deleted", strconv.FormatBool(storedConfigWithRows.IsDeleted), attributes["is_deleted"])
		if err != nil {
			return err
		}
		err = checkAttribute("name", storedConfigWithRows.Name, attributes["name"])
		if err != nil {
			return err
		}

		actualContent := storedConfigWithRows.Content
		expectedContent := orderedmap.New()
		expectedContentStr := attributes["configuration"]
		err = expectedContent.UnmarshalJSON([]byte(expectedContentStr))
		if err != nil {
			return provider_test.NewConfigParseError(err)
		}
		actualBytes, err := actualContent.MarshalJSON()
		if err != nil {
			return fmt.Errorf("Could not marshal actual configuration to string. Error: %w", err)
		}

		if !reflect.DeepEqual(actualContent, expectedContent) {
			return checkAttribute("configuration", string(actualBytes), expectedContentStr)
		}

		return nil
	}
}

func testAccCheckExampleConfigurationDataSet(resourceName, path, expectedValue string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return provider_test.NewResourceNotFoundError(resourceName)
		}
		configStr := rs.Primary.Attributes["configuration"]
		configMap := orderedmap.New()
		err := configMap.UnmarshalJSON([]byte(configStr))
		if err != nil {
			return provider_test.NewConfigParseError(err)
		}
		path := orderedmap.PathFromStr(path)
		value, found, err := configMap.GetNestedPath(path)
		if err != nil {
			return fmt.Errorf("Get path failed: %w", err)
		}
		if !found {
			return provider_test.NewPathNotFoundError(path.String())
		}
		if value != expectedValue {
			return provider_test.NewPathValueMismatchError(path.String(), expectedValue, value)
		}

		return nil
	}
}

func TestAccConfigResource(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provider_test.TestAccProtoV6ProviderFactories(),
		PreCheck:                 provider_test.TestAccPreCheck,
		Steps: []resource.TestStep{
			// create empty config
			{
				Config: provider_test.ProviderConfig() + notFoundComponentResource("testnonexistingcomponent", map[string]any{
					"name": "test nonexisting component config",
				}),
				ExpectError: regexp.MustCompile("Invalid Component ID"),
			},
			// create empty config
			{
				Config: provider_test.ProviderConfig() + exGenericResource("testempty", map[string]any{
					"name": "test empty config",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAllAttributesSet("testempty"),
					resource.TestCheckResourceAttr("keboola_component_configuration.testempty", "name", "test empty config"),
					resource.TestMatchResourceAttr("keboola_component_configuration.testempty", "id", regexp.MustCompile(`\d+/ex-generic-v2/\d+`)),
					resource.TestCheckResourceAttr("keboola_component_configuration.testempty", "is_disabled", "false"),
					testAccCheckExampleConfigMatchesReality(t, "keboola_component_configuration.testempty"),
				),
			},
			// Create and Read testing
			{
				Config: provider_test.ProviderConfig() + exGenericResource("test", map[string]any{
					"name": "test config",
					"configuration": `{
						"a": 1,
						"b": 2
					}`,
					"is_disabled": false,
					"description": "description",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAllAttributesSet("test"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test", "is_disabled", "false"),
					resource.TestMatchResourceAttr("keboola_component_configuration.test", "id", regexp.MustCompile(`\d+/ex-generic-v2/\d+`)),
					resource.TestCheckResourceAttr("keboola_component_configuration.test", "description", "description"),
					testAccCheckExampleConfigMatchesReality(t, "keboola_component_configuration.test"),
				),
			},
			// Update with empty configuration
			{
				Config: provider_test.ProviderConfig() + exGenericResource("test", map[string]any{
					"name":               "test config",
					"is_disabled":        true,
					"change_description": "new change",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAllAttributesSet("test"),
					resource.TestMatchResourceAttr("keboola_component_configuration.test", "id", regexp.MustCompile(`\d+/ex-generic-v2/\d+`)),
					resource.TestCheckResourceAttr("keboola_component_configuration.test", "is_disabled", "true"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test", "description", ""),
					resource.TestCheckResourceAttr("keboola_component_configuration.test", "configuration", "{}"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test", "change_description", "new change"),
					testAccCheckExampleConfigMatchesReality(t, "keboola_component_configuration.test"),
				),
			},
			// Update with some configuration
			{
				Config: provider_test.ProviderConfig() + exGenericResource("test", map[string]any{
					"name":               "test config",
					"change_description": "update by Keboola terraform provider",
					"configuration": `{
						"host": "example.com",
						"port": 123,
						"storage": {
							"input": {
								"tables": [
									"in.data1"
								]
							}
						}
					}`,
				}),
				Check: resource.ComposeTestCheckFunc(
					checkAllAttributesSet("test"),
					resource.TestMatchResourceAttr("keboola_component_configuration.test", "id", regexp.MustCompile(`\d+/ex-generic-v2/\d+`)),
					resource.TestCheckResourceAttr("keboola_component_configuration.test", "is_disabled", "true"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test", "change_description", "update by Keboola terraform provider"),
					testAccCheckExampleConfigMatchesReality(t, "keboola_component_configuration.test"),
					testAccCheckExampleConfigurationDataSet("keboola_component_configuration.test", "storage.input.tables[0]", "in.data1"),
				),
			},
			// Change configuration id - expects error
			{
				Config: provider_test.ProviderConfig() + exGenericResource("test", map[string]any{
					"name":             "test config",
					"configuration_id": "aaa",
				}),
				ExpectError: regexp.MustCompile("Cannot change configuration_id after configuration is created"),
			},
			// create configuration with id
			{
				Config: provider_test.ProviderConfig() + exGenericResource("configwithid", map[string]any{
					"name":             "test config with id",
					"configuration_id": "mycustomconfiguid123",
					"configuration": `{
						"a": 1,
						"foo": "bar"
					}`,
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAllAttributesSet("configwithid"),
					resource.TestMatchResourceAttr("keboola_component_configuration.configwithid", "id", regexp.MustCompile(`\d+/ex-generic-v2/mycustomconfiguid123`)),
					resource.TestCheckResourceAttr("keboola_component_configuration.configwithid", "name", "test config with id"),
					resource.TestCheckResourceAttr("keboola_component_configuration.configwithid", "is_disabled", "false"),
					resource.TestCheckResourceAttr("keboola_component_configuration.configwithid", "configuration_id", "mycustomconfiguid123"),
					testAccCheckExampleConfigMatchesReality(t, "keboola_component_configuration.configwithid"),
				),
			},
		},
	})
}

func TestAccConfigRowsCRUD(t *testing.T) { //nolint: paralleltest
	// t.Parallel()
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provider_test.TestAccProtoV6ProviderFactories(),
		PreCheck:                 provider_test.TestAccPreCheck,
		Steps: []resource.TestStep{
			// Create configuration with initial rows
			{
				Config: provider_test.ProviderConfig() + exGenericResource("test_rows", map[string]any{
					"name": "test config with rows",
					"configuration": `{
						"parameters": {
							"api": {
								"baseUrl": "https://api.example.com"
							}
						}
					}`,
					"rows": []map[string]any{
						{
							"name":        "First Row",
							"description": "Test row 1",
							"is_disabled": false,
							"configuration_row": `{
								"parameters": {
									"api": {
										"endpoint": "endpoint1"
									}
								}
							}`,
						},
						{
							"name":        "Second Row",
							"description": "Test row 2",
							"is_disabled": true,
							"configuration_row": `{
								"parameters": {
									"api": {
										"endpoint": "endpoint2"
									}
								}
							}`,
						},
					},
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAllAttributesSet("test_rows"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test_rows", "name", "test config with rows"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test_rows", "change_description", "Created by Keboola Terraform Provider"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test_rows", "rows.#", "2"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test_rows", "rows.0.name", "First Row"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test_rows", "rows.0.is_disabled", "false"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test_rows", "rows.1.name", "Second Row"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test_rows", "rows.1.is_disabled", "true"),
					testAccCheckExampleConfigMatchesReality(t, "keboola_component_configuration.test_rows"),
				),
			},
			// Update existing rows
			{
				Config: provider_test.ProviderConfig() + exGenericResource("test_rows", map[string]any{
					"name": "test config with rows",
					"configuration": `{
						"parameters": {
							"api": {
								"baseUrl": "https://api.example.com"
							}
						}
					}`,
					"rows": []map[string]any{
						{
							"name":        "First Row Updated",
							"description": "Test row 1 updated",
							"is_disabled": true,
							"configuration_row": `{
								"parameters": {
									"api": {
										"endpoint": "endpoint1-updated"
									}
								}
							}`,
						},
						{
							"name":        "Second Row",
							"description": "Test row 2",
							"is_disabled": true,
							"configuration_row": `{
								"parameters": {
									"api": {
										"endpoint": "endpoint2"
									}
								}
							}`,
						},
					},
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAllAttributesSet("test_rows"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test_rows", "change_description", "Updated by Keboola Terraform Provider"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test_rows", "rows.#", "2"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test_rows", "rows.0.name", "First Row Updated"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test_rows", "rows.0.is_disabled", "true"),
					testAccCheckExampleConfigMatchesReality(t, "keboola_component_configuration.test_rows"),
				),
			},
			// Add a new row
			{
				Config: provider_test.ProviderConfig() + exGenericResource("test_rows", map[string]any{
					"name": "test config with rows",
					"configuration": `{
						"parameters": {
							"api": {
								"baseUrl": "https://api.example.com"
							}
						}
					}`,
					"rows": []map[string]any{
						{
							"name":        "First Row Updated",
							"description": "Test row 1 updated",
							"is_disabled": true,
							"configuration_row": `{
								"parameters": {
									"api": {
										"endpoint": "endpoint1-updated"
									}
								}
							}`,
						},
						{
							"name":        "Second Row",
							"description": "Test row 2",
							"is_disabled": true,
							"configuration_row": `{
								"parameters": {
									"api": {
										"endpoint": "endpoint2"
									}
								}
							}`,
						},
						{
							"name":        "Third Row",
							"description": "Test row 3",
							"is_disabled": false,
							"configuration_row": `{
								"parameters": {
									"api": {
										"endpoint": "endpoint3"
									}
								}
							}`,
						},
					},
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAllAttributesSet("test_rows"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test_rows", "change_description", "Updated by Keboola Terraform Provider"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test_rows", "rows.#", "3"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test_rows", "rows.2.name", "Third Row"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test_rows", "rows.2.is_disabled", "false"),
					testAccCheckExampleConfigMatchesReality(t, "keboola_component_configuration.test_rows"),
				),
			},
			// Remove rows
			{
				Config: provider_test.ProviderConfig() + exGenericResource("test_rows", map[string]any{
					"name": "test config with rows",
					"configuration": `{
						"parameters": {
							"api": {
								"baseUrl": "https://api.example.com"
							}
						}
					}`,
					"rows": []map[string]any{
						{
							"name":        "First Row Updated",
							"description": "Test row 1 updated",
							"is_disabled": true,
							"configuration_row": `{
								"parameters": {
									"api": {
										"endpoint": "endpoint1-updated"
									}
								}
							}`,
						},
					},
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAllAttributesSet("test_rows"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test_rows", "change_description", "Updated by Keboola Terraform Provider"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test_rows", "rows.#", "1"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test_rows", "rows.0.name", "First Row Updated"),
					testAccCheckExampleConfigMatchesReality(t, "keboola_component_configuration.test_rows"),
				),
			},
		},
	})
}
