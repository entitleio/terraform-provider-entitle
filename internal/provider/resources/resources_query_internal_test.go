package resources

import (
	"context"
	"net/url"
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

func nullResourcesFilter() *ResourcesFilterModel {
	return &ResourcesFilterModel{
		IntegrationID:         types.StringNull(),
		Search:                types.StringNull(),
		ExternalID:            types.StringNull(),
		IDs:                   types.SetNull(types.StringType),
		IDsOperator:           types.StringNull(),
		Requestable:           types.BoolNull(),
		WorkflowIDs:           types.SetNull(types.StringType),
		WorkflowIDsOperator:   types.StringNull(),
		OwnerIDs:              types.SetNull(types.StringType),
		OwnerIDsOperator:      types.StringNull(),
		MaintainerIDs:         types.SetNull(types.StringType),
		MaintainerIDsOperator: types.StringNull(),
		Types:                 types.SetNull(types.StringType),
		TypesOperator:         types.StringNull(),
		Tags:                  types.SetNull(types.StringType),
		TagsOperator:          types.StringNull(),
		TagsRelation:          types.StringNull(),
		Page:                  types.Int64Null(),
		PerPage:               types.Int64Null(),
	}
}

// Configured filters are passed to the query as-is; search is not sent.
func TestResourcesFilterQuery(t *testing.T) {
	f := nullResourcesFilter()
	f.IntegrationID = types.StringValue("00000000-0000-0000-0000-000000000001")
	f.Search = types.StringValue("prod")
	f.Requestable = types.BoolValue(true)
	f.OwnerIDs = stringSet("00000000-0000-0000-0000-000000000002")
	f.OwnerIDsOperator = types.StringValue("isNot")
	f.MaintainerIDs = stringSet("00000000-0000-0000-0000-000000000003")
	f.MaintainerIDsOperator = types.StringValue("notContains")
	f.Types = stringSet("Policy")
	f.TypesOperator = types.StringValue("contains")
	f.Tags = stringSet("env: prod")
	f.TagsRelation = types.StringValue("and")

	q, diags := f.query(context.Background())
	if diags.HasError() {
		t.Fatal(diags)
	}

	want := url.Values{
		"integrationId":        {"00000000-0000-0000-0000-000000000001"},
		"requestable":          {"true"},
		"ownerId":              {"00000000-0000-0000-0000-000000000002"},
		"ownerIdOperator":      {"isNot"},
		"maintainerId":         {"00000000-0000-0000-0000-000000000003"},
		"maintainerIdOperator": {"notContains"},
		"type":                 {"Policy"},
		"typeOperator":         {"contains"},
		"tags":                 {"env: prod"},
		"tagsRelation":         {"and"},
	}
	if got := q.Values(); got.Encode() != want.Encode() {
		t.Fatalf("got %s, want %s", got.Encode(), want.Encode())
	}
}

// Unset operators stay empty here; the client fills in the defaults.
func TestResourcesFilterQueryUnsetOperators(t *testing.T) {
	q, diags := nullResourcesFilter().query(context.Background())
	if diags.HasError() {
		t.Fatal(diags)
	}
	if enc := q.Values().Encode(); enc != "" {
		t.Fatalf("expected an empty query, got %s", enc)
	}
}
