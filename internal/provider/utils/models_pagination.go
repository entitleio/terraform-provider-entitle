package utils

import "github.com/hashicorp/terraform-plugin-framework/types"

type PaginationWithSearchModel struct {
	Search  types.String `tfsdk:"search"`
	Page    types.Int64  `tfsdk:"page"`
	PerPage types.Int64  `tfsdk:"per_page"`
}

func (m PaginationWithSearchModel) GetValues() (search *string, page, perPage *int) {
	s := m.Search.ValueString()
	if s != "" {
		search = new(s)
	}

	p := int(m.Page.ValueInt64())
	if p > 0 {
		page = new(p)
	}

	pp := int(m.PerPage.ValueInt64())
	if pp > 0 {
		perPage = new(pp)
	}

	return
}

type PaginationWithSearchAndExternalIdModel struct {
	Search     types.String `tfsdk:"search"`
	Page       types.Int64  `tfsdk:"page"`
	PerPage    types.Int64  `tfsdk:"per_page"`
	ExternalID types.String `tfsdk:"external_id"`
}

func (m PaginationWithSearchAndExternalIdModel) GetValues() (search *string, page, perPage *int, externalId *string) {
	s := m.Search.ValueString()
	if s != "" {
		search = new(s)
	}

	p := int(m.Page.ValueInt64())
	if p > 0 {
		page = new(p)
	}

	pp := int(m.PerPage.ValueInt64())
	if pp > 0 {
		perPage = new(pp)
	}

	eId := m.ExternalID.ValueString()
	if eId != "" {
		externalId = &eId
	}

	return
}
