package provider

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keboola/go-client/pkg/keboola"
	. "github.com/keboola/go-client/pkg/keboola"
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
	sapiClient *keboola.API
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
	IsDisabled        types.Bool   `tfsdk:"is_disabled"`
	Content           types.String `tfsdk:"configuration"`
}

func getConfigModelId(model *configModel) string {
	return fmt.Sprintf("%d/%v/%v", model.BranchID.ValueInt64(), model.ComponentID.ValueString(), model.ConfigID.ValueString())
}

// Metadata returns the resource type name.
func (r *configResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_component_configuration"
}

// GetSchema defines the schema for the resource.
func (r *configResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages component configurations (https://keboola.docs.apiary.io/#reference/components-and-configurations).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique string identifier assembled as branchId/componentId/configId.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"configuration_id": schema.StringAttribute{
				Description: "Id of the configuration. If not specified, then will be autogenerated.",
				Optional:    true,
				Computed:    true,
			},
			"component_id": schema.StringAttribute{
				Description: "Id of the component.",
				Required:    true,
			},
			"branch_id": schema.Int64Attribute{
				Description: "Id of the branch. If not specified, then default branch will be used.",
				Optional:    true,
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the configuration.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Description of the configuration.",
				Optional:    true,
				Computed:    true,
			},
			"change_description": schema.StringAttribute{
				Description: "Change description associated with the configuration change.",
				Optional:    true,
				Computed:    true,
			},
			"is_disabled": schema.BoolAttribute{
				Description: "Wheter configuration is enabled or disabled.",
				Optional:    true,
				Computed:    true,
			},
			"configuration": schema.StringAttribute{
				Description: "Content of the configuration specified as JSON string.",
				Optional:    true,
				Computed:    true,
			},
			"is_deleted": schema.BoolAttribute{
				Description: "Wheter configuration has been deleted or not.",
				Computed:    true,
			},
			"created": schema.StringAttribute{
				Description: "Timestamp of the configuration creation date.",
				Computed:    true,
			},
			"version": schema.Int64Attribute{
				Description: "Id of the version",
				Computed:    true,
			},
		},
	}
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
		branch, err := r.sapiClient.GetDefaultBranchRequest().Send(ctx)
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
	resConfig, err := r.sapiClient.CreateConfigRequest(config).Send(ctx)
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
	plan.Created = types.StringValue(resConfig.Created.UTC().String())
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
	config, err := r.sapiClient.GetConfigRequest(key).Send(ctx)

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
	state.Created = types.StringValue(config.Created.UTC().String())
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

	resConfig, err := r.sapiClient.UpdateConfigRequest(config, changeFields).Send(ctx)
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
	plan.Created = types.StringValue(resConfig.Created.UTC().String())
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
	err := r.sapiClient.DeleteConfigRequest(key).SendOrErr(ctx)
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
	r.sapiClient = req.ProviderData.(*providerData).client
}

// TODO: implement import via https://developer.hashicorp.com/terraform/plugin/framework/resources/import#multiple-attributes
// func (r *configResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
//     // Retrieve import ID and save to id attribute
//
//     resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
// }
