package permissions

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"github.com/entitleio/terraform-provider-entitle/internal/validators"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &PermissionDataSource{}

// PermissionDataSource defines the data source implementation.
type PermissionDataSource struct {
	client *client.ClientWithResponses
}

// NewPermissionDataSource creates a new instance of PermissionDataSource.
func NewPermissionDataSource() datasource.DataSource {
	return &PermissionDataSource{}
}

// PermissionDataSourceModel describes the data source data model.
type PermissionDataSourceModel struct {
	ID    types.String        `tfsdk:"id"`
	Actor *utils.IdEmailModel `tfsdk:"actor"`
	Role  *utils.Role         `tfsdk:"role"`
	Path  types.String        `tfsdk:"path"`
	Types types.Set           `tfsdk:"types"`
}

// Metadata sets the metadata for the data source.
func (d *PermissionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_permission"
}

// Schema defines the data source schema.
func (d *PermissionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "",
		Description:         "",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Entitle Permission identifier in uuid format",
				Description:         "Entitle Permission identifier in uuid format",
				Validators: []validator.String{
					validators.UUID{},
				},
			},
			"actor": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed:            true,
						Description:         "the owner's id",
						MarkdownDescription: "the owner's id",
					},
					"email": schema.StringAttribute{
						Computed:            true,
						Description:         "the owner's email",
						MarkdownDescription: "the owner's email",
					},
				},
				Computed: true,
				Description: "Define the owner of the permission, which will be used for administrative " +
					"purposes and approval workflows.",
				MarkdownDescription: "Define the owner of the permission, which will be used for administrative " +
					"purposes and approval workflows.",
			},
			"role": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed:            true,
						Description:         "the owner's id",
						MarkdownDescription: "the owner's id",
					},
					"name": schema.StringAttribute{
						Computed:            true,
						Description:         "The application's name",
						MarkdownDescription: "The application's name",
					},
					"resource": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"id": schema.StringAttribute{
								Computed:            true,
								Description:         "Resource's unique identifier.",
								MarkdownDescription: "Resource's unique identifier.",
							},
							"name": schema.StringAttribute{
								Computed:            true,
								Description:         "Resource's name.",
								MarkdownDescription: "Resource's name.",
							},
							"integration": schema.SingleNestedAttribute{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										Computed:            true,
										Description:         "Integration's unique identifier.",
										MarkdownDescription: "Integration's unique identifier.",
									},
									"name": schema.StringAttribute{
										Computed:            true,
										Description:         "Integration's name.",
										MarkdownDescription: "Integration's name.",
									},
									"application": schema.SingleNestedAttribute{
										Attributes: map[string]schema.Attribute{
											"name": schema.StringAttribute{
												Computed:            true,
												Description:         "Name of the application.",
												MarkdownDescription: "Name of the application.",
											},
										},
										Computed:            true,
										Description:         "Application of the integration.",
										MarkdownDescription: "Application of the integration.",
									},
								},
								Computed:            true,
								Description:         "Integration related to the resource.",
								MarkdownDescription: "Integration related to the resource.",
							},
						},
						Computed:            true,
						Description:         "The resource associated with the role.",
						MarkdownDescription: "The resource associated with the role.",
					},
				},
				Computed: true,
				Description: "The application the permission connects to must be chosen from the list " +
					"of supported applications.",
				MarkdownDescription: "The application the permission connects to must be chosen from the list " +
					"of supported applications.",
			},
			"path": schema.StringAttribute{
				Computed:            true,
				Description:         "go to https://app.entitle.io/permissions and provide the latest schema.",
				MarkdownDescription: "go to https://app.entitle.io/permissions and provide the latest schema.",
			},
			"types": schema.SetAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "As the admin, you can set different durations for the permission, " +
					"compared to the workflow linked to it.",
				MarkdownDescription: "As the admin, you can set different durations for the permission, " +
					"compared to the workflow linked to it.",
			},
		},
	}
}

// Configure configures the data source with the provided client.
func (d *PermissionDataSource) Configure(
	ctx context.Context,
	req datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse,
) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.ClientWithResponses)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = c
}

// Read retrieves data from the provider and populates the data source model.
func (d *PermissionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PermissionDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uid, err := uuid.Parse(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("failed to parse permission id to uuid format, got error: %s", err),
		)
		return
	}

	permissionResp, err := d.client.PermissionsIndexWithResponse(ctx, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to get the permission by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(permissionResp.HTTPResponse.StatusCode, permissionResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"Failed to get the Permission by the id (%s), status code: %d, %s",
				uid.String(),
				permissionResp.HTTPResponse.StatusCode,
				err.Error(),
			),
		)
		return
	}

	for _, permission := range permissionResp.JSON200.Result {
		if permission.PermissionId.String() == uid.String() {
			// Extract and convert types from the API response
			allowedTypes, advDiags := GetTypesFromResponse(permission.Types)
			if advDiags.HasError() {
				resp.Diagnostics.Append(advDiags...)
				return
			}

			resource, diags := utils.RoleResource{
				Id:   utils.TrimmedStringValue(permission.Role.Resource.Id.String()),
				Name: utils.TrimmedStringValue(permission.Role.Resource.Name),
				Integration: utils.RoleResourceIntegration{
					Id:   utils.TrimmedStringValue(permission.Role.Resource.Integration.Id.String()),
					Name: utils.TrimmedStringValue(permission.Role.Resource.Integration.Name),
					Application: utils.NameModel{
						Name: utils.TrimmedStringValue(permission.Role.Resource.Integration.Application.Name),
					},
				},
			}.AsObjectValue(ctx)
			if diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}

			data = PermissionDataSourceModel{
				ID: utils.TrimmedStringValue(permission.PermissionId.String()),
				Actor: &utils.IdEmailModel{
					Id:    utils.TrimmedStringValue(permission.Account.Id.String()),
					Email: utils.TrimmedStringValue(permission.Account.Email),
				},
				Role: &utils.Role{
					ID:       utils.TrimmedStringValue(permission.Role.Id.String()),
					Name:     utils.TrimmedStringValue(permission.Role.Name),
					Resource: resource,
				},
				Path:  utils.TrimmedStringValue(string(permission.Path)),
				Types: allowedTypes,
			}

			break
		}
	}

	if data.ID.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to get the permission by the id (%s), got error: %s", uid.String(), "not found"),
		)
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	diagsResult := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diagsResult...)
	if resp.Diagnostics.HasError() {
		return
	}
}
