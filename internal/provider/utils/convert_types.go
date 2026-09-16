package utils

import (
	"context"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
)

func WorkflowRuleSchemaPointer(v []client.WorkflowRuleSchema) *[]client.WorkflowRuleSchema {
	if v == nil {
		return nil
	}

	return &v
}

// StringPointer takes a string 'v' as input and returns a pointer to that string.
func StringPointer(v string) *string {
	return &v
}

// StringValue takes a pointer to a string 'v' as input and returns the string value it points to.
// If the pointer is nil, it returns an empty string.
func StringValue(v *string) string {
	if v != nil {
		return *v
	}

	return ""
}

// StringSlicePointer takes a string 'v' as input and returns a pointer to that string.
func StringSlicePointer(v []string) *[]string {
	if v == nil {
		return nil
	}

	return &v
}

// StringSliceValue takes a pointer to a string 'v' as input and returns the string value it points to.
// If the pointer is nil, it returns an empty string.
func StringSliceValue(v *[]string) []string {
	if v != nil {
		return *v
	}

	return nil
}

// BoolPointer takes a bool 'v' as input and returns a pointer to that bool.
func BoolPointer(v bool) *bool {
	return &v
}

// BoolValue takes a pointer to a bool 'v' as input and returns the bool value it points to.
// If the pointer is nil, it returns false.
func BoolValue(v *bool) bool {
	if v != nil {
		return *v
	}
	return false
}

// BoolOrDefault returns v's underlying bool if it is known (explicitly set
// by the user, or already resolved for the plan), otherwise it returns def.
//
// This is primarily useful when building a Create request body for an
// Optional+Computed attribute: on create there is no prior state, so an
// unset attribute stays Unknown in the plan, and this is where the
// attribute's documented default should be applied instead. On Update, a
// UseStateForUnknown plan modifier on the attribute already carries the
// previous state value into the plan when it's left unset in config, so
// callers building the Update body can read the plan value directly without
// this fallback.
func BoolOrDefault(v types.Bool, def bool) bool {
	if v.IsUnknown() || v.IsNull() {
		return def
	}
	return v.ValueBool()
}

// BoolPtrOrDefault is the *bool variant of BoolOrDefault, for API fields that
// are themselves optional.
func BoolPtrOrDefault(v types.Bool, def bool) *bool {
	value := BoolOrDefault(v, def)
	return &value
}

// IntPointer takes an int 'v' as input and returns a pointer to that int.
func IntPointer(v int) *int {
	return new(v)
}

// IntValue takes a pointer to an int 'v' as input and returns the int value it points to.
// If the pointer is nil, it returns 0.
func IntValue(v *int) int {
	if v != nil {
		return *v
	}
	return 0
}

// Int32Value takes a pointer to an int32 'v' as input and returns the int32 value it points to.
// If the pointer is nil, it returns 0.
func Int32Value(v *int32) int32 {
	if v != nil {
		return *v
	}
	return 0
}

// Int64Value takes a pointer to an int64 'v' as input and returns the int64 value it points to.
// If the pointer is nil, it returns 0.
func Int64Value(v *int64) int64 {
	if v != nil {
		return *v
	}
	return 0
}

// Float32Pointer takes a float32 'v' as input and returns a pointer to that float32.
func Float32Pointer(v float32) *float32 {
	return new(v)
}

// Float32Value takes a pointer to a float32 'v' as input and returns the float32 value it points to.
// If the pointer is nil, it returns 0.0.
func Float32Value(v *float32) float32 {
	if v != nil {
		return *v
	}
	return 0.0
}

// Float64Value takes a pointer to a float64 'v' as input and returns the float64 value it points to.
// If the pointer is nil, it returns 0.0.
func Float64Value(v *float64) float64 {
	if v != nil {
		return *v
	}
	return 0.0
}

func IdParamsSchemaSlicePointer(v []client.IdParamsSchema) *[]client.IdParamsSchema {
	return &v
}

func IdParamsSchemaSliceValue(v *[]client.IdParamsSchema) []client.IdParamsSchema {
	if v != nil {
		return *v
	}

	return nil
}

func GetStringSet(data *[]string) (types.Set, diag.Diagnostics) {
	if data == nil || len(*data) == 0 {
		return types.SetNull(types.StringType), nil
	}

	result := make([]attr.Value, 0)
	for _, tag := range StringSliceValue(data) {
		result = append(result, TrimmedStringValue(tag))
	}

	return types.SetValue(types.StringType, result)
}

func GetEnumAllowedDurationsSliceFromNumberSet(ctx context.Context, data types.Set) ([]client.EnumAllowedDurations, diag.Diagnostics) {
	allowedDurations := make([]client.EnumAllowedDurations, 0)
	if !data.IsNull() && !data.IsUnknown() {
		for _, item := range data.Elements() {
			val, ok := item.(types.Number)
			if !ok {
				continue
			}

			val, diags := val.ToNumberValue(ctx)
			if diags.HasError() {
				return nil, diags
			}

			valFloat32, _ := val.ValueBigFloat().Float32()
			allowedDurations = append(allowedDurations, client.EnumAllowedDurations(valFloat32))
		}
	}

	return allowedDurations, nil
}

func GetNumberSetFromAllowedDurations(data []client.EnumAllowedDurations) (types.Set, diag.Diagnostics) {
	s := make([]float32, 0, len(data))
	for _, v := range data {
		s = append(s, float32(v))
	}

	return GetNumberSet(s)
}

func GetNumberSet(data []float32) (types.Set, diag.Diagnostics) {
	if len(data) == 0 {
		return types.SetNull(types.NumberType), nil
	}

	result := make([]attr.Value, 0, len(data))
	for _, v := range data {
		result = append(result, types.NumberValue(big.NewFloat(float64(v))))
	}

	return types.SetValue(types.NumberType, result)
}
