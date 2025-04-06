package scheduler

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Model represents the Terraform schema for a scheduler.
type Model struct {
	ID                   types.String `tfsdk:"id"`
	ConfigID             types.String `tfsdk:"config_id"`
	ConfigurationVersion types.String `tfsdk:"configuration_version"`
}

// GetSchedulerModelID returns the ID for the scheduler model.
func GetSchedulerModelID(model *Model) string {
	return model.ID.ValueString()
}
