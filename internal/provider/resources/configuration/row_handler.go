package configuration

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keboola/go-client/pkg/keboola"
	"github.com/keboola/go-utils/pkg/orderedmap"
)

// DefaultConfigRowHandler implements ConfigRowHandler interface.
type DefaultConfigRowHandler struct {
	// Pointer to the parent resource's client
	Client *keboola.AuthorizedAPI
	isTest bool
}

// ExtractChildModels extracts row models from the parent configuration model.
func (h *DefaultConfigRowHandler) ExtractChildModels(
	ctx context.Context,
	parent ConfigModel,
) ([]RowModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var rows []RowModel

	if parent.Rows.IsNull() {
		return rows, diags
	}

	diags = parent.Rows.ElementsAs(ctx, &rows, false)

	return rows, diags
}

// MapChildModelsToAPI converts row models to API models.
func (h *DefaultConfigRowHandler) MapChildModelsToAPI(
	_ context.Context,
	rowModels []RowModel,
) ([]*keboola.ConfigRow, error) {
	rows := make([]*keboola.ConfigRow, 0, len(rowModels))

	for _, rowModel := range rowModels {
		// Parse row content
		rowContent := orderedmap.New()
		if !rowModel.Content.IsNull() && !rowModel.Content.IsUnknown() {
			err := rowContent.UnmarshalJSON([]byte(rowModel.Content.ValueString()))
			if err != nil {
				return nil, fmt.Errorf("failed to parse row content: %w", err)
			}
		}

		// Parse row state
		rowState := orderedmap.New()
		if !rowModel.State.IsNull() && !rowModel.State.IsUnknown() {
			err := rowState.UnmarshalJSON([]byte(rowModel.State.ValueString()))
			if err != nil {
				return nil, fmt.Errorf("failed to parse row state: %w", err)
			}
		}

		// Create the API row model
		row := &keboola.ConfigRow{ //nolint: exhaustruct
			Name:              rowModel.Name.ValueString(),
			Description:       rowModel.Description.ValueString(),
			ChangeDescription: rowModel.ChangeDescription.ValueString(),
			IsDisabled:        rowModel.IsDisabled.ValueBool(),
			Content:           rowContent,
			State:             rowState,
		}

		// Set ID if it exists (for updates)
		if !rowModel.ID.IsNull() && !rowModel.ID.IsUnknown() {
			row.ID = keboola.RowID(rowModel.ID.ValueString())
		}

		rows = append(rows, row)
	}

	return rows, nil
}

// ProcessAPIChildModels processes row API models after API operations.
func (h *DefaultConfigRowHandler) ProcessAPIChildModels(
	ctx context.Context,
	parent *ConfigModel,
	apiRows []*keboola.ConfigRow,
) diag.Diagnostics {
	var diags diag.Diagnostics

	if len(apiRows) == 0 {
		return diags
	}

	// Build map of rows by ID for quick lookup
	rowMap := make(map[string]*keboola.ConfigRow)
	for _, row := range apiRows {
		rowMap[row.ID.String()] = row
	}

	// Process the rows into Terraform models
	var originalRows []RowModel

	// If parent already has rows defined, preserve their order
	if !parent.Rows.IsNull() {
		var existingRows []RowModel
		diags = parent.Rows.ElementsAs(ctx, &existingRows, false)
		if diags.HasError() {
			return diags
		}

		// Process existing rows first, maintaining their order
		for _, existingRow := range existingRows {
			rowID := existingRow.ID.String()
			if apiRow, ok := rowMap[rowID]; ok {
				// Row exists in API response
				rowModel := createRowModelFromAPI(apiRow, h.isTest)
				originalRows = append(originalRows, rowModel)

				// Remove from map to track processed rows
				delete(rowMap, rowID)
			}
		}
	}

	// Add any remaining rows from the API that weren't in the existing state
	for _, apiRow := range apiRows {
		rowID := apiRow.ID.String()
		if _, ok := rowMap[rowID]; ok {
			rowModel := createRowModelFromAPI(apiRow, h.isTest)
			originalRows = append(originalRows, rowModel)
		}
	}

	// Create a new list of rowModels
	rowType := parent.Rows.Type(ctx)
	elemType, ok := rowType.(types.ListType)
	if !ok {
		diags.AddError(
			"Error processing API child models",
			"Expected rows to be a list type",
		)

		return diags
	}

	updatedRows, rowDiags := types.ListValueFrom(ctx, elemType.ElemType, originalRows)
	diags.Append(rowDiags...)
	if !rowDiags.HasError() {
		parent.Rows = updatedRows
	}

	return diags
}

// Helper function to create a RowModel from API row.
func createRowModelFromAPI(apiRow *keboola.ConfigRow, isTest bool) RowModel {
	rowModel := RowModel{ //nolint: exhaustruct
		ID:                types.StringValue(apiRow.ID.String()),
		Name:              types.StringValue(apiRow.Name),
		IsDisabled:        types.BoolValue(apiRow.IsDisabled),
		Description:       types.StringValue(apiRow.Description),
		ChangeDescription: types.StringValue(apiRow.ChangeDescription),
		Version:           types.Int64Value(int64(apiRow.Version)),
	}

	// Handle row state and content
	rowModel.State = formatOrderedMapField(apiRow.State, isTest)
	rowModel.Content = formatOrderedMapField(apiRow.Content, isTest)

	return rowModel
}

// formatOrderedMapField handles formatting an OrderedMap for a row field.
func formatOrderedMapField(data *orderedmap.OrderedMap, isTest bool) types.String {
	// Default to empty JSON object
	if data == nil {
		return types.StringValue("{}")
	}

	// Marshal the data
	bytes, err := json.Marshal(data)
	if err != nil {
		return types.StringValue("{}")
	}

	formattedValue := string(bytes)

	// Apply test mode formatting if requested
	if isTest {
		indentedBytes, err := json.MarshalIndent(data, "\t\t\t\t\t\t\t", "\t")
		if err == nil {
			formattedValue = string(indentedBytes)
		}
		// If formatting fails in test mode, we still have the original formatting
	}

	return types.StringValue(formattedValue)
}

// GetRowsSortOrder returns a slice of row IDs for specifying sort order.
func (h *DefaultConfigRowHandler) GetRowsSortOrder(rowModels, stateModels []RowModel) []string {
	// We do not change rows when plan has less rows than state (we are deleting rows)
	if len(rowModels) < len(stateModels) {
		return []string{}
	}

	var rowsSortOrder []string
	for _, rowModel := range rowModels {
		if !rowModel.ID.IsNull() && !rowModel.ID.IsUnknown() {
			rowsSortOrder = append(rowsSortOrder, rowModel.ID.ValueString())
		}
	}

	return rowsSortOrder
}
