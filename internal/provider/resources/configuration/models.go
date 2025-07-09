package configuration

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Config represents the Terraform schema for a configuration.
type ConfigModel struct {
	ID                types.String `tfsdk:"id"`
	BranchID          types.Int64  `tfsdk:"branch_id"`
	ComponentID       types.String `tfsdk:"component_id"`
	ConfigID          types.String `tfsdk:"configuration_id"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	ChangeDescription types.String `tfsdk:"change_description"`
	IsDeleted         types.Bool   `tfsdk:"is_deleted"`
	Created           types.String `tfsdk:"created"`
	IsDisabled        types.Bool   `tfsdk:"is_disabled"`
	Content           types.String `tfsdk:"configuration"`
	Rows              types.List   `tfsdk:"rows"`
	// ConfigurationFQN is the fully qualified name output (branch_id/component_id/configuration_id)
	ConfigurationFQN types.String `tfsdk:"configuration_fqn"`
}

// RowModel represents the schema for a configuration row.
type RowModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	ChangeDescription types.String `tfsdk:"change_description"`
	IsDisabled        types.Bool   `tfsdk:"is_disabled"`
	State             types.String `tfsdk:"state"`
	Content           types.String `tfsdk:"configuration_row"`
}

// GetConfigModelID returns the compound ID for a configuration, URL-encoding each segment.
// This ensures that special characters (such as '/') in IDs do not break the FQN structure.
func GetConfigModelID(model *ConfigModel) string {
	branchID := strconv.FormatInt(model.BranchID.ValueInt64(), 10)
	componentID := model.ComponentID.ValueString()
	configID := model.ConfigID.ValueString()

	return fmt.Sprintf("%s/%s/%s",
		url.QueryEscape(branchID),
		url.QueryEscape(componentID),
		url.QueryEscape(configID),
	)
}
