package scheduler

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SchedulerModel represents the Terraform schema for a scheduler.
type SchedulerModel struct {
	ID                   types.String `tfsdk:"id"`
	ConfigID             types.String `tfsdk:"config_id"`
	ConfigurationVersion types.String `tfsdk:"configuration_version"`
}

// GetSchedulerModelID returns the ID for the scheduler model.
func GetSchedulerModelID(model *SchedulerModel) string {
	return fmt.Sprintf("%s", model.ID.ValueString())
}
