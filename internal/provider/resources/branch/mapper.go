package branch

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keboola/keboola-sdk-go/v2/pkg/keboola"
)

// Mapper implements ResourceMapper for branch resources.
type Mapper struct {
	client    *keboola.AuthorizedAPI
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
	tfModel.Name = types.StringValue(apiModel.Name)
	tfModel.ID = types.Int64Value(int64(apiModel.ID))

	return diags
}

// MapTerraformToAPI converts a Terraform branch model to a Keboola API model.
func (m *Mapper) MapTerraformToAPI(
	ctx context.Context,
	_ Model,
	tfModel Model,
) (*keboola.Branch, error) {
	// Call the API to create the branch
	result, err := m.client.CreateBranchRequest(
		&keboola.Branch{
			Name: tfModel.Name.ValueString(),
		},
	).Send(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create branch: %w", err)
	}

	return result, nil
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
