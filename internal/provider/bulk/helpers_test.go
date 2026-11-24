package bulk

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func TestAddSkippedWarning(t *testing.T) {
	var diags diag.Diagnostics
	// "c" was skipped by a cleanup of an entity no longer in config: not reported.
	AddSkippedWarning(&diags, "roles", []string{"a", "b"}, []string{"b", "c"})

	if diags.HasError() || diags.WarningsCount() != 1 {
		t.Fatalf("diags = %v", diags)
	}
	w := diags.Warnings()[0]
	if w.Summary() != "Skipped 1 roles that no longer exist" {
		t.Errorf("summary = %q", w.Summary())
	}
	if !strings.HasSuffix(w.Detail(), ": b") {
		t.Errorf("detail = %q", w.Detail())
	}
}

func TestAddSkippedWarningNothingSkipped(t *testing.T) {
	var diags diag.Diagnostics
	AddSkippedWarning(&diags, "roles", []string{"a"}, nil)
	AddSkippedWarning(&diags, "roles", []string{"a"}, []string{"c"})
	if len(diags) != 0 {
		t.Fatalf("expected no diagnostics, got %v", diags)
	}
}
