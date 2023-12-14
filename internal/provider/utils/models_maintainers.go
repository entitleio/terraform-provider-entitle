package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// MaintainerCommonResponseSchema represents the common structure of a maintainer response in the integration and resource.
type MaintainerCommonResponseSchema struct {
	Type  string                      `json:"type"`
	Group client.EntityResponseSchema `json:"group"`
	User  client.EntityResponseSchema `json:"user"`
}

// MaintainerModel describes the data source data model.
type MaintainerModel struct {
	Type   types.String `tfsdk:"type"`
	Entity types.Object `tfsdk:"entity"`
}

// attributeTypes returns the attribute types for MaintainerModel.
func (m MaintainerModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id": types.StringType,
		"entity": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":    types.StringType,
				"email": types.StringType,
			},
		},
	}
}

type MaintainerInterface interface {
	AsMaintainerUserResponseSchema() (client.MaintainerUserResponseSchema, error)
	AsMaintainerGroupResponseSchema() (client.MaintainerGroupResponseSchema, error)
	MarshalJSON() ([]byte, error)
}

func GetMaintainers[T MaintainerInterface](
	ctx context.Context,
	maintainers []T,
) ([]*MaintainerModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	result := make([]*MaintainerModel, 0)

	for _, item := range maintainers {
		data, err := item.MarshalJSON()
		if err != nil {
			diags.AddError(
				"Failed to marshal maintainer data",
				err.Error(),
			)
			return nil, diags
		}

		var body MaintainerCommonResponseSchema
		err = json.Unmarshal(data, &body)
		if err != nil {
			diags.AddError(
				"Failed to unmarshal maintainer data",
				err.Error(),
			)
			return nil, diags
		}

		switch strings.ToLower(body.Type) {
		case "user":
			responseSchema, err := item.AsMaintainerUserResponseSchema()
			if err != nil {
				diags.AddError(
					"No data",
					fmt.Sprintf("failed to convert response schema to user response schema, error: %v", err),
				)

				return nil, diags
			}

			bytes, err := responseSchema.User.Email.MarshalJSON()
			if err != nil {
				diags.AddError(
					fmt.Sprintf("Failed to get maintainer email for user id (%s)", body.User.Id),
					err.Error(),
				)

				return nil, diags
			}

			u := &IdEmailModel{
				Id:    types.StringValue(responseSchema.User.Id.String()),
				Email: types.StringValue(string(bytes)),
			}

			uObject, diags := u.AsObjectValue(ctx)
			if diags.HasError() {
				diags.Append(diags...)
				return nil, diags
			}

			result = append(result, &MaintainerModel{
				Type:   types.StringValue(body.Type),
				Entity: uObject,
			})
		case "group":
			responseSchema, err := item.AsMaintainerGroupResponseSchema()
			if err != nil {
				diags.AddError(
					"No data",
					fmt.Sprintf("failed to convert response schema to group response schema, error: %v", err),
				)

				return nil, diags
			}

			bytes, err := responseSchema.Group.Email.MarshalJSON()
			if err != nil {
				diags.AddError(
					"No data",
					fmt.Sprintf("failed to get maintainer group email bytes, error: %v", err),
				)

				return nil, diags
			}

			g := &IdEmailModel{
				Id:    types.StringValue(responseSchema.Group.Id.String()),
				Email: types.StringValue(string(bytes)),
			}

			gObject, diagsAsObject := g.AsObjectValue(ctx)
			diags.Append(diagsAsObject...)
			if diags.HasError() {
				return nil, diags
			}

			result = append(result, &MaintainerModel{
				Type:   types.StringValue(body.Type),
				Entity: gObject,
			})
		default:
			diags.AddError("failed invalid type for maintainer", body.Type)
			return nil, diags
		}
	}

	return result, diags
}
