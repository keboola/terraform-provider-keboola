package scheduler

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/keboola/go-client/pkg/keboola"

	"github.com/keboola/terraform-provider-keboola/internal/provider/abstraction"
	"github.com/keboola/terraform-provider-keboola/internal/providermodels"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource = &Resource{
		base:   abstraction.BaseResource[SchedulerModel, *keboola.Schedule]{},
		client: nil,
		isTest: false,
	}
	_ resource.ResourceWithConfigure = &Resource{
		base:   abstraction.BaseResource[SchedulerModel, *keboola.Schedule]{},
		client: nil,
		isTest: false,
	}
)

// NewResource is a helper function to simplify the provider implementation.
func NewResource() *Resource {
	return &Resource{
		base:   abstraction.BaseResource[SchedulerModel, *keboola.Schedule]{},
		client: nil,
		isTest: false,
	}
}

// Resource is the scheduler resource implementation.
type Resource struct {
	// Base functionality with scheduler model specifics
	base abstraction.BaseResource[SchedulerModel, *keboola.Schedule]

	// Direct access to the API client for specific operations
	client *keboola.AuthorizedAPI
	isTest bool
}

// Metadata returns the resource type name.
func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scheduler"
}

// Schema defines the schema for the resource.
func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manages scheduler configurations.",
		MarkdownDescription: "Manages scheduler configurations.",
		Blocks:              map[string]schema.Block{},
		DeprecationMessage:  "",
		Version:             1,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "Unique identifier of the scheduler.",
				MarkdownDescription: "Unique identifier of the scheduler.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"config_id": schema.StringAttribute{
				Description:         "ID of the configuration that is scheduled to run.",
				MarkdownDescription: "ID of the configuration that is scheduled to run.",
				Required:            true,
			},
			"configuration_version": schema.StringAttribute{
				Description:         "Version of the configuration to run.",
				MarkdownDescription: "Version of the configuration to run.",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *Resource) Configure(
	_ context.Context,
	req resource.ConfigureRequest,
	_ *resource.ConfigureResponse,
) {
	// Return silently if provider data is not available (yet)
	if req.ProviderData == nil {
		return
	}

	// Get the provider data - ignoring the type assertion success
	providerData, _ := req.ProviderData.(*providermodels.ProviderData)
	r.client = providerData.Client
	r.isTest = os.Getenv("TF_ACC") != "" //nolint: forbidigo

	// Set up the mapper
	r.base.Mapper = &SchedulerMapper{
		isTest: r.isTest,
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Creating scheduler resource")

	// Use the base resource abstraction for Create
	r.base.ExecuteCreate(ctx, req, resp, func(ctx context.Context, plan SchedulerModel) (*keboola.Schedule, error) {
		// Note: The Keboola API doesn't have a direct CreateScheduleRequest method.
		// Validate required fields
		if plan.ConfigID.IsNull() || plan.ConfigID.ValueString() == "" {
			return nil, fmt.Errorf("config_id is required for scheduler creation")
		}

		// Get configuration version if specified
		configVersionID := ""
		if !plan.ConfigurationVersion.IsNull() {
			configVersionID = plan.ConfigurationVersion.ValueString()
		}

		// Attempt to activate/create the schedule
		configID := keboola.ConfigID(plan.ConfigID.ValueString())
		schedule, err := r.client.ActivateScheduleRequest(configID, configVersionID).Send(ctx)
		if err != nil {
			return nil, fmt.Errorf("could not create scheduler using ActivateScheduleRequest: %w", err)
		}

		tflog.Info(ctx, "Created schedule using ActivateScheduleRequest", map[string]interface{}{
			"schedule_id": schedule.ID,
			"config_id":   schedule.ConfigID,
			"schedule":    schedule,
		})

		return schedule, nil
	})
}

// Read resource information.
func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Reading scheduler resource")

	// Use the base resource abstraction for Read
	r.base.ExecuteRead(ctx, req, resp, func(ctx context.Context, state SchedulerModel) (*keboola.Schedule, error) {
		// Get all schedules and find the one with matching ID
		schedule, err := r.client.GetScheduleRequest(keboola.ScheduleKey{ID: keboola.ScheduleID(state.ID.ValueString())}).Send(ctx)
		if err != nil {
			return nil, fmt.Errorf("could not get schedule: %w", err)
		}

		return schedule, nil
	})
}

// Update updates the resource.
func (r *Resource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	tflog.Info(ctx, "Updating scheduler resource")

	// Use the base resource abstraction for Update
	r.base.ExecuteUpdate(
		ctx,
		req,
		resp,
		func(ctx context.Context, state, plan SchedulerModel) (
			*keboola.Schedule,
			error,
		) {
			// Preserve scheduler ID from state
			plan.ID = state.ID

			// Execute the update operation using the mapper
			apiModel, err := r.base.Mapper.MapTerraformToAPI(ctx, state, plan)
			if err != nil {
				return nil, fmt.Errorf("failed to map Terraform model to API: %w", err)
			}

			// Note: The Keboola API doesn't have a direct UpdateScheduleRequest.
			// For now, we'll handle the active state if it changed
			// If plan is active but state wasn't, activate the schedule
			configVersionID := ""
			if !plan.ConfigurationVersion.IsNull() {
				configVersionID = plan.ConfigurationVersion.ValueString()
			}

			resSchedule, err := r.client.ActivateScheduleRequest(
				apiModel.ConfigID,
				configVersionID,
			).Send(ctx)
			if err != nil {
				return nil, fmt.Errorf("could not activate scheduler: %w", err)
			}
			return resSchedule, nil
		})
}

// Delete deletes the resource.
func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Deleting scheduler resource")

	// Use the base resource abstraction for Delete
	r.base.ExecuteDelete(ctx, req, resp, func(ctx context.Context, state SchedulerModel) error {
		// Create key from model
		key := keboola.ScheduleKey{
			ID: keboola.ScheduleID(state.ID.ValueString()),
		}

		// Delete the scheduler
		err := r.client.DeleteScheduleRequest(key).SendOrErr(ctx)
		if err != nil {
			return fmt.Errorf("could not delete scheduler: %w", err)
		}

		return nil
	})
}
