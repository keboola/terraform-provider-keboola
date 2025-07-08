package configurationrow

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Model represents the configuration row resource model.
type Model struct {
	BranchID          types.Int64  `tfsdk:"branch_id"`
	ComponentID       types.String `tfsdk:"component_id"`
	ConfigID          types.String `tfsdk:"configuration_id"`
	FQN               types.String `tfsdk:"fqn"`
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	ChangeDescription types.String `tfsdk:"change_description"`
	IsDisabled        types.Bool   `tfsdk:"is_disabled"`
	Content           types.String `tfsdk:"configuration"`
}

// GetConfigRowModelID returns the compound ID for a configuration row.
func GetConfigRowModelID(model *Model) string {
	return fmt.Sprintf("%d/%v/%v",
		model.BranchID.ValueInt64(),
		model.ComponentID.ValueString(),
		model.ConfigID.ValueString(),
	)
}
