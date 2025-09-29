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
		search = &s
	}

	p := int(m.Page.ValueInt64())
	if p > 0 {
		page = &p
	}

	pp := int(m.PerPage.ValueInt64())
	if pp > 0 {
		perPage = &pp
	}

	return
}
