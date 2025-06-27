package resources

import (
	"context"
	"fmt"
	"net/http"
	"strings"

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

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ResourceDataSource{}

// ResourceDataSource defines the data source implementation.
type ResourceDataSource struct {
	client *client.ClientWithResponses
}

// NewResourceDataSource creates a new instance of ResourceDataSource.
func NewResourceDataSource() datasource.DataSource {
	return &ResourceDataSource{}
}

// ResourceDataSourceModel describes the data source data model.
type ResourceDataSourceModel struct {
	Id                     types.String             `tfsdk:"id"`
	Workflow               *utils.IdNameModel       `tfsdk:"workflow"`
	Maintainers            []*utils.MaintainerModel `tfsdk:"maintainers"`
	Integration            types.Object             `tfsdk:"integration"`
	Tags                   types.List               `tfsdk:"tags"`
	UserDefinedTags        types.List               `tfsdk:"user_defined_tags"`
	Owner                  *utils.IdEmailModel      `tfsdk:"owner"`
	Name                   types.String             `tfsdk:"name"`
	Description            types.String             `tfsdk:"description"`
	UserDefinedDescription types.String             `tfsdk:"user_defined_description"`
	AllowedDurations       types.List               `tfsdk:"allowed_durations"`
	Requestable            types.Bool               `tfsdk:"requestable"`
	AllowAsGrantMethod     types.Bool               `tfsdk:"allow_as_grant_method"`
}

// Metadata sets the metadata for the data source.
func (d *ResourceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource"
}

// Schema defines the data source schema.
func (d *ResourceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Defines an Entitle Resource, which represents a target system or asset that can be accessed or governed through Entitle. The schema includes metadata, ownership, integration, workflow, and access management configuration. [Read more about resources](https://docs.beyondtrust.com/entitle/docs/integrations-resources-roles).",
		Description:         "Defines an Entitle Resource, which represents a target system or asset that can be accessed or governed through Entitle. The schema includes metadata, ownership, integration, workflow, and access management configuration. [Read more about resources](https://docs.beyondtrust.com/entitle/docs/integrations-resources-roles).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Entitle Resource identifier in uuid format",
				Description:         "Entitle Resource identifier in uuid format",
				Validators: []validator.String{
					validators.UUID{},
				},
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Entitle Resource name",
				Description:         "Entitle Resource name",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource description",
				Description:         "Resource description",
			},
			"user_defined_description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Custom description provided by the user",
				Description:         "Custom description provided by the user",
			},
			"tags": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Any meta-data searchable tags should be added here, " +
					"like “accounting”, “ATL_Marketing” or “Production_Line_14”.",
				MarkdownDescription: "Any meta-data searchable tags should be added here, " +
					"like “accounting”, “ATL_Marketing” or “Production_Line_14”.",
			},
			"user_defined_tags": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Any meta-data searchable tags should be added here, " +
					"like “accounting”, “ATL_Marketing” or “Production_Line_14”.",
				MarkdownDescription: "Any meta-data searchable tags should be added here, " +
					"like “accounting”, “ATL_Marketing” or “Production_Line_14”.",
			},
			"allowed_durations": schema.ListAttribute{
				ElementType:         types.NumberType,
				Computed:            true,
				Description:         "List of allowed access durations",
				MarkdownDescription: "List of allowed access durations",
			},
			"requestable": schema.BoolAttribute{
				Computed:            true,
				Description:         "Indicates if the resource is requestable (default: true)",
				MarkdownDescription: "Indicates if the resource is requestable (default: true)",
			},
			"owner": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed:            true,
						Description:         "Owner's unique identifier",
						MarkdownDescription: "Owner's unique identifier",
					},
					"email": schema.StringAttribute{
						Computed:            true,
						Description:         "Owner's email address",
						MarkdownDescription: "Owner's email address",
					},
				},
				Computed:            true,
				Description:         "Owner of the resource",
				MarkdownDescription: "Owner of the resource",
			},
			"workflow": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed:            true,
						Description:         "Workflow's unique identifier",
						MarkdownDescription: "Workflow's unique identifier",
					},
					"name": schema.StringAttribute{
						Computed:            true,
						Description:         "Workflow's name",
						MarkdownDescription: "Workflow's name",
					},
				},
				Computed:            true,
				Description:         "Workflow configuration for the resource",
				MarkdownDescription: "Workflow configuration for the resource",
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
								Description:         "Name of the application within the integration",
								MarkdownDescription: "Name of the application within the integration",
							},
						},
						Computed:            true,
						Description:         "Integration's application",
						MarkdownDescription: "Integration's application",
					},
				},
				Computed:            true,
				Description:         "Integration the resource belongs to",
				MarkdownDescription: "Integration the resource belongs to",
			},
			"maintainers": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Computed:            true,
							Description:         "Type of maintainer ",
							MarkdownDescription: "Type of maintainer ",
						},
						"user": schema.SingleNestedAttribute{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Computed:            true,
									Description:         "Maintainer user’s unique identifier",
									MarkdownDescription: "Maintainer user’s unique identifier",
								},
								"email": schema.StringAttribute{
									Computed:            true,
									Description:         "Maintainer user’s email address",
									MarkdownDescription: "Maintainer user’s email address",
								},
							},
							Computed:            true,
							Description:         "Maintainer details if the maintainer is a user",
							MarkdownDescription: "Maintainer details if the maintainer is a user",
						},
						"group": schema.SingleNestedAttribute{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Computed:            true,
									Description:         "Maintainer group’s unique identifier",
									MarkdownDescription: "Maintainer group’s unique identifier",
								},
								"email": schema.StringAttribute{
									Computed:            true,
									Description:         "Maintainer group’s email address",
									MarkdownDescription: "Maintainer group’s email address",
								},
							},
							Computed:            true,
							Description:         "Maintainer details if the maintainer is a group",
							MarkdownDescription: "Maintainer details if the maintainer is a group",
						},
					},
				},
				Computed:            true,
				Description:         "List of resource maintainers",
				MarkdownDescription: "List of resource maintainers",
			},
		},
	}
}

// Configure configures the data source with the provided client.
func (d *ResourceDataSource) Configure(
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
func (d *ResourceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ResourceDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uid, err := uuid.Parse(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("failed to parse Resource id to uuid format, got error: %s", err),
		)
		return
	}

	resourceResp, err := d.client.ResourcesShowWithResponse(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to get the Resource by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	if resourceResp.HTTPResponse.StatusCode != 200 {
		errBody, _ := utils.GetErrorBody(resourceResp.Body)
		if resourceResp.HTTPResponse.StatusCode == http.StatusUnauthorized ||
			(resourceResp.HTTPResponse.StatusCode == http.StatusBadRequest && strings.Contains(errBody.GetMessage(), "is not a valid uuid")) {
			resp.Diagnostics.AddError(
				"Client Error",
				"unauthorized token, update the entitle token and retry please",
			)
			return
		}

		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"failed to get the Resource by the id (%s), status code: %d",
				uid.String(),
				resourceResp.HTTPResponse.StatusCode,
			),
		)
		return
	}

	tags, diagsTags := utils.GetStringList(resourceResp.JSON200.Result.Tags)
	resp.Diagnostics.Append(diagsTags...)
	if resp.Diagnostics.HasError() {
		return
	}

	userDefinedTags, diagsTags := utils.GetStringList(resourceResp.JSON200.Result.UserDefinedTags)
	resp.Diagnostics.Append(diagsTags...)
	if resp.Diagnostics.HasError() {
		return
	}

	allowedDurationsValues := utils.GetAllowedDurations(resourceResp.JSON200.Result.AllowedDurations)
	allowedDurations, diags := types.ListValue(types.NumberType, allowedDurationsValues)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var owner *utils.IdEmailModel

	if resourceResp.JSON200.Result.Owner != nil {
		ownerEmailString, err := utils.GetEmailString(resourceResp.JSON200.Result.Owner.Email)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to convert the owner email to string",
				err.Error(),
			)
			return
		}

		owner = &utils.IdEmailModel{
			Id:    utils.TrimmedStringValue(resourceResp.JSON200.Result.Owner.Id.String()),
			Email: utils.TrimmedStringValue(ownerEmailString),
		}
	}

	var workflow *utils.IdNameModel
	if resourceResp.JSON200.Result.Workflow != nil {
		workflow = &utils.IdNameModel{
			ID:   utils.TrimmedStringValue(resourceResp.JSON200.Result.Workflow.Id.String()),
			Name: utils.TrimmedStringValue(resourceResp.JSON200.Result.Workflow.Name),
		}
	}

	integration := utils.RoleResourceIntegration{
		Id:   utils.TrimmedStringValue(resourceResp.JSON200.Result.Integration.Id.String()),
		Name: utils.TrimmedStringValue(resourceResp.JSON200.Result.Integration.Name),
		Application: utils.NameModel{
			Name: utils.TrimmedStringValue(resourceResp.JSON200.Result.Integration.Application.Name),
		},
	}

	integrationObkect, diags := integration.AsObjectValue(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	maintainers, diags := utils.GetMaintainers(ctx, resourceResp.JSON200.Result.Maintainers)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data = ResourceDataSourceModel{
		Id:                     utils.TrimmedStringValue(resourceResp.JSON200.Result.Id.String()),
		Name:                   utils.TrimmedStringValue(resourceResp.JSON200.Result.Name),
		AllowedDurations:       allowedDurations,
		Requestable:            types.BoolValue(resourceResp.JSON200.Result.Requestable),
		Tags:                   tags,
		UserDefinedTags:        userDefinedTags,
		Description:            utils.TrimmedStringValue(utils.StringValue(resourceResp.JSON200.Result.Description)),
		UserDefinedDescription: utils.TrimmedStringValue(utils.StringValue(resourceResp.JSON200.Result.UserDefinedDescription)),
		Owner:                  owner,
		Workflow:               workflow,
		Integration:            integrationObkect,
		Maintainers:            maintainers,
	}

	tflog.Trace(ctx, "read a entitle resource data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	diagsResult := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diagsResult...)
	if resp.Diagnostics.HasError() {
		return
	}
}
