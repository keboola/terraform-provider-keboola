package encryptedvalue

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keboola/keboola-sdk-go/v2/pkg/keboola"
)

// EncryptResponse is a simple wrapper around the map response from the API.
type EncryptResponse map[string]string

// Mapper implements ResourceMapper for encrypted value resources.
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

// MapTerraformToAPI converts a Terraform encrypted value model to a Keboola API model.
// For encrypted value resource we are not able to reconstruct this object.
func (m *Mapper) MapTerraformToAPI(
	_ context.Context,
	_ Model,
	_ Model,
) (*EncryptResponse, error) {
	return nil, nil //nolint: nilnil
}

// ValidateTerraformModel validates a Terraform encrypted value model.
func (m *Mapper) ValidateTerraformModel(
	_ context.Context,
	oldModel *Model,
	newModel *Model,
) diag.Diagnostics {
	var diags diag.Diagnostics

	// Set defaults for ID if new
	if oldModel == nil && newModel.ID.IsUnknown() {
		newModel.ID = types.StringValue("none")
	}

	return diags
}
