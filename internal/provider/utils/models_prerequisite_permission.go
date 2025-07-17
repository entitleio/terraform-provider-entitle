package utils

import "github.com/hashicorp/terraform-plugin-framework/types"

// PrerequisitePermissionModel represents a model defining prerequisite permissions
// for a specific role and default setting.
type PrerequisitePermissionModel struct {
	Default types.Bool `tfsdk:"default"`
	Role    *Role      `tfsdk:"role"`
}
