package encryption

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Model defines the encryption resource model
type Model struct {
	Id             types.String `tfsdk:"id"`
	ComponentID    types.String `tfsdk:"component_id"`
	Value          types.String `tfsdk:"value"`
	EncryptedValue types.String `tfsdk:"encrypted_value"`
}
