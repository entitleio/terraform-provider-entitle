package utils

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// IdEmailModel is a data model that represents a combination of an ID and an Email.
type IdEmailModel struct {
	// Id is the identifier attribute.
	Id types.String `tfsdk:"id"`

	// Email is the email address attribute.
	Email types.String `tfsdk:"email"`
}

func (m IdEmailModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":    types.StringType,
		"email": types.StringType,
	}
}

func (m IdEmailModel) AsObjectValue(ctx context.Context) (basetypes.ObjectValue, diag.Diagnostics) {
	return types.ObjectValueFrom(ctx, m.AttributeTypes(), m)
}

// IdNameEmailModel is a data model that represents an ID, a Name, and an Email.
type IdNameEmailModel struct {
	// Id is the identifier attribute.
	Id types.String `tfsdk:"id"`

	// Name is the name attribute.
	Name types.String `tfsdk:"name"`

	// Email is the email address attribute.
	Email types.String `tfsdk:"email"`
}

func (m IdNameEmailModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":    types.StringType,
		"name":  types.StringType,
		"email": types.StringType,
	}
}

func (m IdNameEmailModel) AsObjectValue(ctx context.Context) (basetypes.ObjectValue, diag.Diagnostics) {
	return types.ObjectValueFrom(ctx, m.AttributeTypes(), m)
}

// IdentityOnlyModel is a data model that represents only an ID.
type IdentityOnlyModel struct {
	// Id is the identifier attribute.
	Id types.String `tfsdk:"id"`
}

func (m IdentityOnlyModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id": types.StringType,
	}
}

func (m IdentityOnlyModel) AsObjectValue(ctx context.Context) (basetypes.ObjectValue, diag.Diagnostics) {
	return types.ObjectValueFrom(ctx, m.AttributeTypes(), m)
}

// IdNameModel is a data model that represents an ID and a Name.
type IdNameModel struct {
	// ID is the identifier attribute.
	ID types.String `tfsdk:"id"`

	// Name is the name attribute.
	Name types.String `tfsdk:"name"`
}

func (m IdNameModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":   types.StringType,
		"name": types.StringType,
	}
}

func (m IdNameModel) AsObjectValue(ctx context.Context) (basetypes.ObjectValue, diag.Diagnostics) {
	return types.ObjectValueFrom(ctx, m.AttributeTypes(), m)
}

// NameModel is a data model that represents only a Name.
type NameModel struct {
	// Name is the name attribute.
	Name types.String `tfsdk:"name"`
}

func (m NameModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name": types.StringType,
	}
}

func (m NameModel) AsObjectValue(ctx context.Context) (basetypes.ObjectValue, diag.Diagnostics) {
	return types.ObjectValueFrom(ctx, m.AttributeTypes(), m)
}
