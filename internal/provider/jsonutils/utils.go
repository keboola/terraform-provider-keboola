package jsonutils

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/keboola/go-utils/pkg/orderedmap"
)

// ParseJSON parses a JSON string into an orderedmap.
func ParseJSON(jsonStr types.String) (*orderedmap.OrderedMap, error) {
	if jsonStr.IsNull() || jsonStr.IsUnknown() {
		return orderedmap.New(), nil
	}

	contentMap := orderedmap.New()

	err := contentMap.UnmarshalJSON([]byte(jsonStr.ValueString()))
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return contentMap, nil
}

// SerializeJSON serializes an orderedmap to a JSON string.
// Uses json.Marshal for consistent minified JSON output without newlines,
// matching the behavior of the configuration resource.
func SerializeJSON(contentMap *orderedmap.OrderedMap) (types.String, error) {
	if contentMap == nil {
		return types.StringValue("{}"), nil
	}

	// Use json.Marshal for consistent minified JSON output without newlines
	// This matches the behavior in configuration/mapper.go processConfigContent
	contentBytes, err := json.Marshal(contentMap)
	if err != nil {
		return types.StringNull(), fmt.Errorf("failed to serialize JSON: %w", err)
	}

	return types.StringValue(string(contentBytes)), nil
}
