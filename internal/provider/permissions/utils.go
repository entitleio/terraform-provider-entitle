package permissions

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
)

func GetTypesFromResponse(data []client.PermissionSchemaTypes) (types.Set, diag.Diagnostics) {
	s := make([]string, 0, len(data))
	for _, v := range data {
		s = append(s, string(v))
	}

	return utils.GetStringSet(&s)
}
