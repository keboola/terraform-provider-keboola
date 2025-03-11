package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keboola/go-client/pkg/keboola"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &EncryptionResource{}
	_ resource.ResourceWithConfigure = &EncryptionResource{}
	//_ resource.ResourceWithImportState = &configResource{}
)

// NewConfigResource is a helper function to simplify the provider implementation.
func NewEnryptionResource() resource.Resource {
	return &EncryptionResource{}
}

// configResource is the resource implementation.
type EncryptionResource struct {
	sapiClient *keboola.AuthorizedAPI
	projectId  int
}

type EncryptionResourceModel struct {
	Id             types.String `tfsdk:"id"`
	ComponentID    types.String `tfsdk:"component_id"`
	Value          types.String `tfsdk:"value"`
	EncryptedValue types.String `tfsdk:"encrypted_value"`
}

func (r *EncryptionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_encryption"
}

func (r *EncryptionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Encryption resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Encryption identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"component_id": schema.StringAttribute{
				MarkdownDescription: "Id of the component where the encrypted value will be used.",
				Required:            true,
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "Value to be encrypted.",
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
			},
			"encrypted_value": schema.StringAttribute{
				MarkdownDescription: "Actual encrypted value of the value attribute. If the value attribute changes to an empty-string then the encrypted value won't update and keep the current one.",
				Computed:            true,
			},
		},
	}
}

func (r *EncryptionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.sapiClient = req.ProviderData.(*providerData).client
	r.projectId = req.ProviderData.(*providerData).token.ProjectID()
}

func (r *EncryptionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *EncryptionResourceModel
	client := r.sapiClient

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	requestBody := map[string]string{
		"#value": data.Value.ValueString(),
	}
	result, err := client.EncryptRequest(r.projectId, keboola.ComponentID(data.ComponentID.ValueString()), requestBody).Send(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to encrypt value, got error: %s", err))
		return
	}
	encryptedValue := (*result)["#value"]
	data.EncryptedValue = types.StringValue(encryptedValue)

	data.Id = types.StringValue("none")

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EncryptionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *EncryptionResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EncryptionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data *EncryptionResourceModel
	var state *EncryptionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// reencrypt
	if data.Value.ValueString() != "" {
		client := r.sapiClient
		requestBody := map[string]string{
			"#value": data.Value.ValueString(),
		}
		result, err := client.EncryptRequest(r.projectId, keboola.ComponentID(data.ComponentID.ValueString()), requestBody).Send(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to encrypt value, got error: %s", err))
			return
		}
		encryptedValue := (*result)["#value"]
		data.EncryptedValue = types.StringValue(encryptedValue)
	} else {
		resp.Diagnostics.AddWarning("Value is empty, won't be encrypted", "The value has changed to an empty string, the previous encrypted value will be kept in state.")
		data.EncryptedValue = state.EncryptedValue
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *EncryptionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *EncryptionResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

}
