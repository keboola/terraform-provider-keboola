package configuration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keboola/go-utils/pkg/orderedmap"
	"github.com/keboola/keboola-sdk-go/pkg/keboola"
)

// Static errors.
var (
	ErrExtractPlanRowModels  = errors.New("failed to extract plan row models")
	ErrExtractStateRowModels = errors.New("failed to extract state row models")
	ErrParseConfigContent    = errors.New("could not parse configuration content")
	ErrMapRowModelsToAPI     = errors.New("failed to map row models to API")
)

// ConfigMapper implements ResourceMapper for configuration resources.
type ConfigMapper struct {
	// Nested resource handler for rows
	RowHandler *DefaultConfigRowHandler
	isTest     bool
}

// MapAPIToTerraform converts a Keboola API config model to a Terraform model.
func (m *ConfigMapper) MapAPIToTerraform(
	ctx context.Context,
	apiModel *keboola.ConfigWithRows,
	tfModel *ConfigModel,
) diag.Diagnostics {
	var diags diag.Diagnostics

	// Map basic properties
	tfModel.ConfigID = types.StringValue(apiModel.Config.ID.String())
	tfModel.BranchID = types.Int64Value(int64(apiModel.Config.BranchID))
	tfModel.ComponentID = types.StringValue(string(apiModel.Config.ComponentID))
	tfModel.Name = types.StringValue(apiModel.Config.Name)
	tfModel.Description = types.StringValue(apiModel.Config.Description)
	tfModel.ChangeDescription = types.StringValue(apiModel.Config.ChangeDescription)
	tfModel.IsDeleted = types.BoolValue(apiModel.Config.IsDeleted)
	tfModel.IsDisabled = types.BoolValue(apiModel.Config.IsDisabled)
	tfModel.Created = types.StringValue(apiModel.Config.Created.UTC().String())

	// Set the compound ID
	tfModel.ID = types.StringValue(GetConfigModelID(tfModel))

	// Map configuration content
	tfModel.Content = types.StringValue("{}")
	processConfigContent(apiModel.Config.Content, &tfModel.Content, m.isTest, &diags)

	// Process rows if they exist
	if len(apiModel.Rows) > 0 {
		rowDiags := m.RowHandler.ProcessAPIChildModels(ctx, tfModel, apiModel.Rows)
		diags.Append(rowDiags...)
	}

	return diags
}

// MapTerraformToAPI converts a Terraform model to a Keboola API model.
func (m *ConfigMapper) MapTerraformToAPI(
	ctx context.Context,
	stateModel ConfigModel,
	tfModel ConfigModel,
) (*keboola.ConfigWithRows, error) {
	// Create API key structure
	key := keboola.ConfigKey{
		BranchID:    keboola.BranchID(tfModel.BranchID.ValueInt64()),
		ComponentID: keboola.ComponentID(tfModel.ComponentID.ValueString()),
		ID:          keboola.ConfigID(tfModel.ConfigID.ValueString()),
	}

	// Set optional fields if provided
	if !tfModel.ConfigID.IsNull() && !tfModel.ConfigID.IsUnknown() {
		key.ID = keboola.ConfigID(tfModel.ConfigID.ValueString())
	}

	if !tfModel.BranchID.IsNull() && !tfModel.BranchID.IsUnknown() {
		key.BranchID = keboola.BranchID(tfModel.BranchID.ValueInt64())
	}

	// Parse configuration content
	contentMap := orderedmap.New()
	if !tfModel.Content.IsNull() && !tfModel.Content.IsUnknown() {
		err := contentMap.UnmarshalJSON([]byte(tfModel.Content.ValueString()))
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrParseConfigContent, err)
		}
	}

	// Process any rows
	rows, rowsSortOrder, err := m.processRows(ctx, tfModel, stateModel)
	if err != nil {
		return nil, err
	}

	// Create the full API model
	config := &keboola.ConfigWithRows{
		Config: &keboola.Config{
			ConfigKey:         key,
			Name:              tfModel.Name.ValueString(),
			Description:       tfModel.Description.ValueString(),
			ChangeDescription: tfModel.ChangeDescription.ValueString(),
			Content:           contentMap,
			IsDisabled:        tfModel.IsDisabled.ValueBool(),
			RowsSortOrder:     rowsSortOrder,
		},
		Rows: rows,
	}

	return config, nil
}

// ValidateTerraformModel validates a Terraform model for consistency and constraints.
func (m *ConfigMapper) ValidateTerraformModel(
	ctx context.Context,
	oldModel *ConfigModel,
	newModel *ConfigModel,
) diag.Diagnostics {
	var diags diag.Diagnostics

	// If we're updating (oldModel != nil), validate immutable fields
	if oldModel != nil {
		if !newModel.ComponentID.IsUnknown() && oldModel.ComponentID != newModel.ComponentID {
			diags.AddError(
				"Error updating configuration",
				"Cannot change component_id after configuration is created",
			)
		}

		if !newModel.BranchID.IsUnknown() && oldModel.BranchID != newModel.BranchID {
			diags.AddError(
				"Error updating configuration",
				"Cannot change branch_id after configuration is created",
			)
		}

		if !newModel.ConfigID.IsUnknown() && oldModel.ConfigID != newModel.ConfigID {
			diags.AddError(
				"Error updating configuration",
				"Cannot change configuration_id after configuration is created",
			)
		}
	}

	// Validate content JSON
	if !newModel.Content.IsNull() && !newModel.Content.IsUnknown() {
		contentMap := orderedmap.New()
		err := contentMap.UnmarshalJSON([]byte(newModel.Content.ValueString()))
		if err != nil {
			diags.AddError(
				"Error validating configuration",
				"Could not parse configuration JSON: "+err.Error(),
			)
		}
	}

	// Set defaults for required fields that can have default
	if newModel.ChangeDescription.IsUnknown() {
		if oldModel == nil {
			newModel.ChangeDescription = types.StringValue("Created by Keboola Terraform Provider")
		} else {
			newModel.ChangeDescription = types.StringValue("Updated by Keboola Terraform Provider")
		}
	}

	if newModel.Content.IsUnknown() {
		newModel.Content = types.StringValue("{}")
	}

	// Initialize rows
	rows := []*RowModel{}

	// Process unknown rows
	if newModel.Rows.IsUnknown() {
		return processUnknownRows(ctx, oldModel, newModel, diags)
	}

	// Process known rows
	newModel.Rows.ElementsAs(ctx, &rows, false)
	setDefaultRowChangeDescriptions(rows, oldModel == nil)

	// Update rows in the model
	return updateRowsInModel(ctx, newModel, rows, diags)
}

// processRows processes rows and returns them along with their sort order.
func (m *ConfigMapper) processRows(
	ctx context.Context,
	tfModel ConfigModel,
	stateModel ConfigModel,
) ([]*keboola.ConfigRow, []string, error) {
	// Return empty arrays if Rows is null
	if tfModel.Rows.IsNull() {
		return []*keboola.ConfigRow{}, []string{}, nil
	}

	// Extract row models
	rowModels, diags := m.RowHandler.ExtractChildModels(ctx, tfModel)
	if diags.HasError() {
		return nil, nil, fmt.Errorf("%w: %v", ErrExtractPlanRowModels, diags)
	}

	stateRowModels, diags := m.RowHandler.ExtractChildModels(ctx, stateModel)
	if diags.HasError() {
		return nil, nil, fmt.Errorf("%w: %v", ErrExtractStateRowModels, diags)
	}

	// Map rows to API models
	rows, err := m.RowHandler.MapChildModelsToAPI(ctx, rowModels)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", ErrMapRowModelsToAPI, err)
	}

	// Get row sort order
	rowsSortOrder := m.RowHandler.GetRowsSortOrder(rowModels, stateRowModels)

	return rows, rowsSortOrder, nil
}

// processConfigContent handles marshaling the configuration content.
func processConfigContent(
	content *orderedmap.OrderedMap,
	targetContent *types.String,
	isTest bool,
	diags *diag.Diagnostics,
) {
	// Skip if content is nil
	if content == nil {
		return
	}

	// Marshal the content
	contentBytes, err := json.Marshal(content)
	if err != nil {
		diags.AddWarning(
			"Error serializing configuration content",
			"Could not serialize configuration content: "+err.Error(),
		)

		return
	}

	// Set regular content
	*targetContent = types.StringValue(string(contentBytes))

	// Apply test mode formatting if needed
	if isTest {
		formatContentForTestMode(content, targetContent, diags)
	}
}

// formatContentForTestMode handles the special formatting for test mode.
func formatContentForTestMode(content *orderedmap.OrderedMap, targetContent *types.String, diags *diag.Diagnostics) {
	indentedBytes, indentErr := json.MarshalIndent(content, "\t\t\t\t\t", "\t")
	if indentErr != nil {
		diags.AddWarning(
			"Error formatting configuration content in test mode",
			"Could not format configuration content: "+indentErr.Error(),
		)

		return
	}

	*targetContent = types.StringValue(string(indentedBytes))
}

// processUnknownRows handles the case when the rows are unknown.
func processUnknownRows(
	ctx context.Context,
	oldModel *ConfigModel,
	newModel *ConfigModel,
	diags diag.Diagnostics,
) diag.Diagnostics {
	// If no old model, set empty list
	if oldModel == nil {
		var listDiags diag.Diagnostics
		newModel.Rows, listDiags = types.ListValue(types.StringType, []attr.Value{})
		diags.Append(listDiags...)

		return diags
	}

	// Use old model's rows with default change description
	rows := []*RowModel{}
	oldModel.Rows.ElementsAs(ctx, &rows, false)

	// Set default change descriptions
	for _, row := range rows {
		if row.ChangeDescription.IsUnknown() {
			row.ChangeDescription = types.StringValue("Created by Keboola Terraform Provider")
		}
	}

	// Update rows in the model
	return updateRowsInModel(ctx, newModel, rows, diags)
}

// setDefaultRowChangeDescriptions sets default change descriptions for rows.
func setDefaultRowChangeDescriptions(rows []*RowModel, isNew bool) {
	defaultDesc := "Created by Keboola Terraform Provider"
	if !isNew {
		defaultDesc = "Updated by Keboola Terraform Provider"
	}

	for _, row := range rows {
		row.ChangeDescription = types.StringValue(defaultDesc)
	}
}

// updateRowsInModel updates the rows in the model with the provided rows.
func updateRowsInModel(
	ctx context.Context,
	model *ConfigModel,
	rows []*RowModel,
	diags diag.Diagnostics,
) diag.Diagnostics {
	rowType := model.Rows.Type(ctx)
	elemType, ok := rowType.(types.ListType)
	if !ok {
		diags.AddError(
			"Error validating configuration",
			"Expected rows to be a list type",
		)

		return diags
	}

	updatedRows, rowDiags := types.ListValueFrom(ctx, elemType.ElemType, rows)
	diags.Append(rowDiags...)
	if !rowDiags.HasError() {
		model.Rows = updatedRows
	}

	return diags
}
