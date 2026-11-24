package roles

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func stringSet(values ...string) types.Set {
	elems := make([]attr.Value, len(values))
	for i, v := range values {
		elems[i] = types.StringValue(v)
	}
	return types.SetValueMust(types.StringType, elems)
}

func nullFilter() *RolesFilterModel {
	return &RolesFilterModel{
		ResourceID:            types.StringNull(),
		IntegrationID:         types.StringNull(),
		Search:                types.StringNull(),
		ExternalID:            types.StringNull(),
		IDs:                   types.SetNull(types.StringType),
		IDsOperator:           types.StringNull(),
		Requestable:           types.BoolNull(),
		WorkflowIDs:           types.SetNull(types.StringType),
		WorkflowIDsOperator:   types.StringNull(),
		ResourceTypes:         types.SetNull(types.StringType),
		ResourceTypesOperator: types.StringNull(),
		ResourceTags:          types.SetNull(types.StringType),
		ResourceTagsOperator:  types.StringNull(),
		ResourceTagsRelation:  types.StringNull(),
		Page:                  types.Int64Null(),
		PerPage:               types.Int64Null(),
	}
}

// Configured operators are passed to the query as-is.
func TestRolesFilterQueryOperators(t *testing.T) {
	f := nullFilter()
	f.WorkflowIDs = stringSet("00000000-0000-0000-0000-000000000002")
	f.WorkflowIDsOperator = types.StringValue("isNot")
	f.ResourceTags = stringSet("env: prod")
	f.ResourceTagsOperator = types.StringValue("isNot")
	f.ResourceTagsRelation = types.StringValue("and")

	q, diags := f.query(context.Background())
	if diags.HasError() {
		t.Fatal(diags)
	}

	got := q.Values()
	if got.Get("workflowIdOperator") != "isNot" || got.Get("resourceTagsOperator") != "isNot" || got.Get("resourceTagsRelation") != "and" {
		t.Fatalf("unexpected query %s", got.Encode())
	}
}

// Unset operators stay empty here; the client fills in the defaults.
func TestRolesFilterQueryUnsetOperators(t *testing.T) {
	q, diags := nullFilter().query(context.Background())
	if diags.HasError() {
		t.Fatal(diags)
	}
	if q.IDsOperator != "" || q.WorkflowIDOperator != "" || q.ResourceTypeOperator != "" ||
		q.ResourceTagsOperator != "" || q.ResourceTagsRelation != "" {
		t.Fatalf("unexpected operators %+v", q)
	}
}
