package scheduler

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SchedulerModel represents the Terraform schema for a scheduler.
type SchedulerModel struct {
	ID                   types.String `tfsdk:"id"`
	ScheduleID           types.String `tfsdk:"schedule_id"`
	ConfigID             types.String `tfsdk:"config_id"`
	CronExpr             types.String `tfsdk:"cron_expression"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	TimezoneID           types.String `tfsdk:"timezone_id"`
	State                types.String `tfsdk:"state"`
	LastExecutedAt       types.String `tfsdk:"last_executed_at"`
	NextExecutionAt      types.String `tfsdk:"next_execution_at"`
	Active               types.Bool   `tfsdk:"active"`
	RunWithTokenID       types.String `tfsdk:"run_with_token_id"`
	VersionDependent     types.Bool   `tfsdk:"version_dependent"`
	ConfigurationVersion types.String `tfsdk:"configuration_version"`
}

// GetSchedulerModelID returns the ID for the scheduler model.
func GetSchedulerModelID(model *SchedulerModel) string {
	return fmt.Sprintf("%s", model.ScheduleID.ValueString())
}
