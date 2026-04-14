package encryptedvalue

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/keboola/keboola-sdk-go/v2/pkg/keboola"

	"github.com/keboola/terraform-provider-keboola/internal/provider/abstraction"
	"github.com/keboola/terraform-provider-keboola/internal/providermodels"
)

// Sentinel errors for encrypted value resource.
var (
	// ErrStateless indicates that no API call is needed for this operation as the resource is stateless.
	ErrStateless = errors.New("encrypted value resource is stateless, no API call needed")
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.ResourceWithImportState = &Resource{}
	_ resource.Resource                = &Resource{
		base: abstraction.BaseResource[Model, *EncryptResponse]{}, client: nil, projectID: 0,
	}
	_ resource.ResourceWithConfigure = &Resource{
		base: abstraction.BaseResource[Model, *EncryptResponse]{}, client: nil, projectID: 0,
	}
)

// Resource is the encrypted value resource implementation.
type Resource struct {
	// Base functionality with encrypted value model specifics
	base abstraction.BaseResource[Model, *EncryptResponse]

	// Direct access to the API client for specific operations
	client    *keboola.AuthorizedAPI
	projectID int
}

// NewResource is a helper function to simplify the provider implementation.
func NewResource() *Resource {
	return &Resource{}
}

// Metadata returns the resource type name.
func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_encrypted_value"
}

// Schema defines the schema for the resource.
func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server
		MarkdownDescription: "Encrypted value resource",
		Description:         "Encrypted value resource for securely storing sensitive data in Keboola",
		DeprecationMessage:  "",
		Version:             1,
		Blocks:              map[string]schema.Block{},

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Encrypted value identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"component_id": schema.StringAttribute{
				MarkdownDescription: "Id of the component where the encrypted value will be used.",
				Description:         "Id of the component where the encrypted value will be used.",
				Optional:            true,
			},
			"config_id": schema.StringAttribute{
				MarkdownDescription: "Id of the configuration where the encrypted value will be used.",
				Description:         "Id of the configuration where the encrypted value will be used.",
				Optional:            true,
			},
			"branch_type": schema.StringAttribute{
				MarkdownDescription: "Type of the branch where the encrypted value will be used (e.g., 'default', 'dev').",
				Description:         "Type of the branch where the encrypted value will be used (e.g., 'default', 'dev').",
				Optional:            true,
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
	tflog.Info(ctx, "Creating encrypted value resource")

	// Use the base resource abstraction for Create
	r.base.ExecuteCreate(ctx, req, resp, func(ctx context.Context, model Model) (*EncryptResponse, error) {
		// Create request body
		requestBody := map[string]string{
			"#value": model.Value.ValueString(),
		}

		// Prepare optional parameters
		var componentID *keboola.ComponentID
		if v := model.ComponentID.ValueString(); v != "" {
			cid := keboola.ComponentID(v)
			componentID = &cid
		}

		var configID *keboola.ConfigID
		if v := model.ConfigID.ValueString(); v != "" {
			cfgid := keboola.ConfigID(v)
			configID = &cfgid
		}

		var branchType *keboola.BranchType
		if v := model.BranchType.ValueString(); v != "" {
			bt := keboola.BranchType(v)
			branchType = &bt
		}

		// Call the API to encrypt the value
		result, err := r.client.EncryptRequest(
			r.projectID,
			componentID,
			configID,
			branchType,
			requestBody,
		).Send(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt value: %w", err)
		}

		response := EncryptResponse(*result)

		return &response, nil
	})
}

// Read refreshes the Terraform state with the latest data.
func (r *Resource) Read(ctx context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) {
	tflog.Info(ctx, "Reading encrypted value resource")

	// For stateless resources, no API call is needed and no error should be returned.
	// The state is left as-is. This is a no-op.
	// Do not return an error, otherwise Terraform will treat the resource as broken.
}

// Update updates the resource and sets the updated Terraform state.
func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Updating encrypted value resource")

	// Use the base resource abstraction for Update
	r.base.ExecuteUpdate(ctx, req, resp, func(ctx context.Context, state, plan Model) (*EncryptResponse, error) {
		// If the value is empty, keep the previous encrypted value
		if plan.Value.ValueString() == "" {
			tflog.Info(ctx, "Value is empty, keeping previous encrypted value")

			// Return a proper response with the existing encrypted value
			// For encrypted value resources, we're only concerned about the #value field
			response := EncryptResponse{
				"#value": state.EncryptedValue.ValueString(),
			}

			return &response, nil
		}

		// Prepare optional parameters
		var componentID *keboola.ComponentID
		if v := plan.ComponentID.ValueString(); v != "" {
			cid := keboola.ComponentID(v)
			componentID = &cid
		}

		var configID *keboola.ConfigID
		if v := plan.ConfigID.ValueString(); v != "" {
			cfgid := keboola.ConfigID(v)
			configID = &cfgid
		}

		var branchType *keboola.BranchType
		if v := plan.BranchType.ValueString(); v != "" {
			bt := keboola.BranchType(v)
			branchType = &bt
		}

		// Create request body
		requestBody := map[string]string{
			"#value": plan.Value.ValueString(),
		}

		// Call the API to encrypt the value
		result, err := r.client.EncryptRequest(
			r.projectID,
			componentID,
			configID,
			branchType,
			requestBody,
		).Send(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt value: %w", err)
		}

		// Convert to our custom response type
		response := EncryptResponse(*result)

		return &response, nil
	})
}

// Delete deletes the resource and removes the Terraform state.
func (r *Resource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	tflog.Info(ctx, "Deleting encrypted value resource")

	// For stateless resources, no API call is needed and no error should be returned.
	// This is a no-op. Terraform will remove the state automatically.
	// Do not return an error, otherwise Terraform will treat the resource as broken.
}

// ImportState imports an existing encrypted value resource.
// The import ID should be the encrypted value string itself.
func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Info(ctx, "Importing encrypted value resource", map[string]any{
		"id": req.ID,
	})

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
