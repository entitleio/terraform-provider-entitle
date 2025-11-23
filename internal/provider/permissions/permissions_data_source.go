package permissions

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/docs"
	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"github.com/entitleio/terraform-provider-entitle/internal/validators"
)

// Ensure the interface is satisfied.
var _ datasource.DataSource = &PermissionsDataSource{}

// PermissionDataSourceModel describes the data source data model.
type PermissionDataSourceModel struct {
	ID    types.String        `tfsdk:"id"`
	Actor *utils.IdEmailModel `tfsdk:"actor"`
	Role  *utils.Role         `tfsdk:"role"`
	Path  types.String        `tfsdk:"path"`
	Types types.Set           `tfsdk:"types"`
}

// PermissionsDataSource defines the list data source.
type PermissionsDataSource struct {
	client *client.ClientWithResponses
}

// PermissionFilterModel represents optional filters.
type PermissionFilterModel struct {
	IntegrationID types.String `tfsdk:"integration_id"`
	ResourceID    types.String `tfsdk:"resource_id"`
	RoleID        types.String `tfsdk:"role_id"`
	AccountID     types.String `tfsdk:"account_id"`
	Search        types.String `tfsdk:"search"`
}

// PermissionListDataSourceModel is the Terraform state model.
type PermissionListDataSourceModel struct {
	Filter      *PermissionFilterModel      `tfsdk:"filter"`
	Permissions []PermissionDataSourceModel `tfsdk:"permissions"`
}

// NewPermissionsDataSource constructor.
func NewPermissionsDataSource() datasource.DataSource {
	return &PermissionsDataSource{}
}

func (d *PermissionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_permissions"
}

func (d *PermissionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: docs.PermissionsDataSourceMarkdownDescription,
		Blocks: map[string]schema.Block{
			"filter": schema.SingleNestedBlock{
				Attributes: map[string]schema.Attribute{
					"integration_id": schema.StringAttribute{
						Optional:    true,
						Validators:  []validator.String{validators.UUID{}},
						Description: "Filter permissions by integration ID (UUID).",
					},
					"resource_id": schema.StringAttribute{
						Optional:    true,
						Validators:  []validator.String{validators.UUID{}},
						Description: "Filter permissions by resource ID (UUID).",
					},
					"role_id": schema.StringAttribute{
						Optional:    true,
						Validators:  []validator.String{validators.UUID{}},
						Description: "Filter permissions by role ID (UUID).",
					},
					"account_id": schema.StringAttribute{
						Optional:    true,
						Validators:  []validator.String{validators.UUID{}},
						Description: "Filter permissions by account ID (UUID).",
					},
					"search": schema.StringAttribute{
						Optional:    true,
						Description: "Text search on permissions.",
					},
				},
			},
		},
		Attributes: map[string]schema.Attribute{
			"permissions": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{Computed: true},
						"actor": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"id":    schema.StringAttribute{Computed: true},
								"email": schema.StringAttribute{Computed: true},
							},
						},
						"role": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"id":   schema.StringAttribute{Computed: true},
								"name": schema.StringAttribute{Computed: true},
								"resource": schema.SingleNestedAttribute{
									Computed: true,
									Attributes: map[string]schema.Attribute{
										"id":   schema.StringAttribute{Computed: true},
										"name": schema.StringAttribute{Computed: true},
										"integration": schema.SingleNestedAttribute{
											Computed: true,
											Attributes: map[string]schema.Attribute{
												"id":   schema.StringAttribute{Computed: true},
												"name": schema.StringAttribute{Computed: true},
												"application": schema.SingleNestedAttribute{
													Computed: true,
													Attributes: map[string]schema.Attribute{
														"name": schema.StringAttribute{Computed: true},
													},
												},
											},
										},
									},
								},
							},
						},
						"path": schema.StringAttribute{
							Computed: true,
						},
						"types": schema.SetAttribute{
							Computed:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

func (d *PermissionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.ClientWithResponses, got: %T", req.ProviderData),
		)
		return
	}

	d.client = c
}

func (d *PermissionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PermissionListDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build API query parameters
	params := client.PermissionsIndexParams{}
	if data.Filter != nil {
		if data.Filter.IntegrationID.ValueString() != "" {
			params.IntegrationId = data.Filter.IntegrationID.ValueStringPointer()
		}
		if data.Filter.ResourceID.ValueString() != "" {
			params.ResourceId = data.Filter.ResourceID.ValueStringPointer()
		}
		if data.Filter.RoleID.ValueString() != "" {
			params.RoleId = data.Filter.RoleID.ValueStringPointer()
		}
		if data.Filter.Search.ValueString() != "" {
			params.Search = data.Filter.Search.ValueStringPointer()
		}
	}

	tflog.Trace(ctx, "Fetching filtered permissions from Entitle API")

	var reqStatusCode int
	var reqBody []byte
	var reqResponse []client.PermissionSchema
	if data.Filter.AccountID.ValueString() != "" {
		accountId := uuid.MustParse(data.Filter.AccountID.ValueString())

		permissionsResp, err := d.client.PermissionsIndexAccountWithResponse(ctx, accountId, &client.PermissionsIndexAccountParams{
			Page:          params.Page,
			PerPage:       params.PerPage,
			IntegrationId: params.IntegrationId,
			ResourceId:    params.ResourceId,
			RoleId:        params.RoleId,
			Search:        params.Search,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				utils.ErrApiConnection.Error(),
				fmt.Sprintf("Unable to list permissions: %s", err))
			return
		}

		reqStatusCode = permissionsResp.StatusCode()
		reqBody = permissionsResp.Body
		if permissionsResp.JSON200 != nil {
			reqResponse = permissionsResp.JSON200.Result
		}
	} else {
		permissionsResp, err := d.client.PermissionsIndexWithResponse(ctx, &params)
		if err != nil {
			resp.Diagnostics.AddError(
				utils.ErrApiConnection.Error(),
				fmt.Sprintf("Unable to list permissions: %s", err))
			return
		}

		reqStatusCode = permissionsResp.StatusCode()
		reqBody = permissionsResp.Body
		if permissionsResp.JSON200 != nil {
			reqResponse = permissionsResp.JSON200.Result
		}
	}

	if err := utils.HTTPResponseToError(reqStatusCode, reqBody); err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(), fmt.Sprintf("Failed to list permissions: %s", err))
		return
	}

	// Map API results
	var permissionsList []PermissionDataSourceModel
	for _, perm := range reqResponse {
		allowedTypes, advDiags := GetTypesFromResponse(perm.Types)
		if advDiags.HasError() {
			resp.Diagnostics.Append(advDiags...)
			return
		}

		resourceObj, diags := utils.RoleResource{
			Id:   utils.TrimmedStringValue(perm.Role.Resource.Id.String()),
			Name: utils.TrimmedStringValue(perm.Role.Resource.Name),
			Integration: utils.RoleResourceIntegration{
				Id:   utils.TrimmedStringValue(perm.Role.Resource.Integration.Id.String()),
				Name: utils.TrimmedStringValue(perm.Role.Resource.Integration.Name),
				Application: utils.NameModel{
					Name: utils.TrimmedStringValue(perm.Role.Resource.Integration.Application.Name),
				},
			},
		}.AsObjectValue(ctx)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		permissionsList = append(permissionsList, PermissionDataSourceModel{
			ID: utils.TrimmedStringValue(perm.PermissionId.String()),
			Actor: &utils.IdEmailModel{
				Id:    utils.TrimmedStringValue(perm.Account.Id.String()),
				Email: utils.TrimmedStringValue(perm.Account.Email),
			},
			Role: &utils.Role{
				ID:       utils.TrimmedStringValue(perm.Role.Id.String()),
				Name:     utils.TrimmedStringValue(perm.Role.Name),
				Resource: resourceObj,
			},
			Path:  utils.TrimmedStringValue(string(perm.Path)),
			Types: allowedTypes,
		})
	}

	data.Permissions = permissionsList
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
