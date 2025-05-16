package metadata

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keboola/keboola-sdk-go/v2/pkg/keboola"
)

var ErrNoMetadataID = errors.New("failed to find metadata id")

// Mapper implements ResourceMapper for branch resources.
type Mapper struct {
	projectID int
}

// MapAPIToTerraform converts a Keboola API keboola.Branch to a Terraform model.
func (m *Mapper) MapAPIToTerraform(
	_ context.Context,
	apiModel *keboola.MetadataDetail,
	tfModel *Model,
) diag.Diagnostics {
	var diags diag.Diagnostics

	// Set Name field
	tfModel.ID = types.StringValue(apiModel.ID)
	tfModel.Key = types.StringValue(apiModel.Key)
	tfModel.Value = types.StringValue(apiModel.Value)

	return diags
}

// MapTerraformToAPI converts a Terraform branch model to a Keboola API model.
func (m *Mapper) MapTerraformToAPI(
	_ context.Context,
	_ Model,
	tfModel Model,
) (*keboola.MetadataDetail, error) {
	return &keboola.MetadataDetail{
		ID:    tfModel.ID.ValueString(),
		Key:   tfModel.Key.ValueString(),
		Value: tfModel.Value.ValueString(),
	}, nil
}

// ValidateTerraformModel validates a Terraform branch model.
func (m *Mapper) ValidateTerraformModel(
	_ context.Context,
	_ *Model,
	newModel *Model,
) diag.Diagnostics {
	var diags diag.Diagnostics

	// Key is required
	if newModel.Key.IsUnknown() || newModel.Key.IsNull() {
		diags.AddError(
			"Error validating metadata resource",
			"Key is required",
		)
	}
	// Value is required
	if newModel.Value.IsUnknown() || newModel.Value.IsNull() {
		diags.AddError(
			"Error validating metadata resource",
			"Value is required",
		)
	}

	return diags
}
