package utils

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// GetEmailStringValue is a function that extracts a string value from an openapi_types.Email.
// It marshals the email to JSON, trims any surrounding double quotes and escape characters,
// and returns the resulting string type.
func GetEmailStringValue(email openapi_types.Email) types.String {
	result := strings.Trim(string(email), `"`)
	result = strings.Trim(result, "\\\"")
	return TrimmedStringValue(result)
}

// TrimPrefixSuffix is a function that trims double quotes and escape characters
// from the beginning and end of a string. It returns the trimmed string.
func TrimPrefixSuffix(s string) string {
	result := strings.TrimPrefix(s, `"`)
	result = strings.TrimSuffix(result, `"`)
	result = strings.TrimPrefix(result, "\\\"")
	result = strings.TrimSuffix(result, "\\\"")
	return result
}

// TrimmedStringValue is a function that trims double quotes and escape characters
// from the beginning and end of a string. It returns the trimmed hashicorp wrapped string.
func TrimmedStringValue(s string) types.String {
	return types.StringValue(TrimPrefixSuffix(s))
}
