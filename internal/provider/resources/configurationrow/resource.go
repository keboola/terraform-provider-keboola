package configurationrow

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
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
	resp.TypeName = req.ProviderTypeName + "_component_configuration_row"
}

// Schema defines the Terraform schema for the resource.
func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a configuration row in Keboola. You must specify either 'configuration_fqn' or " +
			"all of 'branch_id', 'component_id', and 'configuration_id'.",
		MarkdownDescription: "Manages a configuration row in Keboola. You must specify either `configuration_fqn` or " +
			"all of `branch_id`, `component_id`, and `configuration_id`.",
		Attributes: map[string]schema.Attribute{
			"branch_id": schema.StringAttribute{
				Description:         "ID of the branch. Mutually exclusive with 'configuration_fqn'.",
				MarkdownDescription: "ID of the branch. Mutually exclusive with `configuration_fqn`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("configuration_fqn"),
					),
					stringvalidator.ExactlyOneOf(
						path.MatchRelative().AtParent().AtName("branch_id"),
						path.MatchRelative().AtParent().AtName("configuration_fqn"),
					),
				},
				// ForceNew: Changing branch_id requires recreation of the row resource.
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"component_id": schema.StringAttribute{
				Description:         "ID of the component. Mutually exclusive with 'configuration_fqn'.",
				MarkdownDescription: "ID of the component. Mutually exclusive with `configuration_fqn`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("configuration_fqn"),
					),
				},
				// ForceNew: Changing component_id requires recreation of the row resource.
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"configuration_id": schema.StringAttribute{
				Description:         "ID of the configuration. Mutually exclusive with 'configuration_fqn'.",
				MarkdownDescription: "ID of the configuration. Mutually exclusive with `configuration_fqn`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("configuration_fqn"),
					),
				},
				// ForceNew: Changing configuration_id requires recreation of the row resource.
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"configuration_fqn": schema.StringAttribute{
				Description: "Fully qualified name for the configuration row. " +
					"Required if not using branch_id/component_id/configuration_id.",
				Optional: true,
				// Validator ensures the value matches the pattern: integer/component_id/configuration_id
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						// Pattern: branch_id/component_id/configuration_id
						// - branch_id: one or more digits
						// - component_id: one or more non-slash characters
						// - configuration_id: one or more non-slash characters
						regexp.MustCompile(`^\d+/[^/]+/[^/]+$`),
						"configuration_fqn must be in the format 'branch_id/component_id/configuration_id', e.g. '123/ex-generic-v2/456'",
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
	_                                resource.ResourceWithImportState = &Resource{}
	errInvalidConfigurationFQNFormat                                  = errors.New(
		"invalid configuration_fqn format, expected branch_id/component_id/configuration_id",
	)
	errMissingIdentifiers = errors.New(
		"must provide either configuration_fqn or all of branch_id, component_id, configuration_id",
	)
	errInvalidBranchIDFormat = errors.New("invalid branch_id format, expected integer")
	errRowIDMissing          = errors.New("row ID is missing in state")
)

// ResourceNotFoundError represents a resource not found error that should be treated as a sentinel error.
type ResourceNotFoundError struct {
	ResourceName string
}

// Error implements the error interface for ResourceNotFoundError.
func (e *ResourceNotFoundError) Error() string {
	return "Not found: " + e.ResourceName
}

// isNotFoundError checks if the error is a 404 not found error from the Keboola API.
// This function detects the specific error pattern returned by the Keboola API for deleted resources.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	// Check for the specific error pattern from Keboola API
	// Error message contains: errCode: "notFound", httpCode: "404"
	errMsg := err.Error()

	return strings.Contains(errMsg, `errCode: "notFound"`) && strings.Contains(errMsg, `httpCode: "404"`)
}

const configurationFQNPartsCount = 3 // for magic number check

// getKeyFromModel returns a keboola.ConfigRowKey from the model.
// It uses configuration_fqn if set, otherwise the individual IDs.
func getKeyFromModel(ctx context.Context, m Model) (keboola.ConfigRowKey, error) {
	if !m.ConfigurationFQN.IsNull() && m.ConfigurationFQN.ValueString() != "" {
		configurationFQN := m.ConfigurationFQN.ValueString()

		key, ok, err := parseConfigurationFQN(configurationFQN)
		if err != nil {
			return keboola.ConfigRowKey{}, fmt.Errorf("%w: %w", errInvalidConfigurationFQNFormat, err)
		}

		if ok {
			tflog.Info(ctx, "Parsed configuration_fqn", map[string]any{"configuration_fqn": configurationFQN, "key": key})

			return key, nil
		}

		return keboola.ConfigRowKey{}, errInvalidConfigurationFQNFormat
	}

	if m.BranchID.IsNull() || m.ComponentID.IsNull() || m.ConfigID.IsNull() {
		return keboola.ConfigRowKey{}, errMissingIdentifiers
	}

	branchID, err := strconv.ParseInt(m.BranchID.ValueString(), 10, 64)
	if err != nil {
		return keboola.ConfigRowKey{}, fmt.Errorf("invalid branch_id: %w", err)
	}

	return keboola.ConfigRowKey{
		BranchID:    keboola.BranchID(branchID),
		ComponentID: keboola.ComponentID(m.ComponentID.ValueString()),
		ConfigID:    keboola.ConfigID(m.ConfigID.ValueString()),
	}, nil
}

// parseConfigurationFQN parses configuration_fqn with 3 parts (branch_id/component_id/config_id).
func parseConfigurationFQN(configurationFQN string) (keboola.ConfigRowKey, bool, error) {
	parts := strings.Split(configurationFQN, "/")
	if len(parts) != configurationFQNPartsCount {
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

		// Use GetConfigRowModelID for consistent logging of the row identifier
		tflog.Info(ctx, "Creating configuration row", map[string]any{
			"configuration_fqn": GetConfigRowModelID(&plan),
		})

		created, err := r.client.CreateConfigRowRequest(row).Send(ctx)
		if err != nil {
			return nil, fmt.Errorf("CreateConfigRowRequest failed: %w", err)
		}

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

		// Use GetConfigRowModelID for consistent logging of the row identifier
		tflog.Info(ctx, "Reading configuration row", map[string]any{
			"configuration_fqn": GetConfigRowModelID(&state),
			"row_id":            key.ID,
		})

		row, err := r.client.GetConfigRowRequest(key).Send(ctx)
		if err != nil {
			// Check if this is a 404 error indicating the resource was already deleted
			// This is a common scenario during destroy operations when resources are deleted externally
			if isNotFoundError(err) {
				tflog.Info(ctx, "Configuration row not found (likely already deleted), treating as deleted", map[string]any{
					"configuration_fqn": GetConfigRowModelID(&state),
					"row_id":            key.ID,
				})
				// Return a sentinel error to indicate the resource should be treated as deleted
				return nil, &ResourceNotFoundError{ResourceName: "configuration row"}
			}

			return nil, fmt.Errorf("GetConfigRowRequest failed: %w", err)
		}

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

		// Use GetConfigRowModelID for consistent logging of the row identifier
		tflog.Info(ctx, "Updating configuration row", map[string]any{
			"configuration_fqn": GetConfigRowModelID(&plan),
			"row_id":            row.ID,
		})

		updated, err := r.client.UpdateConfigRowRequest(row, []string{}).Send(ctx)
		if err != nil {
			return nil, fmt.Errorf("UpdateConfigRowRequest failed: %w", err)
		}

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

		// Use GetConfigRowModelID for consistent logging of the row identifier
		tflog.Info(ctx, "Deleting configuration row", map[string]any{
			"configuration_fqn": GetConfigRowModelID(&state),
			"row_id":            key.ID,
		})

		_, err = r.client.DeleteConfigRowRequest(key).Send(ctx)
		if err != nil {
			// Check if this is a 404 error indicating the resource was already deleted
			// This is a common scenario during destroy operations when resources are deleted externally
			if isNotFoundError(err) {
				tflog.Info(ctx, "Configuration row not found during delete (likely already deleted), treating as successful",
					map[string]any{
						"configuration_fqn": GetConfigRowModelID(&state),
						"row_id":            key.ID,
					})
				// Return a sentinel error to indicate the deletion was successful
				return &ResourceNotFoundError{ResourceName: "configuration row"}
			}

			return fmt.Errorf("DeleteConfigRowRequest failed: %w", err)
		}

		return nil
	})
}

// ImportState imports an existing configuration row resource by compound ID.
// The import ID should be in format "branch_id/component_id/configuration_id/row_id".
// If branch_id is omitted, default branch will be used (e.g., "component_id/configuration_id/row_id").
func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Info(ctx, "Importing configuration row resource", map[string]any{
		"import_id": req.ID,
	})

	// Parse the import ID
	parts := strings.Split(req.ID, "/")

	var branchID, componentID, configID, rowID string

	switch len(parts) {
	case 3:
		// Format: component_id/configuration_id/row_id (use default branch)
		componentID = parts[0]
		configID = parts[1]
		rowID = parts[2]
	case 4:
		// Format: branch_id/component_id/configuration_id/row_id
		branchID = parts[0]
		componentID = parts[1]
		configID = parts[2]
		rowID = parts[3]
	default:
		resp.Diagnostics.AddError(
			"Invalid Import ID Format",
			fmt.Sprintf("Expected import ID format: 'branch_id/component_id/configuration_id/row_id' or 'component_id/configuration_id/row_id', got: %s", req.ID),
		)
		return
	}

	// Set the parsed values in state
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), rowID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("component_id"), componentID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("configuration_id"), configID)...)

	if branchID != "" {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("branch_id"), branchID)...)
	}

	tflog.Info(ctx, "Parsed import ID", map[string]any{
		"branch_id":        branchID,
		"component_id":     componentID,
		"configuration_id": configID,
		"row_id":           rowID,
	})
}
