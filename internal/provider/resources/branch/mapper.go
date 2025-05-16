package branch

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keboola/keboola-sdk-go/v2/pkg/keboola"
)

// Mapper implements ResourceMapper for branch resources.
type Mapper struct {
	projectID int
}

// MapAPIToTerraform converts a Keboola API keboola.Branch to a Terraform model.
func (m *Mapper) MapAPIToTerraform(
	_ context.Context,
	apiModel *keboola.Branch,
	tfModel *Model,
) diag.Diagnostics {
	var diags diag.Diagnostics

	// Set Name field
	tfModel.ID = types.Int64Value(int64(apiModel.ID))
	tfModel.Name = types.StringValue(apiModel.Name)
	tfModel.Description = types.StringValue(apiModel.Description)
	tfModel.IsDefault = types.BoolValue(apiModel.IsDefault)

	return diags
}

// MapTerraformToAPI converts a Terraform branch model to a Keboola API model.
func (m *Mapper) MapTerraformToAPI(
	_ context.Context,
	stateModel Model,
	tfModel Model,
) (*keboola.Branch, error) {
	// Create a new Branch instance
	branch := &keboola.Branch{}

	// Map ID if it exists in the state
	if !stateModel.ID.IsNull() && stateModel.ID.ValueInt64() != 0 {
		branch.ID = keboola.BranchID(stateModel.ID.ValueInt64())
	}

	// Map name
	if !tfModel.Name.IsNull() {
		branch.Name = tfModel.Name.ValueString()
	}

	// Map description
	if !tfModel.Description.IsNull() {
		branch.Description = tfModel.Description.ValueString()
	}

	// Map default flag
	if !tfModel.IsDefault.IsNull() {
		branch.IsDefault = tfModel.IsDefault.ValueBool()
	}

	return branch, nil
}

// ValidateTerraformModel validates a Terraform branch model.
func (m *Mapper) ValidateTerraformModel(
	_ context.Context,
	_ *Model,
	newModel *Model,
) diag.Diagnostics {
	var diags diag.Diagnostics

	// Name is required
	if newModel.Name.IsUnknown() || newModel.Name.IsNull() {
		diags.AddError(
			"Error validating branch resource",
			"Name is required",
		)
	}

	return diags
}
