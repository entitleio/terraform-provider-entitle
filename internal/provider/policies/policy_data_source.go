package policies

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"github.com/entitleio/terraform-provider-entitle/internal/validators"
)

// Ensure that the provider-defined types fully satisfy the framework interfaces.
var _ datasource.DataSource = &PolicyDataSource{}

// PolicyDataSource defines the data source implementation for the Terraform provider.
type PolicyDataSource struct {
	client *client.ClientWithResponses
}

// NewPolicyDataSource creates a new instance of the PolicyDataSource.
func NewPolicyDataSource() datasource.DataSource {
	return &PolicyDataSource{}
}

// PolicyDataSourceModel defines the data model for FullPolicyResultResponseSchema.
type PolicyDataSourceModel struct {
	ID        types.String          `tfsdk:"id" json:"id"`
	Bundles   []*utils.IdNameModel  `tfsdk:"bundles" json:"bundles"`
	InGroups  []*PolicyInGroupModel `tfsdk:"in_groups" json:"inGroups"`
	Roles     []*utils.Role         `tfsdk:"roles" json:"roles"`
	Number    types.Int64           `tfsdk:"number"`
	SortOrder types.Int64           `tfsdk:"sort_order" json:"sortOrder"`
}

// Metadata sets the data source's metadata, such as its type name.
func (d *PolicyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

// Schema sets the schema for the data source.
func (d *PolicyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "An Entitle policy is a rule that automatically manages users' birthright " +
			"permissions. It assigns a predefined set of permissions to a group of users. When a user " +
			"joins the group—such as when they join the organization—they are automatically granted the " +
			"group's permissions. Conversely, when they leave the group—such as when they leave the " +
			"organization—those permissions are automatically revoked." +
			"[Read more about policies](https://docs.beyondtrust.com/entitle/docs/birthright-policies).",
		Description: "An Entitle policy is a rule that automatically manages users' birthright " +
			"permissions. It assigns a predefined set of permissions to a group of users. When a user " +
			"joins the group—such as when they join the organization—they are automatically granted the " +
			"group's permissions. Conversely, when they leave the group—such as when they leave the " +
			"organization—those permissions are automatically revoked." +
			"[Read more about policies](https://docs.beyondtrust.com/entitle/docs/birthright-policies).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Entitle Policy identifier in uuid format",
				Description:         "Entitle Policy identifier in uuid format",
				Validators: []validator.String{
					validators.UUID{},
				},
			},
			"number": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Entitle Policy number",
				Description:         "Entitle Policy number",
			},
			"sort_order": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Entitle Policy sort order",
				Description:         "Entitle Policy sort order",
			},
			"roles": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							Description:         "Role's unique identifier",
							MarkdownDescription: "Role's unique identifier",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							Description:         "Name of the role",
							MarkdownDescription: "Name of the role",
						},
						"resource": schema.SingleNestedAttribute{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Computed:            true,
									Description:         "Resource's unique identifier",
									MarkdownDescription: "Resource's unique identifier",
								},
								"name": schema.StringAttribute{
									Computed:            true,
									Description:         "Resource's name",
									MarkdownDescription: "Resource's name",
								},
								"integration": schema.SingleNestedAttribute{
									Attributes: map[string]schema.Attribute{
										"id": schema.StringAttribute{
											Computed:            true,
											Description:         "Integration's unique identifier",
											MarkdownDescription: "Integration's unique identifier",
										},
										"name": schema.StringAttribute{
											Computed:            true,
											Description:         "Integration's name",
											MarkdownDescription: "Integration's name",
										},
										"application": schema.SingleNestedAttribute{
											Attributes: map[string]schema.Attribute{
												"name": schema.StringAttribute{
													Computed:            true,
													Description:         "Application's name",
													MarkdownDescription: "Application's name",
												},
											},
											Computed:            true,
											Description:         "The application within the integration",
											MarkdownDescription: "The application within the integration",
										},
									},
									Computed:            true,
									Description:         "The integration associated with the resource",
									MarkdownDescription: "The integration associated with the resource",
								},
							},
							Computed:            true,
							Description:         "The resource to which this role grants access",
							MarkdownDescription: "The resource to which this role grants access",
						},
					},
				},
				Computed:            true,
				Description:         "List of roles granted by the policy",
				MarkdownDescription: "List of roles granted by the policy",
			},
			"bundles": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							Description:         "Bundle's unique identifier",
							MarkdownDescription: "Bundle's unique identifier",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							Description:         "Bundle's name",
							MarkdownDescription: "Bundle's name",
						},
					},
				},
				Computed:            true,
				Description:         "List of bundles granted by the policy",
				MarkdownDescription: "List of bundles granted by the policy",
			},
			"in_groups": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							Description:         "Group's unique identifier",
							MarkdownDescription: "Group's unique identifier",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							Description:         "Group's name",
							MarkdownDescription: "Group's name",
						},
						"type": schema.StringAttribute{
							Computed:            true,
							Description:         "Group type",
							MarkdownDescription: "Group type",
						},
					},
				},
				Computed:            true,
				Description:         "List of groups that trigger the policy",
				MarkdownDescription: "List of groups that trigger the policy",
			},
		},
	}
}

// Configure configures the data source with the provider's client.
func (d *PolicyDataSource) Configure(
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

// Read reads data from the external source and sets it in Terraform state.
func (d *PolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PolicyDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uid, err := uuid.Parse(data.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("failed to parse the entitle policy id (%s) to UUID, got error: %s", data.ID.String(), err),
		)
		return
	}

	policyResp, err := d.client.PoliciesShowWithResponse(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ApiConnectionError.Error(),
			fmt.Sprintf("Unable to get the Policy by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(policyResp.HTTPResponse.StatusCode, policyResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ApiResponseError.Error(),
			fmt.Sprintf(
				"Failed to get the Policy by the id (%s), status code: %d, %s",
				uid.String(),
				policyResp.HTTPResponse.StatusCode,
				err.Error(),
			),
		)
		return
	}

	// Convert the policy response to the data source model
	data, diags := converterPolicyToDataSource(ctx, &policyResp.JSON200.Result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read an entitle policy data source")

	// Save data into Terraform state
	diagsResult := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diagsResult...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// converterPolicyToDataSource converts the API response to the data source model.
func converterPolicyToDataSource(
	ctx context.Context,
	data *client.FullPolicyResultResponseSchema,
) (PolicyDataSourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	if data == nil {
		diags.AddError(
			"No data",
			"failed the given schema data is nil",
		)

		return PolicyDataSourceModel{}, diags
	}

	// Extract and convert roles from the API response
	roles, diagsRoles := getRoles(ctx, data.Roles)
	diags.Append(diagsRoles...)
	if diags.HasError() {
		return PolicyDataSourceModel{}, diags
	}

	// Create the Terraform resource model using the extracted data
	return PolicyDataSourceModel{
		ID:        utils.TrimmedStringValue(data.Id.String()),
		Roles:     roles,
		Bundles:   getBundles(data.Bundles),
		InGroups:  getInGroups(data.InGroups),
		Number:    basetypes.NewInt64Value(int64(data.Number)),
		SortOrder: basetypes.NewInt64Value(int64(data.SortOrder)),
	}, diags
}
