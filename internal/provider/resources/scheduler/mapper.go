package scheduler

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keboola/keboola-sdk-go/pkg/keboola"
)

// Mapper implements ResourceMapper for scheduler resources.
type Mapper struct {
	isTest bool
}

// MapAPIToTerraform converts an API model to a Terraform model.
func (m *Mapper) MapAPIToTerraform(
	_ context.Context,
	apiModel *keboola.Schedule,
	tfModel *Model,
) diag.Diagnostics {
	var diags diag.Diagnostics

	if apiModel == nil {
		return diags
	}

	// Map fields from API to Terraform model
	tfModel.ID = types.StringValue(string(apiModel.ID))
	tfModel.ConfigID = types.StringValue(string(apiModel.ConfigID))
	tfModel.ConfigurationVersion = types.StringValue(apiModel.ConfigurationVersionID)

	return diags
}

// MapTerraformToAPI converts a Terraform model to an API model.
func (m *Mapper) MapTerraformToAPI(
	_ context.Context,
	stateModel, tfModel Model,
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
func (m *Mapper) ValidateTerraformModel(
	_ context.Context,
	_, newModel *Model,
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
