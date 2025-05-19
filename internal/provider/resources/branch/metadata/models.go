package metadata

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Model defines the metadata resource model.
type Model struct {
	ID       types.String `tfsdk:"id"`
	BranchID types.Int64  `tfsdk:"branch_id"`
	Key      types.String `tfsdk:"key"`
	Value    types.String `tfsdk:"value"`
}
