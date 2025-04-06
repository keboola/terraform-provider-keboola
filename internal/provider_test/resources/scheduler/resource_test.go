package scheduler_test

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/keboola/terraform-provider-keboola/internal/provider_test"
)

func testSchedulerResource(resourceID string, resourceDefinition map[string]any) string {
	result := `resource "keboola_scheduler" "` + resourceID + `" {`
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

// testConfigurationResource creates a generic component configuration for testing.
func testConfigurationResource(resourceID string, resourceDefinition map[string]any) string {
	result := `resource "keboola_component_configuration" "` + resourceID + `" {
	component_id = "ex-generic-v2"`

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

func TestAccSchedulerResource(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provider_test.TestAccProtoV6ProviderFactories(),
		PreCheck:                 provider_test.TestAccPreCheck,
		Steps: []resource.TestStep{
			// Create a configuration and a scheduler for it
			{
				// First create a configuration to use with the scheduler
				Config: provider_test.ProviderConfig() + testConfigurationResource("telemetry_extractor", map[string]any{
					"name": "Telemetry Extractor",
				}) + testConfigurationResource("test_config_orchestrator", map[string]any{
					"name": "Test Configuration for Orchestrator",
					"configuration": `{
						"phases": [
							{
								"id": "123141",
								"name": "Step 1",
								"dependsOn": []
							}
						],
						"tasks": [
							{
								"id": "141241",
								"name": "ex-generic-v2-${keboola_component_configuration.telemetry_extractor.configuration_id}",
								"phase": "123141",
								"task": {
									"componentId": "ex-generic-v2",
									"configId": "${keboola_component_configuration.telemetry_extractor.configuration_id}",
									"mode": "run"
								},
								"continueOnFailure": false,
								"enabled": true
							}
						]
					}`,
					// Then create a scheduler using the configuration
				}) + testConfigurationResource("test_config_scheduler", map[string]any{
					"name": "Test Configuration for Scheduler",
					"configuration": `{
						"schedule": {
							"cronTab": "*/15 * * * *",
							"timezone": "UTC",
							"state": "enabled"
						},
						"target": {
							"componentId": "keboola.orchestrator",
							"configurationId": "${keboola_component_configuration.test_config_orchestrator.configuration_id}",
							"mode": "run"
						}
					}`,
					// Then create a scheduler using the configuration
				}) + testSchedulerResource("test", map[string]any{
					"config_id": "${keboola_component_configuration.test_config_scheduler.configuration_id}",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keboola_scheduler.test", "id"),
					resource.TestCheckResourceAttrSet("keboola_scheduler.test", "schedule_id"),
					resource.TestCheckResourceAttr("keboola_scheduler.test", "name", "Test Scheduler"),
					resource.TestCheckResourceAttr("keboola_scheduler.test", "cron_expression", "0 0 * * *"),
					resource.TestCheckResourceAttr("keboola_scheduler.test", "active", "true"),
				),
			},
			// Update the scheduler
			{
				Config: provider_test.ProviderConfig() +
					// Update the scheduler
					testSchedulerResource("test", map[string]any{
						"config_id": "${keboola_component_configuration.test_config_scheduler.configuration_id}",
					}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keboola_scheduler.test", "id"),
					resource.TestCheckResourceAttr("keboola_scheduler.test", "active", "false"),
					resource.TestCheckResourceAttr("keboola_scheduler.test", "cron_expression", "0 12 * * *"),
					resource.TestCheckResourceAttr("keboola_scheduler.test", "timezone_id", "America/New_York"),
					resource.TestCheckResourceAttr("keboola_scheduler.test", "version_dependent", "true"),
				),
			},
			// Import test
			{
				ResourceName:            "keboola_scheduler.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"name", "description", "timezone_id", "cron_expression"},
			},
		},
	})
}

func TestSchedulerCreateWithInvalidConfigID(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provider_test.TestAccProtoV6ProviderFactories(),
		PreCheck:                 provider_test.TestAccPreCheck,
		Steps: []resource.TestStep{
			{
				Config: provider_test.ProviderConfig() + testSchedulerResource("invalid", map[string]any{
					"config_id":       "invalid-config-id", // Invalid config ID
					"name":            "Invalid Config Test",
					"cron_expression": "0 0 * * *",
				}),
				ExpectError: regexp.MustCompile("could not create scheduler"),
			},
		},
	})
}
