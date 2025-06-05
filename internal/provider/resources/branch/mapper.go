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
	branch.ID = keboola.BranchID(stateModel.ID.ValueInt64())
	branch.Name = tfModel.Name.ValueString()
	branch.Description = tfModel.Description.ValueString()
	branch.IsDefault = tfModel.IsDefault.ValueBool()

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
