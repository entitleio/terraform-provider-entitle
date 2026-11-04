package integrations

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/docs"
	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
)

var _ datasource.DataSource = &IntegrationsDataSource{}

// NewIntegrationsDataSource creates a new entitle_integrations data source.
func NewIntegrationsDataSource() datasource.DataSource {
	return &IntegrationsDataSource{}
}

// IntegrationsDataSource lists integrations.
type IntegrationsDataSource struct {
	client *client.ClientWithResponses
}

// IntegrationsDataSourceModel describes the data source model.
type IntegrationsDataSourceModel struct {
	Filter       *IntegrationsFilterModel   `tfsdk:"filter"`
	Integrations []IntegrationListItemModel `tfsdk:"integrations"`
}

// IntegrationsFilterModel is the filter block of entitle_integrations.
// Pagination is handled internally: every matching integration is returned.
type IntegrationsFilterModel struct {
	IDs                    types.Set    `tfsdk:"ids"`
	IDsOperator            types.String `tfsdk:"ids_operator"`
	Names                  types.Set    `tfsdk:"names"`
	NamesOperator          types.String `tfsdk:"names_operator"`
	NamesRelation          types.String `tfsdk:"names_relation"`
	Requestable            types.Bool   `tfsdk:"requestable"`
	WorkflowIDs            types.Set    `tfsdk:"workflow_ids"`
	WorkflowIDsOperator    types.String `tfsdk:"workflow_ids_operator"`
	OwnerIDs               types.Set    `tfsdk:"owner_ids"`
	OwnerIDsOperator       types.String `tfsdk:"owner_ids_operator"`
	MaintainerIDs          types.Set    `tfsdk:"maintainer_ids"`
	MaintainerIDsOperator  types.String `tfsdk:"maintainer_ids_operator"`
	ApplicationIDs         types.Set    `tfsdk:"application_ids"`
	ApplicationIDsOperator types.String `tfsdk:"application_ids_operator"`
}

// IntegrationListItemModel is a single integration in the list.
type IntegrationListItemModel struct {
	ID          types.String     `tfsdk:"id"`
	Name        types.String     `tfsdk:"name"`
	Application *utils.NameModel `tfsdk:"application"`
}

// Metadata sets the data source type name.
func (d *IntegrationsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integrations"
}

// noComma rejects values with a comma: the API always splits name values on commas.
var noComma = stringvalidator.RegexMatches(regexp.MustCompile(`^[^,]*$`), "must not contain a comma")

func integrationsFilterAttributes() map[string]schema.Attribute {
	ids := utils.IDsAttributes("integrations")

	return map[string]schema.Attribute{
		"ids":          ids["ids"],
		"ids_operator": ids["ids_operator"],
		"names": schema.SetAttribute{
			ElementType:         types.StringType,
			Optional:            true,
			MarkdownDescription: "Return only integrations whose name contains these values (case-insensitive, see `names_operator` and `names_relation`). Values cannot contain a comma.",
			Validators: []validator.Set{
				setvalidator.SizeAtLeast(1),
				setvalidator.ValueStringsAre(noComma),
			},
		},
		"names_operator": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "How `names` is matched: `contains` returns integrations whose name contains the values, `notContains` returns integrations whose name does not contain them. Defaults to `contains`.",
			Validators: []validator.String{
				stringvalidator.OneOf(utils.ContainsOperators...),
			},
		},
		"names_relation": schema.StringAttribute{
			Optional: true,
			MarkdownDescription: "How multiple `names` are combined. With `contains`: `or` matches names that contain any of the values, " +
				"`and` matches names that contain all of them. With `notContains`: `or` matches names that contain none of the values, " +
				"`and` matches names that are missing at least one of them. Defaults to `or`.",
			Validators: []validator.String{
				stringvalidator.OneOf(utils.Relations...),
			},
		},
		"requestable": schema.BoolAttribute{
			Optional:            true,
			MarkdownDescription: "Return only integrations that are (`true`) or are not (`false`) requestable.",
		},
		"workflow_ids": utils.UUIDSetAttribute("Return only integrations with one of these workflow IDs (UUID) (see `workflow_ids_operator`)."),
		"workflow_ids_operator": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "How `workflow_ids` is matched: `is` returns integrations with one of the given workflows, `isNot` returns integrations with none of them. Defaults to `is`.",
			Validators: []validator.String{
				stringvalidator.OneOf(utils.Operators...),
			},
		},
		"owner_ids": utils.UUIDSetAttribute("Return only integrations owned by one of these users (UUID) (see `owner_ids_operator`)."),
		"owner_ids_operator": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "How `owner_ids` is matched: `is` returns integrations owned by one of the given users, `isNot` returns integrations owned by none of them. Defaults to `is`.",
			Validators: []validator.String{
				stringvalidator.OneOf(utils.Operators...),
			},
		},
		"maintainer_ids": utils.UUIDSetAttribute("Return only integrations maintained by one of these users or groups (UUID) (see `maintainer_ids_operator`)."),
		"maintainer_ids_operator": schema.StringAttribute{
			Optional: true,
			MarkdownDescription: "How `maintainer_ids` is matched: `contains` returns integrations that have at least one of the given maintainers, " +
				"`notContains` returns integrations that have none of them. Defaults to `contains`.",
			Validators: []validator.String{
				stringvalidator.OneOf(utils.ContainsOperators...),
			},
		},
		"application_ids": utils.UUIDSetAttribute("Return only integrations of one of these applications (UUID) (see `application_ids_operator`)."),
		"application_ids_operator": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "How `application_ids` is matched: `is` returns integrations of one of the given applications, `isNot` returns integrations of none of them. Defaults to `is`.",
			Validators: []validator.String{
				stringvalidator.OneOf(utils.Operators...),
			},
		},
	}
}

// query converts the filter into list query parameters. A nil filter yields an
// empty query; the client fills in default operators.
func (f *IntegrationsFilterModel) query(ctx context.Context) (client.ListQuery, diag.Diagnostics) {
	var (
		q     client.ListQuery
		diags diag.Diagnostics
	)
	if f == nil {
		return q, diags
	}

	q.IDs = utils.SetStrings(ctx, f.IDs, &diags)
	q.IDsOperator = client.IDsOperator(f.IDsOperator.ValueString())
	q.Names = utils.SetStrings(ctx, f.Names, &diags)
	q.NameOperator = f.NamesOperator.ValueString()
	q.NameRelation = f.NamesRelation.ValueString()
	q.Requestable = f.Requestable.ValueBoolPointer()
	q.WorkflowIDs = utils.SetStrings(ctx, f.WorkflowIDs, &diags)
	q.WorkflowIDOperator = f.WorkflowIDsOperator.ValueString()
	q.OwnerIDs = utils.SetStrings(ctx, f.OwnerIDs, &diags)
	q.OwnerIDOperator = f.OwnerIDsOperator.ValueString()
	q.MaintainerIDs = utils.SetStrings(ctx, f.MaintainerIDs, &diags)
	q.MaintainerIDOperator = f.MaintainerIDsOperator.ValueString()
	q.ApplicationIDs = utils.SetStrings(ctx, f.ApplicationIDs, &diags)
	q.ApplicationIDOperator = f.ApplicationIDsOperator.ValueString()
	return q, diags
}

// Schema defines the data source schema.
func (d *IntegrationsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: docs.IntegrationsDataSourceMarkdownDescription,
		Blocks: map[string]schema.Block{
			"filter": schema.SingleNestedBlock{
				MarkdownDescription: "Optional filters for the integrations. All filters are combined with AND. " +
					"Every matching integration is returned; pagination is handled by the provider.",
				Attributes: integrationsFilterAttributes(),
			},
		},
		Attributes: map[string]schema.Attribute{
			"integrations": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "All integrations matching the filter, sorted by ID.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The integration's unique identifier.",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The integration's display name.",
						},
						"application": schema.SingleNestedAttribute{
							Computed:            true,
							MarkdownDescription: "The application the integration connects to.",
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "The application's name.",
								},
							},
						},
					},
				},
			},
		},
	}
}

// Configure sets the API client.
func (d *IntegrationsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := utils.ClientFromProviderData(req.ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *utils.ProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = c
}

// Read retrieves every matching integration from the Entitle API, following
// the afterId cursor until the last page.
func (d *IntegrationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data IntegrationsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query, diags := data.Filter.query(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	items, apiResp, err := client.ListAll(ctx, query, d.client.IntegrationsListWithResponse,
		func(it client.IntegrationBaseResponseSchema) string { return it.Id.String() })
	if err != nil {
		resp.Diagnostics.AddError(utils.ErrApiConnection.Error(), fmt.Sprintf("Unable to list integrations: %s", err))
		return
	}
	if err := utils.HTTPResponseToError(apiResp.StatusCode(), apiResp.Body); err != nil {
		resp.Diagnostics.AddError(utils.ErrApiResponse.Error(), fmt.Sprintf("Failed to list integrations: %s", err))
		return
	}

	integrations := make([]IntegrationListItemModel, len(items))
	for i, it := range items {
		integrations[i] = IntegrationListItemModel{
			ID:          types.StringValue(it.Id.String()),
			Name:        utils.TrimmedStringValue(it.Name),
			Application: &utils.NameModel{Name: utils.TrimmedStringValue(it.Application.Name)},
		}
	}
	data.Integrations = integrations

	tflog.Trace(ctx, "Read entitle integrations list data source", map[string]any{
		"count": len(integrations),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
