package configurationrow

import (
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Model represents the configuration row resource model.
type Model struct {
	BranchID          types.String `tfsdk:"branch_id"`
	ComponentID       types.String `tfsdk:"component_id"`
	ConfigID          types.String `tfsdk:"configuration_id"`
	ConfigurationFQN  types.String `tfsdk:"configuration_fqn"`
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	ChangeDescription types.String `tfsdk:"change_description"`
	IsDisabled        types.Bool   `tfsdk:"is_disabled"`
	Content           types.String `tfsdk:"configuration"`
}

// GetConfigRowModelID returns the compound ID for a configuration row, URL-encoding each segment.
// This ensures that special characters (such as '/') in IDs do not break the FQN structure.
func GetConfigRowModelID(model *Model) string {
	branchID := model.BranchID.ValueString()
	componentID := model.ComponentID.ValueString()
	configID := model.ConfigID.ValueString()

	return fmt.Sprintf("%s/%s/%s",
		url.QueryEscape(branchID),
		url.QueryEscape(componentID),
		url.QueryEscape(configID),
	)
}
