package keboola

import (
	"context"

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
	BranchID          types.Int64  `tfsdk:"branch_id" writeoptional:"true"`
	ComponentID       types.String `tfsdk:"component_id"`
	ID                types.String `tfsdk:"configuration_id"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	ChangeDescription types.String `tfsdk:"change_description"`
	IsDeleted         types.Bool   `tfsdk:"is_deleted" readonly:"true"`
	Created           types.String `tfsdk:"created" readonly:"true"`
	Version           types.Int64  `tfsdk:"version" readonly:"true"`
	//State             types.Map    `tfsdk:"state" readonly:"true"`
	IsDisabled types.Bool   `tfsdk:"is_disabled"`
	Content    types.String `tfsdk:"configuration"`
}

// Metadata returns the resource type name.
func (r *configResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_config"
}

// GetSchema defines the schema for the resource.
func (r *configResource) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"configuration_id": {
				Type:     types.StringType,
				Optional: true,
				Computed: true,
			},
			"component_id": {
				Type:     types.StringType,
				Required: true,
			},
			"branch_id": {
				Type:     types.Int64Type,
				Optional: true,
				Computed: true,
			},
			"name": {
				Type:     types.StringType,
				Required: true,
			},
			"description": {
				Type:     types.StringType,
				Optional: true,
			},
			"change_description": {
				Type:     types.StringType,
				Optional: true,
				Computed: true,
			},
			"is_deleted": {
				Type:     types.BoolType,
				Computed: true,
			},
			"created": {
				Type:     types.StringType,
				Computed: true,
			},
			"version": {
				Type:     types.Int64Type,
				Computed: true,
			},
			// "state": {
			// 	Type:     types.StringType,
			// 	Computed: true,
			// },
			"is_disabled": {
				Type:     types.BoolType,
				Optional: true,
			},
			"configuration": {
				Type:     types.StringType,
				Required: true,
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
	configContent := orderedmap.New()
	contentBytes := []byte(plan.Content.ValueString())
	err := configContent.UnmarshalJSON(contentBytes)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating configuration",
			"Could not parse configuration: "+err.Error(),
		)
		return
	}
	if plan.ChangeDescription.IsNull() {
		plan.ChangeDescription = types.StringValue("created by Keboola terraform provider")
	}
	key := ConfigKey{
		ComponentID: ComponentID(plan.ComponentID.ValueString()),
	}
	if !plan.ID.IsNull() {
		key.ID = ConfigID(plan.ID.ValueString())
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
	plan.ID = types.StringValue(resConfig.ID.String())
	plan.BranchID = types.Int64Value(int64(resConfig.BranchID))
	plan.IsDeleted = types.BoolValue(resConfig.IsDeleted)
	plan.Version = types.Int64Value(int64(resConfig.Version))
	plan.Created = types.StringValue(resConfig.Created.String())
	plan.ChangeDescription = types.StringValue(resConfig.ChangeDescription)
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
		ID:          ConfigID(state.ID.ValueString()),
		BranchID:    BranchID(state.BranchID.ValueInt64()),
		ComponentID: ComponentID(state.ComponentID.ValueString()),
	}
	config, err := GetConfigRequest(key).Send(ctx, r.sapiClient)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Configuration",
			"Could not read Configuration"+state.BranchID.String()+"/"+state.ComponentID.ValueString()+"/"+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	// Overwrite config with refreshed state
	content, err := config.Content.MarshalJSON()
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Configuration",
			"Could not parse configuration "+state.BranchID.String()+"/"+state.ComponentID.ValueString()+"/"+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}
	state.Name = types.StringValue(config.Name)
	state.Description = types.StringValue(config.Description)
	state.ChangeDescription = types.StringValue(config.ChangeDescription)
	state.IsDeleted = types.BoolValue(config.IsDeleted)
	state.Created = types.StringValue(config.Created.String())
	state.Version = types.Int64Value(int64(config.Version))
	state.IsDisabled = types.BoolValue(config.IsDisabled)
	state.Content = types.StringValue(string(content))

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

	// Generate API request body from plan
	configContent := orderedmap.New()
	err := configContent.UnmarshalJSON([]byte(plan.Content.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating configuration",
			"Could not parse configuration: "+err.Error(),
		)
		return
	}
	key := ConfigKey{
		ID:          ConfigID(state.ID.ValueString()),
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
	changeFields := []string{}

	if !plan.Name.IsUnknown() {
		changeFields = append(changeFields, "name")
	}

	if !plan.Description.IsUnknown() {
		changeFields = append(changeFields, "description")
	}

	if !plan.Content.IsUnknown() {
		changeFields = append(changeFields, "configuration")
	}

	if !plan.ChangeDescription.IsUnknown() {
		changeFields = append(changeFields, "changeDescription")
	}

	if !plan.IsDisabled.IsUnknown() {
		changeFields = append(changeFields, "isDisabled")
	}

	resConfig, err := UpdateConfigRequest(config, changeFields).Send(ctx, r.sapiClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating configuration",
			"Could not update configuration: "+err.Error(),
		)
		return
	}
	plan.ID = types.StringValue(resConfig.ID.String())
	plan.BranchID = types.Int64Value(int64(resConfig.BranchID))
	plan.ChangeDescription = types.StringValue(resConfig.ChangeDescription)
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
		ID:          ConfigID(state.ID.ValueString()),
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
