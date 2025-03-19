package configuration

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/keboola/go-client/pkg/keboola"
	"github.com/keboola/go-utils/pkg/orderedmap"
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
	tfModel.Version = types.Int64Value(int64(apiModel.Config.Version))
	tfModel.IsDisabled = types.BoolValue(apiModel.Config.IsDisabled)
	tfModel.Created = types.StringValue(apiModel.Config.Created.UTC().String())

	// Set the compound ID
	tfModel.ID = types.StringValue(GetConfigModelId(tfModel))

	// Map configuration content
	if apiModel.Config.Content != nil {
		contentBytes, err := json.Marshal(apiModel.Config.Content)
		if err != nil {
			diags.AddWarning(
				"Error serializing configuration content",
				"Could not serialize configuration content: "+err.Error(),
			)
			tfModel.Content = types.StringValue("{}")
		}
		content := string(contentBytes)
		if m.isTest {
			// Clean up newlines for consistent comparison
			contentBytes, _ = json.MarshalIndent(apiModel.Config.Content, "\t\t\t\t\t", "\t")
			content = string(contentBytes)
		}

		tfModel.Content = types.StringValue(content)
	} else {
		tfModel.Content = types.StringValue("{}")
	}

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
			return nil, fmt.Errorf("could not parse configuration content: %w", err)
		}
	}

	// Process any rows
	var rows []*keboola.ConfigRow
	var rowsSortOrder []string
	var err error

	if !tfModel.Rows.IsNull() {
		// Extract row models
		rowModels, diags := m.RowHandler.ExtractChildModels(ctx, tfModel)
		if diags.HasError() {
			return nil, fmt.Errorf("failed to extract plan row models: %v", diags)
		}

		stateRowModels, diags := m.RowHandler.ExtractChildModels(ctx, stateModel)
		if diags.HasError() {
			return nil, fmt.Errorf("failed to extract state row models: %v", diags)
		}

		// Map rows to API models
		rows, err = m.RowHandler.MapChildModelsToAPI(ctx, rowModels)
		if err != nil {
			return nil, fmt.Errorf("failed to map row models to API: %w", err)
		}

		// Get row sort order
		rowsSortOrder = m.RowHandler.GetRowsSortOrder(rowModels, stateRowModels)
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

	tflog.Debug(ctx, "Created API model from Terraform model", map[string]interface{}{
		"config_id":    config.Config.ID,
		"component_id": config.Config.ComponentID,
		"rows_count":   len(rows),
		"sort_order":   rowsSortOrder,
	})

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

	rows := []*RowModel{}
	if newModel.Rows.IsUnknown() {
		if oldModel == nil {
			newModel.Rows, diags = types.ListValue(types.StringType, []attr.Value{})
		} else {
			oldModel.Rows.ElementsAs(ctx, &rows, false)
			for _, row := range rows {
				if row.ChangeDescription.IsUnknown() {
					row.ChangeDescription = types.StringValue("Created by Keboola Terraform Provider")
				}
			}
		}
	} else {
		newModel.Rows.ElementsAs(ctx, &rows, false)
		for _, row := range rows {
			if oldModel == nil {
				row.ChangeDescription = types.StringValue("Created by Keboola Terraform Provider")
			} else {
				row.ChangeDescription = types.StringValue("Updated by Keboola Terraform Provider")
			}
		}
	}

	rowType := newModel.Rows.Type(ctx)
	updatedRows, rowDiags := types.ListValueFrom(ctx, rowType.(types.ListType).ElemType, rows)
	diags.Append(rowDiags...)
	if !rowDiags.HasError() {
		newModel.Rows = updatedRows
	}

	return diags
}
