package configuration

import (
	"fmt"

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
	Version           types.Int64  `tfsdk:"version"`
	IsDisabled        types.Bool   `tfsdk:"is_disabled"`
	Content           types.String `tfsdk:"configuration"`
	Rows              types.List   `tfsdk:"rows"`
}

// RowModel represents the schema for a configuration row.
type RowModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	ChangeDescription types.String `tfsdk:"change_description"`
	IsDisabled        types.Bool   `tfsdk:"is_disabled"`
	Version           types.Int64  `tfsdk:"version"`
	State             types.String `tfsdk:"state"`
	Content           types.String `tfsdk:"configuration_row"`
}

// GetConfigModelID returns the compound ID for a configuration.
func GetConfigModelID(model *ConfigModel) string {
	return fmt.Sprintf("%d/%v/%v",
		model.BranchID.ValueInt64(),
		model.ComponentID.ValueString(),
		model.ConfigID.ValueString(),
	)
}
