package encryptedvalue

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Model defines the encrypted value resource model.
type Model struct {
	ID             types.String `tfsdk:"id"`
	ComponentID    types.String `tfsdk:"component_id"` // Optional: component ID for encrypted value
	ConfigID       types.String `tfsdk:"config_id"`    // Optional: configuration ID for encrypted value
	BranchType     types.String `tfsdk:"branch_type"`  // Optional: branch type for encrypted value
	Value          types.String `tfsdk:"value"`
	EncryptedValue types.String `tfsdk:"encrypted_value"`
}
