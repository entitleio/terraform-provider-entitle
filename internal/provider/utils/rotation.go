package utils

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DefaultRotationAttributeName is the conventional name for a rotation trigger
// attribute. Resources should use this name unless they already use it for
// something else.
const DefaultRotationAttributeName = "rotation"

// RotationChanged reports whether a rotation trigger changed, which is the
// signal for a resource's Update to call its provider's rotate endpoint. It is
// the apply-time decision; see RotationTriggersNewSecret for the deliberately
// more pessimistic plan-time one.
//
// The value itself is meaningless, only the change matters. Setting the
// attribute for the first time on a resource that already exists counts: it is
// how an imported resource, whose secret Terraform never saw, obtains a usable
// one. Creating a resource with the attribute already set does not rotate,
// because a create issues a fresh secret anyway and Update is never called.
//
// Deliberate non-triggers:
//
//   - plan is null (the attribute was removed from the configuration). Removing
//     the trigger should stop future rotations, not perform one.
//   - either side is unknown. At apply time this cannot happen, and at plan
//     time an unknown value is handled separately.
func RotationChanged(state, plan types.String) bool {
	if plan.IsNull() {
		return false
	}

	if state.IsUnknown() || plan.IsUnknown() {
		return false
	}

	return !state.Equal(plan)
}

// RotationSchemaAttribute builds the rotation trigger attribute for a resource
// whose rotatable secret lives in secretAttributeName. Using it keeps the
// semantics and the wording identical across every resource that supports
// rotation.
//
// The attribute is deliberately not WriteOnly. Write-only arguments are always
// null in prior state, planned state and final state, and therefore "cannot
// produce a Terraform plan difference" by design. Marking the trigger write-only
// would leave RotationChanged comparing null to null on every plan, so nothing
// would ever rotate and no diff would ever appear, with no error to explain it.
// This attribute is the counterpart the framework docs recommend pairing *with*
// a write-only argument (the password_wo / password_wo_version pattern, or the
// random provider's keepers): the trigger has to be stored, because being
// diffable is the whole job.
func RotationSchemaAttribute(secretAttributeName string) schema.StringAttribute {
	description := fmt.Sprintf(
		"An arbitrary value that triggers in-place rotation of `%[1]s`. The value itself is "+
			"meaningless; any change to it makes the next `terraform apply` rotate the secret and "+
			"write the new value to `%[1]s`, keeping the same resource id and anything that depends "+
			"on it. This includes setting the attribute for the first time on a resource that "+
			"already exists, so `terraform plan` it first — the previous secret is invalidated "+
			"immediately. Removing the attribute does not rotate. Common values are a version "+
			"counter (`\"1\"`, `\"2\"`) or `time_rotating.example.rfc3339` to rotate on a cadence.",
		secretAttributeName,
	)

	return schema.StringAttribute{
		Optional:            true,
		MarkdownDescription: description,
		Description: fmt.Sprintf(
			"An arbitrary value that triggers in-place rotation of %s when changed.",
			secretAttributeName,
		),
	}
}

// RotatableSecretPlanModifiers returns the plan modifiers for a computed,
// sensitive attribute that holds a secret which the API only returns on create
// and on rotate, and whose rotation is triggered by the attribute at
// rotationPath.
//
// The pair matters. UseStateForUnknown carries the secret forward, because a
// read cannot recover it. RotationTriggersNewSecret then overrides that when a
// rotation is pending, and must therefore come second.
func RotatableSecretPlanModifiers(rotationPath path.Path) []planmodifier.String {
	return []planmodifier.String{
		stringplanmodifier.UseStateForUnknown(),
		RotationTriggersNewSecret(rotationPath),
	}
}

// RotationTriggersNewSecret returns a plan modifier for a computed secret
// attribute that is rotated when the attribute at rotationPath changes.
//
// The secret normally carries its prior value forward via UseStateForUnknown,
// because the API never returns it again after creation. When the rotation
// trigger changes the secret is about to change, so the planned value must
// become unknown instead. That has two effects:
//
//   - the attribute appears in terraform plan as a pending change, which is how
//     the rotation becomes visible before it happens. Note that a Sensitive
//     attribute renders as "(sensitive value)" and not "(known after apply)":
//     Terraform's differ checks sensitivity before unknownness, so sensitivity
//     wins. Any documentation that tells users what to look for in the plan
//     must say "(sensitive value)" for a secret.
//   - anything that consumes the secret (a Kubernetes secret, a Secrets Manager
//     version) sees a planned change and is updated in the same apply.
//
// The plan-time test is deliberately weaker than RotationChanged: an unknown
// trigger (typically time_rotating.x.rfc3339, which is unknown whenever the
// time resource is being replaced) also forces the secret unknown. Terraform
// re-plans during the apply walk with the dependency resolved, and a planned
// value may go unknown -> known but never known -> unknown, so the pessimistic
// choice is the only one that cannot fail with "Provider produced inconsistent
// final plan". Marking the secret unknown and then not rotating is harmless.
//
// It must be registered after UseStateForUnknown so it can override it.
func RotationTriggersNewSecret(rotationPath path.Path) planmodifier.String {
	return rotationPlanModifier{rotationPath: rotationPath}
}

type rotationPlanModifier struct {
	rotationPath path.Path
}

func (m rotationPlanModifier) Description(_ context.Context) string {
	return fmt.Sprintf("Plans the value as unknown when %s changes, so the pending rotation is visible in the plan.", m.rotationPath)
}

func (m rotationPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m rotationPlanModifier) PlanModifyString(
	ctx context.Context,
	req planmodifier.StringRequest,
	resp *planmodifier.StringResponse,
) {
	// Nothing to compare on create (no prior state) or destroy (no plan).
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var stateRotation, planRotation types.String

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, m.rotationPath, &stateRotation)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, m.rotationPath, &planRotation)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if planRotation.IsUnknown() || RotationChanged(stateRotation, planRotation) {
		resp.PlanValue = types.StringUnknown()
	}
}
