package encryption

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/keboola/go-client/pkg/keboola"

	"github.com/keboola/terraform-provider-keboola/internal/provider/abstraction"
	"github.com/keboola/terraform-provider-keboola/internal/providermodels"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ resource.Resource              = &Resource{}
	_ resource.ResourceWithConfigure = &Resource{}
)

// NewResource is a helper function to simplify the provider implementation
func NewResource() resource.Resource {
	return &Resource{}
}

// Resource is the encryption resource implementation
type Resource struct {
	// Base functionality with encryption model specifics
	base abstraction.BaseResource[Model, *EncryptResponse]

	// Direct access to the API client for specific operations
	client    *keboola.AuthorizedAPI
	projectId int
}

// Metadata returns the resource type name
func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_encryption"
}

// Schema defines the schema for the resource
func (r *Resource) Schema(_ context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server
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

// Configure adds the provider configured client to the resource
func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Return silently if provider data is not available (yet)
	if req.ProviderData == nil {
		return
	}

	// Get the provider data - ignoring the type assertion success
	providerData, _ := req.ProviderData.(*providermodels.ProviderData)

	// Set up the API client
	r.client = providerData.Client
	r.projectId = providerData.Token.ProjectID()

	// Set up the mapper
	r.base.Mapper = &EncryptionMapper{
		client:    r.client,
		projectId: r.projectId,
	}
}

// Create creates the resource and sets the initial Terraform state
func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Creating encryption resource")

	// Use the base resource abstraction for Create
	r.base.ExecuteCreate(ctx, req, resp, func(ctx context.Context, model Model) (*EncryptResponse, error) {
		// Handle API call from the mapper
		return r.base.Mapper.MapTerraformToAPI(ctx, Model{}, model)
	})
}

// Read refreshes the Terraform state with the latest data
func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Reading encryption resource")

	// Get current state - nothing to do for encryption resources as they're stateless
	var state Model
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set refreshed state
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the resource and sets the updated Terraform state
func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Updating encryption resource")

	// Get plan and state
	var plan, state Model
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

	// If the value is empty, keep the previous encrypted value
	if plan.Value.ValueString() == "" {
		tflog.Info(ctx, "Value is empty, keeping previous encrypted value")
		plan.EncryptedValue = state.EncryptedValue
		diags = resp.State.Set(ctx, plan)
		resp.Diagnostics.Append(diags...)
		return
	}

	// Use the base resource abstraction for Update
	r.base.ExecuteUpdate(ctx, req, resp, func(ctx context.Context, state Model, plan Model) (*EncryptResponse, error) {
		// Handle API call from the mapper
		return r.base.Mapper.MapTerraformToAPI(ctx, state, plan)
	})
}

// Delete deletes the resource and removes the Terraform state
func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Deleting encryption resource")

	// Nothing to do for encryption resources as they're stateless
	// but we use the base resource for consistency and proper diagnostics
	r.base.ExecuteDelete(ctx, req, resp, func(ctx context.Context, model Model) error {
		// No API call needed for deletion
		return nil
	})
}
