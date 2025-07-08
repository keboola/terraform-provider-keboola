package configurationrow

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keboola/keboola-sdk-go/v2/pkg/keboola"

	"github.com/keboola/terraform-provider-keboola/internal/provider/jsonutils"
)

// Mapper handles mapping between the Terraform schema and the internal model for configurationRow.
// Implements abstraction.ResourceMapper[Model, *keboola.ConfigRow].
type Mapper struct{}

// MapAPIToTerraform converts an API model to a Terraform model.
func (m *Mapper) MapAPIToTerraform(_ context.Context, apiModel *keboola.ConfigRow, tfModel *Model) diag.Diagnostics {
	if apiModel == nil {
		return nil
	}

	// Always set ID from API
	tfModel.ID = types.StringValue(string(apiModel.ID))

	// Determine what the user originally provided (FQN or IDs)
	userProvidedFQN := !tfModel.FQN.IsNull() && tfModel.FQN.ValueString() != ""

	if userProvidedFQN {
		// User provided FQN: set only FQN in state, null the 3 IDs
		tfModel.FQN = types.StringValue(
			apiModel.BranchID.String() + "/" + string(apiModel.ComponentID) + "/" + string(apiModel.ConfigID),
		)
		tfModel.BranchID = types.Int64Null()
		tfModel.ComponentID = types.StringNull()
		tfModel.ConfigID = types.StringNull()
	} else {
		// User provided 3 IDs: set only those, null FQN
		tfModel.BranchID = types.Int64Value(int64(apiModel.BranchID))
		tfModel.ComponentID = types.StringValue(string(apiModel.ComponentID))
		tfModel.ConfigID = types.StringValue(string(apiModel.ConfigID))
		tfModel.FQN = types.StringNull()
	}

	// Set all other fields as usual
	tfModel.Name = types.StringValue(apiModel.Name)
	tfModel.Description = types.StringValue(apiModel.Description)
	tfModel.ChangeDescription = types.StringValue(apiModel.ChangeDescription)
	tfModel.IsDisabled = types.BoolValue(apiModel.IsDisabled)

	contentStr, err := jsonutils.SerializeJSON(apiModel.Content)
	if err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("Failed to serialize content", err.Error())}
	}

	tfModel.Content = contentStr

	return nil
}

// MapTerraformToAPI converts a Terraform model to an API model.
func (m *Mapper) MapTerraformToAPI(ctx context.Context, _, tfModel Model) (*keboola.ConfigRow, error) {
	// Use getKeyFromModel to extract the parent key from the model (handles FQN or direct fields)
	rowKey, err := getKeyFromModel(ctx, tfModel)
	if err != nil {
		return nil, err
	}

	// Only override if FQN is not provided
	if tfModel.FQN.IsNull() || tfModel.FQN.ValueString() == "" {
		if !tfModel.BranchID.IsNull() {
			rowKey.BranchID = keboola.BranchID(tfModel.BranchID.ValueInt64())
		}

		if !tfModel.ComponentID.IsNull() {
			rowKey.ComponentID = keboola.ComponentID(tfModel.ComponentID.ValueString())
		}

		if !tfModel.ConfigID.IsNull() {
			rowKey.ConfigID = keboola.ConfigID(tfModel.ConfigID.ValueString())
		}
	}
	// Only set ID if provided by the user
	if !tfModel.ID.IsNull() && tfModel.ID.ValueString() != "" {
		rowKey.ID = keboola.RowID(tfModel.ID.ValueString())
	}

	contentMap, err := jsonutils.ParseJSON(tfModel.Content)
	if err != nil {
		return nil, err
	}

	return &keboola.ConfigRow{
		ConfigRowKey:      rowKey,
		Name:              tfModel.Name.ValueString(),
		Description:       tfModel.Description.ValueString(),
		ChangeDescription: tfModel.ChangeDescription.ValueString(),
		IsDisabled:        tfModel.IsDisabled.ValueBool(),
		Content:           contentMap,
	}, nil
}

// ValidateTerraformModel validates the Terraform model. Signature must match abstraction.ResourceMapper interface.
func (m *Mapper) ValidateTerraformModel(_ context.Context, _, _ *Model) diag.Diagnostics {
	return nil
}
