package encryption

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keboola/keboola-sdk-go/v2/pkg/keboola"
)

// EncryptResponse is a simple wrapper around the map response from the API.
type EncryptResponse map[string]string

// Mapper implements ResourceMapper for encryption resources.
type Mapper struct {
	client    *keboola.AuthorizedAPI
	projectID int
}

// MapAPIToTerraform converts a Keboola API EncryptResponse to a Terraform model.
func (m *Mapper) MapAPIToTerraform(
	_ context.Context,
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
	tfModel.ID = types.StringValue("none")

	return diags
}

// MapTerraformToAPI converts a Terraform encryption model to a Keboola API model.
// For encryption resource we are not able to reconstruct this object.
func (m *Mapper) MapTerraformToAPI(
	_ context.Context,
	_ Model,
	_ Model,
) (*EncryptResponse, error) {
	return nil, nil //nolint: nilnil
}

// ValidateTerraformModel validates a Terraform encryption model.
func (m *Mapper) ValidateTerraformModel(
	_ context.Context,
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
	if oldModel == nil && newModel.ID.IsUnknown() {
		newModel.ID = types.StringValue("none")
	}

	return diags
}
