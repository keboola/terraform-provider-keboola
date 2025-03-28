package scheduler

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keboola/go-client/pkg/keboola"
)

// SchedulerMapper implements ResourceMapper for scheduler resources.
type SchedulerMapper struct {
	isTest bool
}

// MapAPIToTerraform converts an API model to a Terraform model.
func (m *SchedulerMapper) MapAPIToTerraform(
	ctx context.Context,
	apiModel *keboola.Schedule,
	tfModel *SchedulerModel,
) diag.Diagnostics {
	var diags diag.Diagnostics

	if apiModel == nil {
		return diags
	}

	// Map fields from API to Terraform model
	tfModel.ID = types.StringValue(string(apiModel.ID))
	tfModel.ConfigID = types.StringValue(string(apiModel.ConfigID))
<<<<<<< Updated upstream
	tfModel.ConfigurationVersion = types.StringValue(string(apiModel.ScheduleCron.CronTab))
=======
	tfModel.ConfigurationVersion = types.StringValue(apiModel.ConfigurationVersionID)

>>>>>>> Stashed changes
	return diags
}

// MapTerraformToAPI converts a Terraform model to an API model.
func (m *SchedulerMapper) MapTerraformToAPI(
	ctx context.Context,
	stateModel, tfModel SchedulerModel,
) (*keboola.Schedule, error) {
	// Create a new Schedule instance
	schedule := &keboola.Schedule{}

	// Map ID if it exists in the state
	if !stateModel.ID.IsNull() && stateModel.ID.ValueString() != "" {
		schedule.ID = keboola.ScheduleID(stateModel.ID.ValueString())
	}

	// Map config ID
	if !tfModel.ConfigID.IsNull() {
		schedule.ConfigID = keboola.ConfigID(tfModel.ConfigID.ValueString())
	}

	return schedule, nil
}

// ValidateTerraformModel validates a Terraform model against constraints.
func (m *SchedulerMapper) ValidateTerraformModel(
	ctx context.Context,
	oldModel, newModel *SchedulerModel,
) diag.Diagnostics {
	var diags diag.Diagnostics

	// Skip validation in test mode
	if m.isTest {
		return diags
	}

	// Validate required fields
	if newModel.ConfigID.IsNull() || newModel.ConfigID.ValueString() == "" {
		diags.AddError(
			"Invalid Configuration",
			"ConfigID is required for scheduler",
		)
	}

	return diags
}
