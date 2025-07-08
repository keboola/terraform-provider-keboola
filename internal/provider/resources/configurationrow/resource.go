package configurationrow

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/keboola/keboola-sdk-go/v2/pkg/keboola"

	"github.com/keboola/terraform-provider-keboola/internal/provider/abstraction"
	"github.com/keboola/terraform-provider-keboola/internal/providermodels"
)

// Resource implements the configuration row resource for Terraform.
type Resource struct {
	base   abstraction.BaseResource[Model, *keboola.ConfigRow]
	client *keboola.AuthorizedAPI
}

// NewResource returns a new keboola_configuration_row resource.
// Implements github.com/hashicorp/terraform-plugin-framework/resource.Resource.
func NewResource() *Resource {
	return &Resource{}
}

// Metadata sets the resource type name.
func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_configuration_row"
}

// Schema defines the Terraform schema for the resource.
func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a configuration row in Keboola. You must specify either 'fqn' or all of 'branch_id', " +
			"'component_id', and 'configuration_id'. These options are mutually exclusive.",
		MarkdownDescription: "Manages a configuration row in Keboola. You must specify either `fqn` or all of `branch_id`, " +
			"`component_id`, and `configuration_id`. These options are mutually exclusive.",
		Attributes: map[string]schema.Attribute{
			"branch_id": schema.Int64Attribute{
				Description:         "ID of the branch. Mutually exclusive with 'fqn'.",
				MarkdownDescription: "ID of the branch. Mutually exclusive with `fqn`.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("fqn"),
					),
					int64validator.ExactlyOneOf(
						path.MatchRelative().AtParent().AtName("branch_id"),
						path.MatchRelative().AtParent().AtName("fqn"),
					),
				},
				// ForceNew: Changing branch_id requires recreation of the row resource.
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"component_id": schema.StringAttribute{
				Description:         "ID of the component. Mutually exclusive with 'fqn'.",
				MarkdownDescription: "ID of the component. Mutually exclusive with `fqn`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("fqn"),
					),
				},
				// ForceNew: Changing component_id requires recreation of the row resource.
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"configuration_id": schema.StringAttribute{
				Description:         "ID of the configuration. Mutually exclusive with 'fqn'.",
				MarkdownDescription: "ID of the configuration. Mutually exclusive with `fqn`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("fqn"),
					),
				},
				// ForceNew: Changing configuration_id requires recreation of the row resource.
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"fqn": schema.StringAttribute{
				Description: "Fully qualified name for the configuration row. " +
					"Required if not using branch_id/component_id/configuration_id.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("branch_id"),
						path.MatchRelative().AtParent().AtName("component_id"),
						path.MatchRelative().AtParent().AtName("configuration_id"),
					),
					stringvalidator.ExactlyOneOf(
						path.MatchRelative().AtParent().AtName("fqn"),
						path.MatchRelative().AtParent().AtName("branch_id"),
					),
				},
			},
			"id": schema.StringAttribute{
				Description:         "Compound ID of the configuration row.",
				MarkdownDescription: "Compound ID of the configuration row.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description:         "Name of the configuration row.",
				MarkdownDescription: "Name of the configuration row.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				Description:         "Description of the configuration row.",
				MarkdownDescription: "Description of the configuration row.",
				Optional:            true,
				Computed:            true,
			},
			"change_description": schema.StringAttribute{
				Description:         "Change description associated with the configuration row change.",
				MarkdownDescription: "Change description associated with the configuration row change.",
				Optional:            true,
				Computed:            true,
			},
			"is_disabled": schema.BoolAttribute{
				Description:         "Whether configuration row is enabled or disabled.",
				MarkdownDescription: "Whether configuration row is enabled or disabled.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"configuration": schema.StringAttribute{
				Description:         "Content of the configuration row as a JSON object.",
				MarkdownDescription: "Content of the configuration row as a JSON object.",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

// Configure sets the Keboola API client for the resource.
func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	// For consistency with other resources, expect ProviderData to be of type *providermodels.ProviderData.
	// This ensures the client is set correctly and avoids nil pointer panics.
	if providerData, ok := req.ProviderData.(*providermodels.ProviderData); ok {
		r.client = providerData.Client
	} else {
		r.client = nil // Defensive: set to nil if type assertion fails
	}

	// r.isTest = os.Getenv("TF_ACC") != "" // forbidden, use env.Map or project abstraction instead
	// Set up the mapper
	r.base.Mapper = &Mapper{}
}

// Static error variables for consistent error handling.
var (
	errInvalidFQNFormat      = errors.New("invalid fqn format, expected branch_id/component_id/configuration_id")
	errMissingIdentifiers    = errors.New("must provide either fqn or all of branch_id, component_id, configuration_id")
	errInvalidBranchIDFormat = errors.New("invalid branch_id format, expected integer")
	errRowIDMissing          = errors.New("row ID is missing in state")
)

const fqnPartsCount = 3 // for magic number check

// getKeyFromModel returns a keboola.ConfigRowKey from the model, using fqn if set, otherwise the individual IDs.
func getKeyFromModel(ctx context.Context, m Model) (keboola.ConfigRowKey, error) {
	if !m.FQN.IsNull() && m.FQN.ValueString() != "" {
		fqn := m.FQN.ValueString()

		key, ok, err := parseFQN(fqn)
		if err != nil {
			return keboola.ConfigRowKey{}, fmt.Errorf("%w: %w", errInvalidFQNFormat, err)
		}

		if ok {
			tflog.Info(ctx, "Parsed FQN", map[string]any{"fqn": fqn, "key": key})

			return key, nil
		}

		return keboola.ConfigRowKey{}, errInvalidFQNFormat
	}

	if m.BranchID.IsNull() || m.ComponentID.IsNull() || m.ConfigID.IsNull() {
		return keboola.ConfigRowKey{}, errMissingIdentifiers
	}

	return keboola.ConfigRowKey{
		BranchID:    keboola.BranchID(m.BranchID.ValueInt64()),
		ComponentID: keboola.ComponentID(m.ComponentID.ValueString()),
		ConfigID:    keboola.ConfigID(m.ConfigID.ValueString()),
	}, nil
}

// parseFQN parses FQN with 3 parts (branch_id/component_id/config_id).
func parseFQN(fqn string) (keboola.ConfigRowKey, bool, error) {
	parts := strings.Split(fqn, "/")
	if len(parts) != fqnPartsCount {
		return keboola.ConfigRowKey{}, false, nil
	}

	branchID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return keboola.ConfigRowKey{}, false, fmt.Errorf("%w: %w", errInvalidBranchIDFormat, err)
	}

	return keboola.ConfigRowKey{
		BranchID:    keboola.BranchID(branchID),
		ComponentID: keboola.ComponentID(parts[1]),
		ConfigID:    keboola.ConfigID(parts[2]),
	}, true, nil
}

// Create creates a new configuration row using the Keboola API.
func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.base.ExecuteCreate(ctx, req, resp, func(ctx context.Context, plan Model) (*keboola.ConfigRow, error) {
		row, err := r.base.Mapper.MapTerraformToAPI(ctx, Model{}, plan)
		if err != nil {
			return nil, err
		}

		// Log the parent context and row ID before creation
		tflog.Info(ctx, "Creating configuration row", map[string]any{
			"branch_id":        row.BranchID,
			"component_id":     row.ComponentID,
			"configuration_id": row.ConfigID,
			"row_id":           row.ID,
		})

		created, err := r.client.CreateConfigRowRequest(row).Send(ctx)
		if err != nil {
			return nil, fmt.Errorf("CreateConfigRowRequest failed: %w", err)
		}

		// Log the returned row ID after creation
		tflog.Info(ctx, "Created configuration row", map[string]any{
			"branch_id":        created.BranchID,
			"component_id":     created.ComponentID,
			"configuration_id": created.ConfigID,
			"row_id":           created.ID,
		})

		return created, nil
	})
}

// Read fetches the configuration row from the Keboola API.
func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.base.ExecuteRead(ctx, req, resp, func(ctx context.Context, state Model) (*keboola.ConfigRow, error) {
		key, err := getKeyFromModel(ctx, state)
		if err != nil {
			return nil, fmt.Errorf("getKeyFromModel failed: %w", err)
		}
		// Ensure the row ID is set from state.ID
		if !state.ID.IsNull() && state.ID.ValueString() != "" {
			key.ID = keboola.RowID(state.ID.ValueString())
		} else {
			return nil, errRowIDMissing
		}

		// Log the parent context and row ID before read
		tflog.Info(ctx, "Reading configuration row", map[string]any{
			"branch_id":        key.BranchID,
			"component_id":     key.ComponentID,
			"configuration_id": key.ConfigID,
			"row_id":           key.ID,
		})

		row, err := r.client.GetConfigRowRequest(key).Send(ctx)
		if err != nil {
			return nil, fmt.Errorf("GetConfigRowRequest failed: %w", err)
		}

		// Log the returned row ID after read
		tflog.Info(ctx, "Read configuration row", map[string]any{
			"branch_id":        row.BranchID,
			"component_id":     row.ComponentID,
			"configuration_id": row.ConfigID,
			"row_id":           row.ID,
		})

		return row, nil
	})
}

// Update updates the configuration row using the Keboola API.
func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.base.ExecuteUpdate(ctx, req, resp, func(ctx context.Context, state, plan Model) (*keboola.ConfigRow, error) {
		row, err := r.base.Mapper.MapTerraformToAPI(ctx, state, plan)
		if err != nil {
			return nil, fmt.Errorf("MapTerraformToAPI failed: %w", err)
		}

		// Log the parent context and row ID before update
		tflog.Info(ctx, "Updating configuration row", map[string]any{
			"branch_id":        row.BranchID,
			"component_id":     row.ComponentID,
			"configuration_id": row.ConfigID,
			"row_id":           row.ID,
		})

		updated, err := r.client.UpdateConfigRowRequest(row, []string{}).Send(ctx)
		if err != nil {
			return nil, fmt.Errorf("UpdateConfigRowRequest failed: %w", err)
		}

		// Log the returned row ID after update
		tflog.Info(ctx, "Updated configuration row", map[string]any{
			"branch_id":        updated.BranchID,
			"component_id":     updated.ComponentID,
			"configuration_id": updated.ConfigID,
			"row_id":           updated.ID,
		})

		return updated, nil
	})
}

// Delete removes the configuration row using the Keboola API.
func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.base.ExecuteDelete(ctx, req, resp, func(ctx context.Context, state Model) error {
		key, err := getKeyFromModel(ctx, state)
		if err != nil {
			return fmt.Errorf("getKeyFromModel failed: %w", err)
		}
		// Ensure the row ID is set from state.ID
		if !state.ID.IsNull() && state.ID.ValueString() != "" {
			key.ID = keboola.RowID(state.ID.ValueString())
		} else {
			return errRowIDMissing
		}

		_, err = r.client.DeleteConfigRowRequest(key).Send(ctx)
		if err != nil {
			return fmt.Errorf("DeleteConfigRowRequest failed: %w", err)
		}

		return nil
	})
}
