package encryption

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/keboola/keboola-sdk-go/v2/pkg/keboola"

	"github.com/keboola/terraform-provider-keboola/internal/provider/abstraction"
	"github.com/keboola/terraform-provider-keboola/internal/providermodels"
)

// Sentinel errors for encryption resource.
var (
	// ErrStateless indicates that no API call is needed for this operation as the resource is stateless.
	ErrStateless = errors.New("encryption resource is stateless, no API call needed")
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource = &Resource{
		base: abstraction.BaseResource[Model, *EncryptResponse]{}, client: nil, projectID: 0,
	}
	_ resource.ResourceWithConfigure = &Resource{
		base: abstraction.BaseResource[Model, *EncryptResponse]{}, client: nil, projectID: 0,
	}
)

// NewResource is a helper function to simplify the provider implementation.
func NewResource() *Resource {
	return &Resource{}
}

// Resource is the encryption resource implementation.
type Resource struct {
	// Base functionality with encryption model specifics
	base abstraction.BaseResource[Model, *EncryptResponse]

	// Direct access to the API client for specific operations
	client    *keboola.AuthorizedAPI
	projectID int
}

// Metadata returns the resource type name.
func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_encryption"
}

// Schema defines the schema for the resource.
func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server
		MarkdownDescription: "Encryption resource",
		Description:         "Encryption resource for securely storing sensitive data in Keboola",
		DeprecationMessage:  "",
		Version:             1,
		Blocks:              map[string]schema.Block{},

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
				Description:         "Id of the component where the encrypted value will be used.",
				Required:            true,
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "Value to be encrypted.",
				Description:         "Value to be encrypted.",
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
			},
			"encrypted_value": schema.StringAttribute{
				MarkdownDescription: "Actual encrypted value of the value attribute. If the value attribute changes to an empty-string then the encrypted value won't update and keep the current one.", //nolint: lll
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	// Return silently if provider data is not available (yet)
	if req.ProviderData == nil {
		return
	}

	// Get the provider data - ignoring the type assertion success
	providerData, _ := req.ProviderData.(*providermodels.ProviderData)

	// Set up the API client
	r.client = providerData.Client
	r.projectID = providerData.Token.ProjectID()

	// Set up the mapper
	r.base.Mapper = &Mapper{
		client:    r.client,
		projectID: r.projectID,
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Creating encryption resource")

	// Use the base resource abstraction for Create
	r.base.ExecuteCreate(ctx, req, resp, func(ctx context.Context, model Model) (*EncryptResponse, error) {
		// Handle API call from the mapper
		emptyModel := Model{
			ID:             types.StringNull(),
			ComponentID:    types.StringNull(),
			Value:          types.StringNull(),
			EncryptedValue: types.StringNull(),
		}

		return r.base.Mapper.MapTerraformToAPI(ctx, emptyModel, model)
	})
}

// Read refreshes the Terraform state with the latest data.
func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Reading encryption resource")

	// Use the base resource abstraction for Read
	r.base.ExecuteRead(ctx, req, resp, func(_ context.Context, _ Model) (*EncryptResponse, error) {
		// Nothing to do for encryption resources as they're stateless
		// Return sentinel error to indicate no API call is needed
		return nil, ErrStateless
	})
}

// Update updates the resource and sets the updated Terraform state.
func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Updating encryption resource")

	// Use the base resource abstraction for Update
	r.base.ExecuteUpdate(ctx, req, resp, func(ctx context.Context, state, plan Model) (*EncryptResponse, error) {
		// If the value is empty, keep the previous encrypted value
		if plan.Value.ValueString() == "" {
			tflog.Info(ctx, "Value is empty, keeping previous encrypted value")

			// Return a proper response with the existing encrypted value
			// For encryption resources, we're only concerned about the #value field
			response := EncryptResponse{
				"#value": state.EncryptedValue.ValueString(),
			}

			return &response, nil
		}

		// Handle API call from the mapper
		return r.base.Mapper.MapTerraformToAPI(ctx, state, plan)
	})
}

// Delete deletes the resource and removes the Terraform state.
func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Deleting encryption resource")

	// Use the generic base resource implementation
	r.base.ExecuteDelete(ctx, req, resp, func(_ context.Context, _ Model) error {
		// Nothing to do for encryption resources - they're virtual
		return ErrStateless
	})
}
