package branch

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Model defines the branch resource model.
type Model struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}
