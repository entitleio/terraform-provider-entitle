package utils

import (
	"context"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Role represents the internal representation of Role Model.
type Role struct {
	ID       types.String `tfsdk:"id" json:"id"`
	Name     types.String `tfsdk:"name" json:"name"`
	Resource types.Object `tfsdk:"resource" json:"resource"`
}

// attributeTypes returns the attribute types for Role.
func (m Role) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":   types.StringType,
		"name": types.StringType,
		"resource": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":   types.StringType,
				"name": types.StringType,
				"integration": types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"id":   types.StringType,
						"name": types.StringType,
						"application": types.ObjectType{
							AttrTypes: map[string]attr.Type{
								"name": types.StringType,
							},
						},
					},
				},
			},
		},
	}
}

// RoleResource represents the Terraform resource model for the Role resource.
type RoleResource struct {
	Id          types.String            `tfsdk:"id" json:"id"`
	Name        types.String            `tfsdk:"name" json:"name"`
	Integration RoleResourceIntegration `tfsdk:"integration" json:"integration"`
}

// AsObjectValue converts RoleResourceModel to ObjectValue.
func (m RoleResource) AsObjectValue(ctx context.Context) (basetypes.ObjectValue, diag.Diagnostics) {
	return types.ObjectValueFrom(ctx, RoleResourceModel{}.AttributeTypes(), m)
}

// RoleResourceModel represents the internal representation of RoleResourceModel.
type RoleResourceModel struct {
	ID          types.String `tfsdk:"id" json:"id"`
	Name        types.String `tfsdk:"name" json:"name"`
	Integration types.Object `tfsdk:"integration" json:"integration"`
}

// AttributeTypes returns the attribute types for RoleResourceModel.
func (m RoleResourceModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":   types.StringType,
		"name": types.StringType,
		"integration": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":   types.StringType,
				"name": types.StringType,
				"application": types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"name": types.StringType,
					},
				},
			},
		},
	}
}

// RoleResourceIntegration represents the Terraform resource model for the integration part of Role resource.
type RoleResourceIntegration struct {
	Id          types.String `tfsdk:"id" json:"id"`
	Name        types.String `tfsdk:"name" json:"name"`
	Application NameModel    `tfsdk:"application" json:"application"`
}

// AsObjectValue converts RoleResourceIntegration to ObjectValue.
func (m RoleResourceIntegration) AsObjectValue(ctx context.Context) (basetypes.ObjectValue, diag.Diagnostics) {
	return types.ObjectValueFrom(ctx, RoleResourceIntegrationModel{}.AttributeTypes(), m)
}

// RoleResourceIntegrationModel represents the internal representation of RoleResourceIntegration.
type RoleResourceIntegrationModel struct {
	ID          types.String `tfsdk:"id" json:"id"`
	Name        types.String `tfsdk:"name" json:"name"`
	Application types.Object `tfsdk:"application" json:"application"`
}

// AttributeTypes returns the attribute types for RoleResourceIntegrationModel.
func (m RoleResourceIntegrationModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":   types.StringType,
		"name": types.StringType,
		"application": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name": types.StringType,
			},
		},
	}
}

// RoleResourceIntegrationApplication represents the Terraform resource model for the application part of Role resource.
type RoleResourceIntegrationApplication struct {
	Name types.String `tfsdk:"name" json:"name"`
}

// attributeTypes returns the attribute types for RoleResourceIntegrationApplicationModel.
func (m RoleResourceIntegrationApplication) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name": types.StringType,
	}
}

// AsObjectValue converts RoleResourceIntegrationApplication to ObjectValue.
func (m RoleResourceIntegrationApplication) AsObjectValue(ctx context.Context) (basetypes.ObjectValue, diag.Diagnostics) {
	return types.ObjectValueFrom(ctx, m.AttributeTypes(), m)
}

func GetRole(
	ctx context.Context,
	id, name string,
	resource interface{},
) (*Role, diag.Diagnostics) {
	var diags diag.Diagnostics
	var r RoleResource

	// Create a new RoleModel with ID and Name from the response data.
	roleModel := &Role{
		ID:   TrimmedStringValue(id),
		Name: TrimmedStringValue(name),
	}

	switch res := resource.(type) {
	case client.ResourceResponseSchema:
		// Create a RoleResourceModel to store information about the role's resource.
		r = RoleResource{
			Id:   TrimmedStringValue(res.Id.String()),
			Name: TrimmedStringValue(res.Name),
			Integration: RoleResourceIntegration{
				Application: NameModel{
					Name: TrimmedStringValue(res.Integration.Application.Name),
				},
				Id:   TrimmedStringValue(res.Integration.Id.String()),
				Name: TrimmedStringValue(res.Integration.Name),
			},
		}
	case client.PolicyResourceResponseSchema:
		// Create a RoleResourceModel to store information about the role's resource.
		r = RoleResource{
			Id:   TrimmedStringValue(res.Id.String()),
			Name: TrimmedStringValue(res.Name),
			Integration: RoleResourceIntegration{
				Application: NameModel{
					Name: TrimmedStringValue(res.Integration.Application.Name),
				},
				Id:   TrimmedStringValue(res.Integration.Id.String()),
				Name: TrimmedStringValue(res.Integration.Name),
			},
		}
	default:
		diags.AddError(
			"Client Error",
			"failed invalid resource type in role",
		)
		return roleModel, diags
	}

	// Convert the role resource data to Terraform object value.
	val, diags := types.ObjectValueFrom(ctx, RoleResourceModel{}.AttributeTypes(), r)
	if diags.HasError() {
		return roleModel, diags
	}

	// Set the Terraform object value as the resource for the role model.
	roleModel.Resource = val
	return roleModel, diags
}
