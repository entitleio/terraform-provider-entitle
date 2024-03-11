package policies

import (
	"context"
	"fmt"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"github.com/entitleio/terraform-provider-entitle/internal/validators"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
	ID       types.String          `tfsdk:"id" json:"id"`
	Bundles  []*utils.IdNameModel  `tfsdk:"bundles" json:"bundles"`
	InGroups []*PolicyInGroupModel `tfsdk:"in_groups" json:"inGroups"`
	Roles    []*utils.Role         `tfsdk:"roles" json:"roles"`
}

// Metadata sets the data source's metadata, such as its type name.
func (d *PolicyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

// Schema sets the schema for the data source.
func (d *PolicyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Entitle policy is a rule which manages users birthright permissions automatically, " +
			"a group of users is entitled to a set of permissions. When a user joins the group, e.g. upon joining " +
			"the organization, he will be granted with the permissions defined for the group automatically, and " +
			"upon leaving the group, e.g. leaving the organization, the permissions will be revoked automatically.",
		Description: "Entitle policy is a rule which manages users birthright permissions automatically, " +
			"a group of users is entitled to a set of permissions. When a user joins the group, e.g. upon joining " +
			"the organization, he will be granted with the permissions defined for the group automatically, and " +
			"upon leaving the group, e.g. leaving the organization, the permissions will be revoked automatically.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Entitle Policy identifier in uuid format",
				Description:         "Entitle Policy identifier in uuid format",
				Validators: []validator.String{
					validators.UUID{},
				},
			},
			"roles": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							Description:         "id",
							MarkdownDescription: "id",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							Description:         "name",
							MarkdownDescription: "name",
						},
						"resource": schema.SingleNestedAttribute{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Computed:            true,
									Description:         "id",
									MarkdownDescription: "id",
								},
								"name": schema.StringAttribute{
									Computed:            true,
									Description:         "name",
									MarkdownDescription: "name",
								},
								"integration": schema.SingleNestedAttribute{
									Attributes: map[string]schema.Attribute{
										"id": schema.StringAttribute{
											Computed:            true,
											Description:         "id",
											MarkdownDescription: "id",
										},
										"name": schema.StringAttribute{
											Computed:            true,
											Description:         "name",
											MarkdownDescription: "name",
										},
										"application": schema.SingleNestedAttribute{
											Attributes: map[string]schema.Attribute{
												"name": schema.StringAttribute{
													Computed:            true,
													Description:         "name",
													MarkdownDescription: "name",
												},
											},
											Computed:            true,
											Description:         "integration",
											MarkdownDescription: "integration",
										},
									},
									Computed:            true,
									Description:         "integration",
									MarkdownDescription: "integration",
								},
							},
							Computed:            true,
							Description:         "resource",
							MarkdownDescription: "resource",
						},
					},
				},
				Computed:            true,
				Description:         "roles",
				MarkdownDescription: "roles",
			},
			"bundles": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							Description:         "bundle's id",
							MarkdownDescription: "bundle's id",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							Description:         "bundle's name",
							MarkdownDescription: "bundle's name",
						},
					},
				},
				Computed:            true,
				Description:         "bundles",
				MarkdownDescription: "bundles",
			},
			"in_groups": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							Description:         "id",
							MarkdownDescription: "id",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							Description:         "name",
							MarkdownDescription: "name",
						},
						"type": schema.StringAttribute{
							Computed:            true,
							Description:         "type",
							MarkdownDescription: "type",
						},
					},
				},
				Computed:            true,
				Description:         "roles",
				MarkdownDescription: "roles",
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
			"Client Error",
			fmt.Sprintf("Unable to get the bundle by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	if policyResp.HTTPResponse.StatusCode != 200 {
		errBody, _ := utils.GetErrorBody(policyResp.Body)
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"failed to get the policy by the id (%s), status code: %d%s",
				uid.String(),
				policyResp.HTTPResponse.StatusCode,
				errBody.GetMessage(),
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
		ID:       utils.TrimmedStringValue(data.Id.String()),
		Roles:    roles,
		Bundles:  getBundles(data.Bundles),
		InGroups: getInGroups(data.InGroups),
	}, diags
}
