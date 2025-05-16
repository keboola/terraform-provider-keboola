package metadata

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/keboola/keboola-sdk-go/v2/pkg/keboola"

	"github.com/keboola/terraform-provider-keboola/internal/provider/abstraction"
	"github.com/keboola/terraform-provider-keboola/internal/providermodels"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource = &Resource{
		base: abstraction.BaseResource[Model, *keboola.MetadataDetail]{}, client: nil, projectID: 0,
	}
)

// Resource is the branch resource implementation.
type Resource struct {
	// Base functionality with branch model specifics
	base abstraction.BaseResource[Model, *keboola.MetadataDetail]

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
	resp.TypeName = req.ProviderTypeName + "_branch_metadata"
}

// Schema defines the schema for the resource.
func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server
		MarkdownDescription: "Branch resource",
		Description:         "Development branch resource",
		DeprecationMessage:  "",
		Version:             1,
		Blocks:              map[string]schema.Block{},

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Metadata ID",
				Computed:            true,
			},
			"branch_id": schema.Int64Attribute{
				MarkdownDescription: "Branch ID",
				Required:            true,
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "Metadata key",
				Required:            true,
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "Metadata value",
				Required:            true,
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
		projectID: r.projectID,
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Creating branch metadata resource")

	// Use the base resource abstraction for Create
	r.base.ExecuteCreate(ctx, req, resp, func(ctx context.Context, model Model) (*keboola.MetadataDetail, error) {
		return r.updateMetadata(ctx, model)
	})
}

// Read refreshes the Terraform state with the latest data.
func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Reading branch metadata resource")

	// Use the base resource abstraction for Read
	r.base.ExecuteRead(ctx, req, resp, func(ctx context.Context, state Model) (*keboola.MetadataDetail, error) {
		// Get metadata with matching ID
		branch, err := r.client.ListBranchMetadataRequest(
			keboola.BranchKey{
				ID: keboola.BranchID(state.BranchID.ValueInt64()),
			},
		).Send(ctx)
		if err != nil {
			return nil, fmt.Errorf("could not get branch metadata: %w", err)
		}

		value := branch.ToMap()[state.Key.ValueString()]

		return &keboola.MetadataDetail{
			ID:    state.ID.ValueString(),
			Key:   state.Key.ValueString(),
			Value: value,
		}, nil
	})
}

// Update updates the resource and sets the updated Terraform state.
func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Updating branch metadata resource")

	// Use the base resource abstraction for Update
	r.base.ExecuteUpdate(ctx, req, resp, func(ctx context.Context, _, plan Model) (*keboola.MetadataDetail, error) {
		return r.updateMetadata(ctx, plan)
	})
}

// Delete deletes the resource and removes the Terraform state.
func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Deleting branch metadata resource")

	// Use the generic base resource implementation
	r.base.ExecuteDelete(ctx, req, resp, func(ctx context.Context, state Model) error {
		// Create key from model
		key := keboola.BranchKey{
			ID: keboola.BranchID(state.BranchID.ValueInt64()),
		}

		// Delete the branch
		err := r.client.DeleteBranchMetadataRequest(key, state.ID.ValueString()).SendOrErr(ctx)
		if err != nil {
			return fmt.Errorf("could not delete branch metadata: %w", err)
		}

		return nil
	})
}

func (r *Resource) updateMetadata(ctx context.Context, model Model) (*keboola.MetadataDetail, error) {
	metadata := make(keboola.Metadata)
	metadata[model.Key.ValueString()] = model.Value.ValueString()

	// Call the API to create the branch
	branchKey := keboola.BranchKey{
		ID: keboola.BranchID(int(model.BranchID.ValueInt64())),
	}

	_, err := r.client.AppendBranchMetadataRequest(
		branchKey,
		metadata,
	).Send(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create metadata: %w", err)
	}

	result, err := r.client.ListBranchMetadataRequest(branchKey).Send(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list metadata: %w", err)
	}

	var id string
	for _, metadataDetail := range *result {
		if metadataDetail.Key == model.Key.ValueString() {
			id = metadataDetail.ID
		}
	}

	if id == "" {
		return nil, ErrNoMetadataID
	}

	return &keboola.MetadataDetail{
		ID:    id,
		Key:   model.Key.ValueString(),
		Value: model.Value.ValueString(),
	}, nil
}
