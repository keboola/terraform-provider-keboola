package configurationrow

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

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

	// Determine what the user originally provided (configuration_fqn or IDs)
	userProvidedConfigurationFQN := !tfModel.ConfigurationFQN.IsNull() && tfModel.ConfigurationFQN.ValueString() != ""

	if userProvidedConfigurationFQN {
		// User provided configuration_fqn: set only configuration_fqn in state, null the 3 IDs
		// URL-encode each segment for output
		encodedFQN := url.QueryEscape(apiModel.BranchID.String()) + "/" +
			url.QueryEscape(string(apiModel.ComponentID)) + "/" +
			url.QueryEscape(string(apiModel.ConfigID))
		tfModel.ConfigurationFQN = types.StringValue(encodedFQN)
		tfModel.BranchID = types.Int64Null()
		tfModel.ComponentID = types.StringNull()
		tfModel.ConfigID = types.StringNull()
	} else {
		// User provided 3 IDs: set only those, null configuration_fqn
		tfModel.BranchID = types.Int64Value(int64(apiModel.BranchID))
		tfModel.ComponentID = types.StringValue(string(apiModel.ComponentID))
		tfModel.ConfigID = types.StringValue(string(apiModel.ConfigID))
		tfModel.ConfigurationFQN = types.StringNull()
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
	// Use getKeyFromModel to extract the parent key from the model (handles configuration_fqn or direct fields)
	rowKey, err := getKeyFromModelWithDecode(ctx, tfModel)
	if err != nil {
		return nil, err
	}

	// Only override if configuration_fqn is not provided
	if tfModel.ConfigurationFQN.IsNull() || tfModel.ConfigurationFQN.ValueString() == "" {
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

// getKeyFromModelWithDecode returns a keboola.ConfigRowKey from the model, decoding FQN segments if needed.
func getKeyFromModelWithDecode(_ context.Context, m Model) (keboola.ConfigRowKey, error) {
	if !m.ConfigurationFQN.IsNull() && m.ConfigurationFQN.ValueString() != "" {
		configurationFQN := m.ConfigurationFQN.ValueString()

		key, ok, err := parseConfigurationFQNWithDecode(configurationFQN)
		if err != nil {
			return keboola.ConfigRowKey{}, fmt.Errorf("failed to parse configuration_fqn: %w", err)
		}

		if ok {
			return key, nil
		}

		return keboola.ConfigRowKey{}, errInvalidConfigurationFQNFormat
	}

	if m.BranchID.IsNull() || m.ComponentID.IsNull() || m.ConfigID.IsNull() {
		return keboola.ConfigRowKey{}, errMissingIdentifiers
	}

	return keboola.ConfigRowKey{
		BranchID:    keboola.BranchID(m.BranchID.ValueInt64()),
		ComponentID: keboola.ComponentID(m.ComponentID.ValueString()),
		ConfigID:    keboola.ConfigID(m.ConfigID.ValueString()),
	}, nil
}

// parseConfigurationFQNWithDecode parses configuration_fqn with 3 URL-decoded parts.
func parseConfigurationFQNWithDecode(configurationFQN string) (keboola.ConfigRowKey, bool, error) {
	parts := strings.Split(configurationFQN, "/")
	if len(parts) != configurationFQNPartsCount {
		return keboola.ConfigRowKey{}, false, nil
	}

	branchIDStr, err := url.QueryUnescape(parts[0])
	if err != nil {
		return keboola.ConfigRowKey{}, false, fmt.Errorf("failed to decode branch_id: %w", err)
	}

	componentID, err := url.QueryUnescape(parts[1])
	if err != nil {
		return keboola.ConfigRowKey{}, false, fmt.Errorf("failed to decode component_id: %w", err)
	}

	configID, err := url.QueryUnescape(parts[2])
	if err != nil {
		return keboola.ConfigRowKey{}, false, fmt.Errorf("failed to decode config_id: %w", err)
	}

	branchID, err := strconv.ParseInt(branchIDStr, 10, 64)
	if err != nil {
		return keboola.ConfigRowKey{}, false, fmt.Errorf("failed to parse branch_id as int: %w", err)
	}

	return keboola.ConfigRowKey{
		BranchID:    keboola.BranchID(branchID),
		ComponentID: keboola.ComponentID(componentID),
		ConfigID:    keboola.ConfigID(configID),
	}, true, nil
}
