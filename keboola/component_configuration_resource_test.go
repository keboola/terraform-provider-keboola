package keboola

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/keboola/go-client/pkg/client"
	"github.com/keboola/go-client/pkg/storageapi"
	"github.com/keboola/go-utils/pkg/orderedmap"
)

func exGenericResource(resourceId string, resourceDefinition map[string]any) string {
	result := `	resource "keboola_component_configuration" "` + resourceId + `" {
		component_id = "ex-generic-v2"`
	for attribute, value := range resourceDefinition {
		pair := ""
		switch v := value.(type) {
		case string:
			pair = fmt.Sprintf("%s = %v ", attribute, strconv.Quote(v))
		default:
			pair = fmt.Sprintf("%s = %v ", attribute, v)
		}
		result = result + "\n" + pair
	}
	result = result + `
	` + " }\n"
	return result
}

func checkAllAttributesSet(resourceId string) resource.TestCheckFunc {
	fullResourceId := "keboola_component_configuration." + resourceId
	return resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttrSet(fullResourceId, "id"),
		resource.TestCheckResourceAttrSet(fullResourceId, "configuration_id"),
		resource.TestCheckResourceAttrSet(fullResourceId, "component_id"),
		resource.TestCheckResourceAttrSet(fullResourceId, "branch_id"),
		resource.TestCheckResourceAttrSet(fullResourceId, "name"),
		//resource.TestCheckResourceAttrSet(fullResourceId, "description"),
		resource.TestCheckResourceAttrSet(fullResourceId, "change_description"),
		resource.TestCheckResourceAttrSet(fullResourceId, "is_deleted"),
		resource.TestCheckResourceAttrSet(fullResourceId, "created"),
		resource.TestCheckResourceAttrSet(fullResourceId, "version"),
		resource.TestCheckResourceAttrSet(fullResourceId, "is_disabled"),
		resource.TestCheckResourceAttrSet(fullResourceId, "configuration"),
	)

}

func checkAttribute(attributeName string, actualValue string, expectedValue string) error {
	if actualValue != expectedValue {
		return fmt.Errorf("Stored configuration doesn't match state, attribute: %s \n expected: %s \n actual:%s \n", attributeName, expectedValue, actualValue)
	}
	return nil
}

// loads configuration from host and compares to the terraform state
func testAccCheckExampleConfigMatchesReality(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}

		host := os.Getenv("TEST_KBC_HOST")
		token := os.Getenv("TEST_KBC_TOKEN")

		//fmt.Println(rs.Primary.Attributes)
		attributes := rs.Primary.Attributes
		branchId, err := strconv.Atoi(attributes["branch_id"])
		if err != nil {
			return fmt.Errorf("Could not parse string %s to int: %s", attributes["branch_id"], err)
		}
		key := storageapi.ConfigKey{
			ID:          storageapi.ConfigID(attributes["configuration_id"]),
			BranchID:    storageapi.BranchID(branchId),
			ComponentID: storageapi.ComponentID(attributes["component_id"]),
		}
		sapiClient := storageapi.ClientWithHostAndToken(client.New(), host, token)
		storedConfig, err := storageapi.GetConfigRequest(key).Send(context.Background(), sapiClient)
		if err != nil {
			return fmt.Errorf("Could not load config: %s", err)
		}

		err = checkAttribute("change_description", storedConfig.ChangeDescription, attributes["change_description"])
		if err != nil {
			return err
		}
		err = checkAttribute("description", storedConfig.Description, attributes["description"])
		if err != nil {
			return err
		}
		err = checkAttribute("is_disabled", fmt.Sprintf("%v", storedConfig.IsDisabled), attributes["is_disabled"])
		if err != nil {
			return err
		}
		err = checkAttribute("is_deleted", fmt.Sprintf("%v", storedConfig.IsDeleted), attributes["is_deleted"])
		if err != nil {
			return err
		}
		err = checkAttribute("name", storedConfig.Name, attributes["name"])
		if err != nil {
			return err
		}

		actualContent := storedConfig.Content
		expectedContent := orderedmap.New()
		expectedContentStr := attributes["configuration"]
		err = expectedContent.UnmarshalJSON([]byte(expectedContentStr))
		if err != nil {
			return fmt.Errorf("Could not unmarshal expected configuration to ordered map. Error: %s", err)
		}
		actualBytes, err := actualContent.MarshalJSON()
		if err != nil {
			return fmt.Errorf("Could not marshal actual configuration to string. Error: %s", err)
		}

		if !reflect.DeepEqual(actualContent, expectedContent) {
			return checkAttribute("configuration", string(actualBytes), expectedContentStr)
		}

		return nil
	}
}

func testAccCheckExampleConfigurationDataSet(resourceName string, path string, expectedValue string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}
		configStr := rs.Primary.Attributes["configuration"]
		configMap := orderedmap.New()
		err := configMap.UnmarshalJSON([]byte(configStr))
		if err != nil {
			return fmt.Errorf("Couldn't parse configuration: %s", err)
		}
		path := orderedmap.PathFromStr(path)
		value, found, err := configMap.GetNestedPath(path)
		if err != nil {
			return fmt.Errorf("Get path failed: %s", err)
		}
		if !found {
			return fmt.Errorf("Get path %s not found:", path)
		}
		if value != expectedValue {
			return fmt.Errorf("Get path %s value didn't match: expected: %s found: %s", path, expectedValue, value)
		}
		return nil
	}
}

func TestAccConfigResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// create empty config
			{
				Config: providerConfig + exGenericResource("testempty", map[string]any{
					"name": "test empty config",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAllAttributesSet("testempty"),
					resource.TestCheckResourceAttr("keboola_component_configuration.testempty", "name", "test empty config"),
					resource.TestCheckResourceAttr("keboola_component_configuration.testempty", "is_disabled", "false"),
					testAccCheckExampleConfigMatchesReality("keboola_component_configuration.testempty"),
				),
			},
			// Create and Read testing
			{
				Config: providerConfig + exGenericResource("test", map[string]any{
					"name": "test config",
					"configuration": `{
						"a":1,
						"b":2
					}`,
					"is_disabled": false,
					"description": "description",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAllAttributesSet("test"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test", "is_disabled", "false"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test", "description", "description"),
					testAccCheckExampleConfigMatchesReality("keboola_component_configuration.test"),
				),
			},
			// Update with empty configuration
			{
				Config: providerConfig + exGenericResource("test", map[string]any{
					"name":               "test config",
					"is_disabled":        true,
					"change_description": "new change",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAllAttributesSet("test"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test", "is_disabled", "true"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test", "description", ""),
					resource.TestCheckResourceAttr("keboola_component_configuration.test", "configuration", "{}"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test", "change_description", "new change"),
					testAccCheckExampleConfigMatchesReality("keboola_component_configuration.test"),
				),
			},
			// Update with some configuration
			{
				Config: providerConfig + exGenericResource("test", map[string]any{
					"name": "test config",
					"configuration": `{
						"host": "example.com",
						"port": 123,
						"storage": {
							"input": {
								 "tables": ["in.data1"]
							}
						}
					}`,
				}),
				Check: resource.ComposeTestCheckFunc(
					checkAllAttributesSet("test"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test", "is_disabled", "false"),
					resource.TestCheckResourceAttr("keboola_component_configuration.test", "change_description", "update by Keboola terraform provider"),
					testAccCheckExampleConfigMatchesReality("keboola_component_configuration.test"),
					testAccCheckExampleConfigurationDataSet("keboola_component_configuration.test", "storage.input.tables[0]", "in.data1"),
				),
			},
			// Change configuration id - expects error
			{
				Config: providerConfig + exGenericResource("test", map[string]any{
					"name":             "test config",
					"configuration_id": "aaa",
				}),
				ExpectError: regexp.MustCompile("Can not change configuration_id after configuration is created"),
			},
			// create configuration with id
			{
				Config: providerConfig + exGenericResource("configwithid", map[string]any{
					"name":             "test config with id",
					"configuration_id": "mycustomconfiguid123",
					"configuration":    `{"a":1, "foo": "bar"}`,
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAllAttributesSet("configwithid"),
					resource.TestCheckResourceAttr("keboola_component_configuration.configwithid", "name", "test config with id"),
					resource.TestCheckResourceAttr("keboola_component_configuration.configwithid", "is_disabled", "false"),
					resource.TestCheckResourceAttr("keboola_component_configuration.configwithid", "configuration_id", "mycustomconfiguid123"),
					testAccCheckExampleConfigMatchesReality("keboola_component_configuration.configwithid"),
				),
			},
		},
	})
}
