package abstraction

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// ResourceMapper defines the interface for mapping between API and Terraform models.
type ResourceMapper[TfModel any, ApiModel any] interface {
	// MapAPIToTerraform converts an API model to a Terraform model
	MapAPIToTerraform(ctx context.Context, apiModel ApiModel, tfModel *TfModel) diag.Diagnostics

	// MapTerraformToAPI converts a Terraform model to an API model
	MapTerraformToAPI(ctx context.Context, stateModel, tfModel TfModel) (ApiModel, error)

	// ValidateTerraformModel validates a Terraform model against constraints
	ValidateTerraformModel(ctx context.Context, oldModel, newModel *TfModel) diag.Diagnostics
}

// NestedResourceHandler handles operations for nested resources within a parent resource.
type NestedResourceHandler[ParentTfModel any, ChildTfModel any, ParentApiModel any, ChildApiModel any] interface {
	// ExtractChildModels extracts child models from parent Terraform model
	ExtractChildModels(ctx context.Context, parent ParentTfModel) ([]ChildTfModel, diag.Diagnostics)

	// MapChildModelsToAPI converts child Terraform models to API models
	MapChildModelsToAPI(ctx context.Context, childModels []ChildTfModel) ([]ChildApiModel, error)

	// ProcessAPIChildModels processes child API models after API operations
	ProcessAPIChildModels(ctx context.Context, parent *ParentTfModel, childAPIModels []ChildApiModel) diag.Diagnostics
}

// BaseResource provides common functionality for resources with generic CRUD operations.
type BaseResource[TfModel any, ApiModel any] struct {
	Mapper ResourceMapper[TfModel, ApiModel]

	// Optional nested resource handler
	NestedHandler any // Type will be cast based on context
}

// ExecuteCreate executes the create operation with proper error handling and mapping.
func (r *BaseResource[TfModel, ApiModel]) ExecuteCreate(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
	createFn func(ctx context.Context, model TfModel) (ApiModel, error),
) {
	tflog.Info(ctx, "Starting resource create operation")

	// Get plan data
	var plan TfModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate the plan
	diags = r.Mapper.ValidateTerraformModel(ctx, nil, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the resource
	apiModel, err := createFn(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating resource",
			"Could not create resource: "+err.Error(),
		)

		return
	}

	// Map API model back to Terraform model
	diags = r.Mapper.MapAPIToTerraform(ctx, apiModel, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "Completed resource create operation")
}

// ExecuteRead executes the read operation with proper error handling and mapping.
func (r *BaseResource[TfModel, ApiModel]) ExecuteRead(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
	readFn func(ctx context.Context, model TfModel) (ApiModel, error),
) {
	tflog.Info(ctx, "Starting resource read operation")

	// Get current state
	var state TfModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read the resource
	apiModel, err := readFn(ctx, state)
	if err != nil {
		// Check if this is a sentinel error (like ErrStateless)
		// If it's a sentinel error indicating a stateless resource or a similar
		// known condition that's not an actual error, we can just use the existing state
		if isSentinelError(err) {
			tflog.Debug(ctx, "Received sentinel error, using existing state", map[string]any{
				"error": err.Error(),
			})
			// Just set the existing state and return
			diags = resp.State.Set(ctx, state)
			resp.Diagnostics.Append(diags...)

			return
		}

		// It's a real error
		resp.Diagnostics.AddError(
			"Error reading resource",
			"Could not read resource: "+err.Error(),
		)

		return
	}

	// Map API model back to Terraform model
	diags = r.Mapper.MapAPIToTerraform(ctx, apiModel, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "Completed resource read operation")
}

// isSentinelError determines if an error is a sentinel error that should be handled specially.
// Currently this is done by checking for known error messages that indicate
// a non-problematic condition like a stateless resource.
func isSentinelError(err error) bool {
	// This is a simple approach that checks for the error message
	// If more sentinel errors are added, this could be enhanced to use error wrapping
	// and type assertions
	return err != nil && err.Error() == "encryption resource is stateless, no API call needed"
}

// ExecuteUpdate executes the update operation with proper error handling and mapping.
func (r *BaseResource[TfModel, ApiModel]) ExecuteUpdate(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
	updateFn func(ctx context.Context, state, plan TfModel) (ApiModel, error),
) {
	tflog.Info(ctx, "Starting resource update operation")

	// Get plan and state
	var plan, state TfModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate the plan
	diags = r.Mapper.ValidateTerraformModel(ctx, &state, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the resource
	apiModel, err := updateFn(ctx, state, plan)
	if err != nil {
		// Check if this is a sentinel error (like ErrStateless)
		if isSentinelError(err) {
			tflog.Debug(ctx, "Received sentinel error during update, using existing state", map[string]any{
				"error": err.Error(),
			})
			// Just set the plan as the new state and return
			diags = resp.State.Set(ctx, plan)
			resp.Diagnostics.Append(diags...)

			return
		}

		// It's a real error
		resp.Diagnostics.AddError(
			"Error updating resource",
			"Could not update resource: "+err.Error(),
		)

		return
	}

	// Map API model back to Terraform model
	diags = r.Mapper.MapAPIToTerraform(ctx, apiModel, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set state
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)

	tflog.Info(ctx, "Completed resource update operation")
}

// ExecuteDelete executes the delete operation with proper error handling.
func (r *BaseResource[TfModel, ApiModel]) ExecuteDelete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
	deleteFn func(ctx context.Context, model TfModel) error,
) {
	tflog.Info(ctx, "Starting resource delete operation")

	// Get current state
	var state TfModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the resource
	err := deleteFn(ctx, state)
	if err != nil {
		// Check if this is a sentinel error (like ErrStateless)
		if isSentinelError(err) {
			tflog.Debug(ctx, "Received sentinel error during delete, operation successful", map[string]any{
				"error": err.Error(),
			})

			return
		}

		// It's a real error
		resp.Diagnostics.AddError(
			"Error deleting resource",
			"Could not delete resource: "+err.Error(),
		)

		return
	}

	tflog.Info(ctx, "Completed resource delete operation")
}

// HandleNestedResources is a helper for processing nested resources.
func HandleNestedResources[ParentTfModel, ChildTfModel, ParentApiModel, ChildApiModel any](
	ctx context.Context,
	handler NestedResourceHandler[ParentTfModel, ChildTfModel, ParentApiModel, ChildApiModel],
	parent *ParentTfModel,
	_ ParentApiModel,
	apiChildren []ChildApiModel,
) diag.Diagnostics {
	return handler.ProcessAPIChildModels(ctx, parent, apiChildren)
}
