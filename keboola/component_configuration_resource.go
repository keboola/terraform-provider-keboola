package keboola

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keboola/go-client/pkg/client"
	. "github.com/keboola/go-client/pkg/storageapi"
	"github.com/keboola/go-utils/pkg/orderedmap"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &configResource{}
	_ resource.ResourceWithConfigure = &configResource{}
	//_ resource.ResourceWithImportState = &configResource{}
)

// NewConfigResource is a helper function to simplify the provider implementation.
func NewConfigResource() resource.Resource {
	return &configResource{}
}

// configResource is the resource implementation.
type configResource struct {
	sapiClient *client.Client
}

// Config https://keboola.docs.apiary.io/#reference/components-and-configurations/component-configurations/list-configurations
type configModel struct {
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
	//State             types.Map    `tfsdk:"state" readonly:"true"`
	IsDisabled types.Bool   `tfsdk:"is_disabled"`
	Content    types.String `tfsdk:"configuration"`
}

func getConfigModelId(model *configModel) string {
	return fmt.Sprintf("%d/%s/%s", model.BranchID, model.ComponentID, model.ID)
}

// Metadata returns the resource type name.
func (r *configResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_component_configuration"
}

// GetSchema defines the schema for the resource.
func (r *configResource) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: "Manages component configurations (https://keboola.docs.apiary.io/#reference/components-and-configurations).",
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Description: "Unique string identifier assembled as branchId/componentId/configId.",
				Type:        types.StringType,
				Computed:    true,
				PlanModifiers: []tfsdk.AttributePlanModifier{
					resource.UseStateForUnknown(),
				},
			},
			"configuration_id": {
				Description: "Id of the configuration.",
				Type:        types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"component_id": {
				Description: "Id of the component.",
				Type:        types.StringType,
				Required:    true,
			},
			"branch_id": {
				Description: "Id of the branch. If not specified, then default branch will be used.",
				Type:        types.Int64Type,
				Optional:    true,
				Computed:    true,
			},
			"name": {
				Description: "Name of the configuration.",
				Type:        types.StringType,
				Required:    true,
			},
			"description": {
				Description: "Description of the configuration.",
				Type:        types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"change_description": {
				Description: "Change description associated with the configuration change.",
				Type:        types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"is_disabled": {
				Description: "Wheter configuration is enabled or disabled.",
				Type:        types.BoolType,
				Optional:    true,
				Computed:    true,
			},
			"configuration": {
				Description: "Content of the configuration specified as JSON string.",
				Type:        types.StringType,
				Optional:    true,
				Computed:    true,
			},
			"is_deleted": {
				Description: "Wheter configuration has been deleted or not.",
				Type:        types.BoolType,
				Computed:    true,
			},
			"created": {
				Description: "Timestamp of the configuration creation date.",
				Type:        types.StringType,
				Computed:    true,
			},
			"version": {
				Description: "Id of the version",
				Type:        types.Int64Type,
				Computed:    true,
			},
		},
	}, nil
}

// Create creates the resource and sets the initial Terraform state.
func (r *configResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan
	var plan configModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Content.IsUnknown() {
		plan.Content = types.StringValue("{}")
	}
	configContent := orderedmap.New()
	contentBytes := []byte(plan.Content.ValueString())
	err := configContent.UnmarshalJSON(contentBytes)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating configuration",
			"Could not parse configuration: "+err.Error()+string(contentBytes),
		)
		return
	}
	if plan.ChangeDescription.IsNull() {
		plan.ChangeDescription = types.StringValue("created by Keboola terraform provider")
	}
	key := ConfigKey{
		ComponentID: ComponentID(plan.ComponentID.ValueString()),
	}
	if !plan.ConfigID.IsNull() {
		key.ID = ConfigID(plan.ConfigID.ValueString())
	}
	if plan.BranchID.IsUnknown() {
		branch, err := GetDefaultBranchRequest().Send(ctx, r.sapiClient)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating configuration",
				"Could not get default branch: "+err.Error(),
			)
			return
		}
		key.BranchID = branch.ID
	}
	config := &ConfigWithRows{
		Config: &Config{
			ConfigKey:         key,
			Name:              plan.Name.ValueString(),
			Description:       plan.Description.ValueString(),
			ChangeDescription: plan.ChangeDescription.ValueString(),
			Content:           configContent,
			IsDisabled:        plan.IsDisabled.ValueBool(),
		},
		Rows: []*ConfigRow{},
	}
	resConfig, err := CreateConfigRequest(config).Send(ctx, r.sapiClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating configuration",
			"Could not create configuration: "+err.Error(),
		)
		return
	}
	plan.ConfigID = types.StringValue(resConfig.ID.String())
	plan.BranchID = types.Int64Value(int64(resConfig.BranchID))
	plan.IsDeleted = types.BoolValue(resConfig.IsDeleted)
	plan.ID = types.StringValue(getConfigModelId(&plan))
	plan.Version = types.Int64Value(int64(resConfig.Version))
	plan.Created = types.StringValue(resConfig.Created.String())
	plan.ChangeDescription = types.StringValue(resConfig.ChangeDescription)
	plan.Description = types.StringValue(resConfig.Description)
	plan.IsDisabled = types.BoolValue(resConfig.IsDisabled)

	// Set state to fully populated data
	diags = resp.State.Set(ctx, plan)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read resource information
func (r *configResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state configModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed configuration
	key := ConfigKey{
		ID:          ConfigID(state.ConfigID.ValueString()),
		BranchID:    BranchID(state.BranchID.ValueInt64()),
		ComponentID: ComponentID(state.ComponentID.ValueString()),
	}
	config, err := GetConfigRequest(key).Send(ctx, r.sapiClient)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Configuration",
			"Could not read Configuration"+getConfigModelId(&state)+": "+err.Error(),
		)
		return
	}

	// Overwrite config with refreshed state
	state.Name = types.StringValue(config.Name)
	state.Description = types.StringValue(config.Description)
	state.ChangeDescription = types.StringValue(config.ChangeDescription)
	state.IsDeleted = types.BoolValue(config.IsDeleted)
	state.Created = types.StringValue(config.Created.String())
	state.Version = types.Int64Value(int64(config.Version))
	state.IsDisabled = types.BoolValue(config.IsDisabled)
	state.ID = types.StringValue(getConfigModelId(&state))

	currentContentMap := orderedmap.New()
	err = currentContentMap.UnmarshalJSON([]byte(state.Content.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading configuration",
			"Could not parse state configuration: "+err.Error(),
		)
		return
	}

	newContentMap := config.Content
	newContentBytes, err := newContentMap.MarshalJSON()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Configuration",
			"Could not parse read configuration "+getConfigModelId(&state)+": "+err.Error(),
		)
		return
	}
	newContentStr := string(newContentBytes)

	if !reflect.DeepEqual(currentContentMap, newContentMap) {
		resp.Diagnostics.AddWarning(
			"Read configuration changed",
			"Updating local state with to match the read configuration",
		)
		state.Content = types.StringValue(newContentStr)
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *configResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan
	var plan, state configModel
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

	if !plan.ComponentID.IsUnknown() && state.ComponentID != plan.ComponentID {
		resp.Diagnostics.AddError(
			"Error updating configuration",
			"Can not change component_id after configuration is created",
		)
		return
	}

	if !plan.BranchID.IsUnknown() && state.BranchID != plan.BranchID {
		resp.Diagnostics.AddError(
			"Error updating configuration",
			"Can not change branch_id after configuration is created",
		)
		return
	}

	if !plan.ConfigID.IsUnknown() && state.ConfigID != plan.ConfigID {
		resp.Diagnostics.AddError(
			"Error updating configuration",
			"Can not change configuration_id after configuration is created",
		)
		return
	}

	// Generate API request body from plan
	if plan.ChangeDescription.IsUnknown() {
		plan.ChangeDescription = types.StringValue("update by Keboola terraform provider")
	}

	configContent := orderedmap.New()
	if plan.Content.IsUnknown() {
		plan.Content = types.StringValue("{}")
	}
	err := configContent.UnmarshalJSON([]byte(plan.Content.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating configuration",
			"Could not parse configuration: "+err.Error(),
		)
		return
	}
	key := ConfigKey{
		ID:          ConfigID(state.ConfigID.ValueString()),
		BranchID:    BranchID(state.BranchID.ValueInt64()),
		ComponentID: ComponentID(state.ComponentID.ValueString()),
	}
	config := &Config{
		ConfigKey:         key,
		Name:              plan.Name.ValueString(),
		Description:       plan.Description.ValueString(),
		ChangeDescription: plan.ChangeDescription.ValueString(),
		Content:           configContent,
		IsDisabled:        plan.IsDisabled.ValueBool(),
	}
	changeFields := []string{"name", "description", "configuration", "changeDescription", "isDisabled"}
	//fmt.Println(state)
	//fmt.Println(plan)

	resConfig, err := UpdateConfigRequest(config, changeFields).Send(ctx, r.sapiClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating configuration",
			"Could not update configuration: "+err.Error(),
		)
		return
	}
	plan.ConfigID = types.StringValue(resConfig.ID.String())
	plan.BranchID = types.Int64Value(int64(resConfig.BranchID))
	plan.ChangeDescription = types.StringValue(resConfig.ChangeDescription)
	plan.Description = types.StringValue(resConfig.Description)
	plan.IsDisabled = types.BoolValue(resConfig.IsDisabled)
	plan.IsDeleted = types.BoolValue(resConfig.IsDeleted)
	plan.Version = types.Int64Value(int64(resConfig.Version))
	plan.Created = types.StringValue(resConfig.Created.String())
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *configResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state
	var state configModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing order
	key := ConfigKey{
		ID:          ConfigID(state.ConfigID.ValueString()),
		BranchID:    BranchID(state.BranchID.ValueInt64()),
		ComponentID: ComponentID(state.ComponentID.ValueString()),
	}
	err := DeleteConfigRequest(key).SendOrErr(ctx, r.sapiClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Configuration",
			"Could not delete configuration: "+err.Error(),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *configResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.sapiClient = req.ProviderData.(*client.Client)
}

// TODO: implement import via https://developer.hashicorp.com/terraform/plugin/framework/resources/import#multiple-attributes
// func (r *configResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
//     // Retrieve import ID and save to id attribute
//
//     resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
// }