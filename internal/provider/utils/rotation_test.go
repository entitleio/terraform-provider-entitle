package utils

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestRotationChanged(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		state types.String
		plan  types.String
		want  bool
	}{
		{
			name:  "value changed rotates",
			state: types.StringValue("1"),
			plan:  types.StringValue("2"),
			want:  true,
		},
		{
			name:  "same value does not rotate",
			state: types.StringValue("1"),
			plan:  types.StringValue("1"),
			want:  false,
		},
		{
			name:  "empty string is a value like any other",
			state: types.StringValue(""),
			plan:  types.StringValue("1"),
			want:  true,
		},
		{
			name:  "adopting the attribute rotates",
			state: types.StringNull(),
			plan:  types.StringValue("1"),
			want:  true,
		},
		{
			name:  "removing the attribute does not rotate",
			state: types.StringValue("1"),
			plan:  types.StringNull(),
			want:  false,
		},
		{
			name:  "never set stays unrotated",
			state: types.StringNull(),
			plan:  types.StringNull(),
			want:  false,
		},
		{
			name:  "unknown plan defers the decision to apply",
			state: types.StringValue("1"),
			plan:  types.StringUnknown(),
			want:  false,
		},
		{
			name:  "unknown state defers the decision to apply",
			state: types.StringUnknown(),
			plan:  types.StringValue("1"),
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := RotationChanged(tt.state, tt.plan); got != tt.want {
				t.Errorf("RotationChanged(%s, %s) = %v, want %v", tt.state, tt.plan, got, tt.want)
			}
		})
	}
}

func TestRotatableSecretPlanModifiersOrder(t *testing.T) {
	t.Parallel()

	// RotationTriggersNewSecret overrides UseStateForUnknown, so it has to run
	// after it. Guard the order, since swapping the two silently reintroduces
	// the bug where a pending rotation is invisible in the plan.
	modifiers := RotatableSecretPlanModifiers(path.Root(DefaultRotationAttributeName))
	if len(modifiers) != 2 {
		t.Fatalf("expected 2 plan modifiers, got %d", len(modifiers))
	}

	if _, ok := modifiers[1].(rotationPlanModifier); !ok {
		t.Errorf("expected the rotation modifier last, got %T", modifiers[1])
	}
}
