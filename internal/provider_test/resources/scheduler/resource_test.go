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
	result := `resource "keboola_component_configuration" "` + resourceID + `" {`

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
		ExternalProviders: map[string]resource.ExternalProvider{
			"random": {
				Source:            "hashicorp/random",
				VersionConstraint: "3.1.0",
			},
		},
		Steps: []resource.TestStep{
			// Create a configuration and a scheduler for it
			{
				// First create a configuration to use with the scheduler
				Config: provider_test.ProviderConfig() + `
				resource "random_string" "test" {
					length  = 8
					special = false
					upper   = false
				}
					` + testConfigurationResource("telemetry_extractor", map[string]any{
					"name":         "Telemetry Extractor",
					"component_id": "ex-generic-v2",
				}) + testConfigurationResource("test_config_orchestrator", map[string]any{
					"name":         "Test Configuration for Orchestrator",
					"component_id": "keboola.orchestrator",
					"configuration": `{
						"phases": [
							{
								"id": "${random_string.test.result}-phase",
								"name": "Step 1",
								"dependsOn": []
							}
						],
						"tasks": [
							{
								"id": "${random_string.test.result}-task",
								"name": "ex-generic-v2-${keboola_component_configuration.telemetry_extractor.configuration_id}",
								"phase": "${random_string.test.result}-phase",
								"task": {
									"componentId": "ex-generic-v2",
									"configurationId": "${keboola_component_configuration.telemetry_extractor.configuration_id}",
									"mode": "run"
								},
								"continueOnFailure": false,
								"enabled": true
							}
						]
					}`,
					// Then create a scheduler configuration using the orchestrator config
				}) + testConfigurationResource("test_config_scheduler", map[string]any{
					"name":         "Test Configuration for Scheduler",
					"component_id": "keboola.scheduler", // Make sure component ID is correct for scheduler
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
					// Then create the actual scheduler resource using the scheduler config
				}) + testSchedulerResource("test", map[string]any{
					"configuration_id": "${keboola_component_configuration.test_config_scheduler.configuration_id}",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("keboola_scheduler.test", "id"),
					resource.TestCheckResourceAttrSet("keboola_scheduler.test", "configuration_version"),
					resource.TestCheckResourceAttrSet("keboola_component_configuration.test_config_scheduler", "configuration_id"),
				),
			},
			// Attempt to update the schedule by changing the underlying config's cronTab - should force replacement
			{
				Config: provider_test.ProviderConfig() + testSchedulerResource("test", map[string]any{
					"configuration_id":      "123",
					"configuration_version": "123",
				}),
				Destroy: true,
			},
		},
	})
}

func TestSchedulerCreateWithInvalidConfigID(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provider_test.TestAccProtoV6ProviderFactories(),
		PreCheck:                 provider_test.TestAccPreCheck,
		ExternalProviders: map[string]resource.ExternalProvider{
			"random": {
				Source:            "hashicorp/random",
				VersionConstraint: "3.1.0",
			},
		},
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
