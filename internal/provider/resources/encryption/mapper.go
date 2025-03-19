package encryption

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keboola/go-client/pkg/keboola"
)

// EncryptResponse is a simple wrapper around the map response from the API
type EncryptResponse map[string]string

// EncryptionMapper implements ResourceMapper for encryption resources
type EncryptionMapper struct {
	client    *keboola.AuthorizedAPI
	projectId int
}

// MapAPIToTerraform converts a Keboola API EncryptResponse to a Terraform model
func (m *EncryptionMapper) MapAPIToTerraform(
	ctx context.Context,
	apiModel *EncryptResponse,
	tfModel *Model,
) diag.Diagnostics {
	var diags diag.Diagnostics

	// Set the encrypted value from the API response
	if apiModel != nil {
		encryptedValue := (*apiModel)["#value"]
		tfModel.EncryptedValue = types.StringValue(encryptedValue)
	}

	// Set ID field
	tfModel.Id = types.StringValue("none")

	return diags
}

// MapTerraformToAPI converts a Terraform model to a Keboola API EncryptResponse
func (m *EncryptionMapper) MapTerraformToAPI(
	ctx context.Context,
	tfModel Model,
) (*EncryptResponse, error) {
	// Create request body
	requestBody := map[string]string{
		"#value": tfModel.Value.ValueString(),
	}

	// Call the API to encrypt the value
	result, err := m.client.EncryptRequest(
		m.projectId,
		keboola.ComponentID(tfModel.ComponentID.ValueString()),
		requestBody,
	).Send(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to encrypt value: %w", err)
	}

	// Convert to our custom response type
	response := EncryptResponse(*result)
	return &response, nil
}

// ValidateTerraformModel validates a Terraform model for consistency and constraints
func (m *EncryptionMapper) ValidateTerraformModel(
	ctx context.Context,
	oldModel *Model,
	newModel *Model,
) diag.Diagnostics {
	var diags diag.Diagnostics

	// Component ID is required
	if newModel.ComponentID.IsUnknown() || newModel.ComponentID.IsNull() {
		diags.AddError(
			"Error validating encryption resource",
			"Component ID is required",
		)
	}

	// Set defaults for ID if new
	if oldModel == nil && newModel.Id.IsUnknown() {
		newModel.Id = types.StringValue("none")
	}

	return diags
}
