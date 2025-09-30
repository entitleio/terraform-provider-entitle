// Package roles provides the implementation of the Entitle Role data source for Terraform.
// This data source allows Terraform to query information about Roles from the Entitle API.
package roles

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"github.com/entitleio/terraform-provider-entitle/internal/validators"
)

// Ensure that the types defined by the provider satisfy framework interfaces.
var _ datasource.DataSource = &RoleDataSource{}

// RoleDataSource defines the implementation of the data source.
type RoleDataSource struct {
	client *client.ClientWithResponses
}

// NewRoleDataSource creates a new instance of RoleDataSource.
func NewRoleDataSource() datasource.DataSource {
	return &RoleDataSource{}
}

// RoleDataSourceModel describes the data source data model.
type RoleDataSourceModel struct {
	ID                      types.String                        `tfsdk:"id"`
	Name                    types.String                        `tfsdk:"name"`
	Resource                *utils.IdNameModel                  `tfsdk:"resource" json:"resource"`
	AllowedDurations        types.Set                           `tfsdk:"allowed_durations"`
	Workflow                *utils.IdNameModel                  `tfsdk:"workflow"`
	PrerequisitePermissions []utils.PrerequisitePermissionModel `tfsdk:"prerequisite_permissions"`
	VirtualizedRole         *utils.IdNameModel                  `tfsdk:"virtualized_role"`
	Requestable             types.Bool                          `tfsdk:"requestable"`
}

// Metadata sets the metadata for the data source.
func (d *RoleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

// Schema sets the schema for the data source.
func (d *RoleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Defines the schema for an Entitle Role resource.",
		Description:         "Defines the schema for an Entitle Role resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Entitle Role identifier in UUID format",
				Description:         "Entitle Role identifier in UUID format",
				Validators: []validator.String{
					validators.UUID{},
				},
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The display name for Entitle Role.",
				Description:         "The display name for Entitle Role.",
			},
			"resource": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed:            true,
						Description:         "The unique ID of the resource assigned to the role.",
						MarkdownDescription: "The unique ID of the resource assigned to the role.",
					},
					"name": schema.StringAttribute{
						Computed:            true,
						Description:         "The name of the assigned resource.",
						MarkdownDescription: "The name of the assigned resource.",
					},
				},
				Computed:            true,
				Description:         "The resource associated with the role.",
				MarkdownDescription: "The resource associated with the role.",
			},
			"allowed_durations": schema.SetAttribute{
				ElementType: types.NumberType,
				Computed:    true,
				Description: "As the admin, you can set different durations for the role, " +
					"compared to the workflow linked to it.",
				MarkdownDescription: "As the admin, you can set different durations for the role, " +
					"compared to the workflow linked to it.",
			},
			"workflow": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					// Attribute: workflow id
					"id": schema.StringAttribute{
						Required:            true,
						Description:         "The unique ID of the workflow assigned to the role.",
						MarkdownDescription: "The unique ID of the workflow assigned to the role.",
					},
					// Attribute: workflow name
					"name": schema.StringAttribute{
						Computed:            true,
						Description:         "The name of the assigned workflow.",
						MarkdownDescription: "The name of the assigned workflow.",
					},
				},
				Computed:            true,
				Description:         "In this field, you can assign an existing workflow to the new role.",
				MarkdownDescription: "In this field, you can assign an existing workflow to the new role.",
			},
			"prerequisite_permissions": schema.ListNestedAttribute{
				Computed:            true,
				Description:         "Users granted any role from this role through a request will automatically receive the permissions to the roles selected below.",
				MarkdownDescription: "Users granted any role from this role through a request will automatically receive the permissions to the roles selected below.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"default": schema.BoolAttribute{
							Optional:            true,
							Computed:            true,
							Description:         "Indicates whether this prerequisite permission should be automatically granted as a default permission. When set to true, users will receive this permission by default when accessing the associated resource (default: false).",
							MarkdownDescription: "Indicates whether this prerequisite permission should be automatically granted as a default permission. When set to true, users will receive this permission by default when accessing the associated resource (default: false).",
						},
						"role": schema.SingleNestedAttribute{
							Required: true,
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Required:            true,
									Description:         "The identifier of the role to be granted.",
									MarkdownDescription: "The identifier of the role to be granted.",
								},
								"name": schema.StringAttribute{
									Computed:            true,
									Description:         "The name of the role.",
									MarkdownDescription: "The name of the role.",
								},
								"resource": schema.SingleNestedAttribute{
									Attributes: map[string]schema.Attribute{
										"id": schema.StringAttribute{
											Computed:            true,
											Description:         "The unique identifier of the resource.",
											MarkdownDescription: "The unique identifier of the resource.",
										},
										"name": schema.StringAttribute{
											Computed:            true,
											Description:         "The display name of the resource.",
											MarkdownDescription: "The display name of the resource.",
										},
										"integration": schema.SingleNestedAttribute{
											Attributes: map[string]schema.Attribute{
												"id": schema.StringAttribute{
													Computed:            true,
													Description:         "The identifier of the integration.",
													MarkdownDescription: "The identifier of the integration.",
												},
												"name": schema.StringAttribute{
													Computed:            true,
													Description:         "The display name of the integration.",
													MarkdownDescription: "The display name of the integration.",
												},
												"application": schema.SingleNestedAttribute{
													Attributes: map[string]schema.Attribute{
														"name": schema.StringAttribute{
															Computed:            true,
															Description:         "The name of the connected application.",
															MarkdownDescription: "The name of the connected application.",
														},
													},
													Computed:            true,
													Description:         "The application that the integration is connected to.",
													MarkdownDescription: "The application that the integration is connected to.",
												},
											},
											Computed:            true,
											Description:         "The integration that the resource belongs to.",
											MarkdownDescription: "The integration that the resource belongs to.",
										},
									},
									Computed:            true,
									Description:         "The specific resource associated with the role.",
									MarkdownDescription: "The specific resource associated with the role.",
								},
							},
						},
					},
				},
			},
			"virtualized_role": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					// Attribute: workflow id
					"id": schema.StringAttribute{
						Computed:            true,
						Description:         "The unique ID of the virtualized role assigned to the role.",
						MarkdownDescription: "The unique ID of the virtualized role assigned to the role.",
					},
					// Attribute: workflow name
					"name": schema.StringAttribute{
						Computed:            true,
						Description:         "The name of the assigned virtualized role.",
						MarkdownDescription: "The name of the assigned virtualized role.",
					},
				},
				Computed:            true,
				Description:         "In this field, you can assign an existing virtualized role to the new role.",
				MarkdownDescription: "In this field, you can assign an existing virtualized role to the new role.",
			},
			"requestable": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Indicates if the role is requestable (default: true)",
				Description:         "Indicates if the role is requestable (default: true)",
			},
		},
	}
}

// Configure configures the data source with the provided client.
func (d *RoleDataSource) Configure(
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

// Read retrieves data from the Entitle API and populates the data source state.
// This function is responsible for fetching details about an Role from the Entitle API
// based on the provided Terraform configuration. It reads the configuration data into a model,
// sends a request to the Entitle API, and processes the API response. The retrieved data is then
// saved into Terraform state for further use in the Terraform plan.
func (d *RoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Create a model to hold data from Terraform configuration
	var data RoleDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the Role ID from the configuration model
	uid, err := uuid.Parse(data.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Failed to parse the resource ID (%s) to UUID, got error: %s", data.ID.String(), err),
		)
		return
	}

	// Fetch Role details from the Entitle API
	apiResp, err := d.client.RolesShowWithResponse(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ApiConnectionError.Error(),
			fmt.Sprintf("Unable to get the Role by the ID (%s), got error: %s", uid.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(apiResp.StatusCode(), apiResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ApiResponseError.Error(),
			fmt.Sprintf(
				"Failed to get the Role by the ID (%s), %s",
				uid.String(),
				err.Error(),
			),
		)
		return
	}

	var workflow *utils.IdNameModel
	if apiResp.JSON200.Result.Workflow != nil {
		workflow = &utils.IdNameModel{
			ID:   utils.TrimmedStringValue(apiResp.JSON200.Result.Workflow.Id.String()),
			Name: utils.TrimmedStringValue(apiResp.JSON200.Result.Workflow.Name),
		}
	}

	var virtualizedRole *utils.IdNameModel
	if apiResp.JSON200.Result.VirtualizedRole != nil {
		virtualizedRole = &utils.IdNameModel{
			ID:   utils.TrimmedStringValue(apiResp.JSON200.Result.VirtualizedRole.Id.String()),
			Name: utils.TrimmedStringValue(apiResp.JSON200.Result.VirtualizedRole.Name),
		}
	}

	var prerequisitePermissions []utils.PrerequisitePermissionModel
	if apiResp.JSON200.Result.PrerequisitePermissions != nil {
		items := *apiResp.JSON200.Result.PrerequisitePermissions
		prerequisitePermissions = make([]utils.PrerequisitePermissionModel, len(items))
		for i, item := range items {
			v, err := item.AsPrerequisiteRolePermissionResponseSchema()
			if err != nil {
				resp.Diagnostics.AddError(
					"No data",
					fmt.Sprintf("Failed to unmarshal the prerequisite permissions data, err: %s", err.Error()),
				)
				return
			}

			roleModel, diagsGetRoles := utils.GetRole(ctx, v.Role.Id.String(), v.Role.Name, v.Role.Resource)
			if diagsGetRoles.HasError() {
				resp.Diagnostics.Append(diagsGetRoles...)
				return
			}

			prerequisitePermissions[i] =
				utils.PrerequisitePermissionModel{
					Default: types.BoolValue(v.Default),
					Role:    roleModel,
				}
		}
	}

	allowedDurations, diags := utils.GetNumberSetFromAllowedDurations(apiResp.JSON200.Result.AllowedDurations)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Populate the data model with details from the API response
	data = RoleDataSourceModel{
		ID:   utils.TrimmedStringValue(apiResp.JSON200.Result.Id.String()),
		Name: utils.TrimmedStringValue(apiResp.JSON200.Result.Name),
		Resource: &utils.IdNameModel{
			ID:   utils.TrimmedStringValue(apiResp.JSON200.Result.Resource.Id.String()),
			Name: utils.TrimmedStringValue(apiResp.JSON200.Result.Resource.Name),
		},
		AllowedDurations:        allowedDurations,
		Workflow:                workflow,
		PrerequisitePermissions: prerequisitePermissions,
		VirtualizedRole:         virtualizedRole,
		Requestable:             types.BoolValue(apiResp.JSON200.Result.Requestable),
	}

	// Log a trace message indicating a successful read of the Entitle Role data source
	tflog.Trace(ctx, "Read an Entitle Role data source")

	// Save the retrieved data into Terraform state
	diagsResult := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diagsResult...)
	if resp.Diagnostics.HasError() {
		return
	}
}
