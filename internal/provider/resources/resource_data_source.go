package resources

import (
	"context"
	"fmt"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"github.com/entitleio/terraform-provider-entitle/internal/validators"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
	AllowRequests          types.Bool               `tfsdk:"allow_requests"`
	AllowAsGrantMethod     types.Bool               `tfsdk:"allow_as_grant_method"`
}

// Metadata sets the metadata for the data source.
func (d *ResourceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource"
}

// Schema defines the data source schema.
func (d *ResourceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Entitle Resource Description",
		Description:         "Entitle Resource Description",
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
				MarkdownDescription: "description",
				Description:         "description",
			},
			"user_defined_description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "user_defined_description",
				Description:         "user_defined_description",
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
				Description:         "allowedDurations",
				MarkdownDescription: "allowedDurations",
			},
			"allow_requests": schema.BoolAttribute{
				Computed:            true,
				Description:         "allowRequests (default: true)",
				MarkdownDescription: "allowRequests (default: true)",
			},
			"allow_as_grant_method": schema.BoolAttribute{
				Computed:            true,
				Description:         "allowAsGrantMethod (default: true)",
				MarkdownDescription: "allowAsGrantMethod (default: true)",
			},
			"owner": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed:            true,
						Description:         "id",
						MarkdownDescription: "id",
					},
					"email": schema.StringAttribute{
						Computed:            true,
						Description:         "email",
						MarkdownDescription: "email",
					},
				},
				Computed:            true,
				Description:         "owner",
				MarkdownDescription: "owner",
			},
			"workflow": schema.SingleNestedAttribute{
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
				},
				Computed:            true,
				Description:         "workflow",
				MarkdownDescription: "workflow",
			},
			"integration": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed:            true,
						Description:         "integration's id",
						MarkdownDescription: "integration's id",
					},
					"name": schema.StringAttribute{
						Computed:            true,
						Description:         "integration's name",
						MarkdownDescription: "integration's name",
					},
					"application": schema.SingleNestedAttribute{
						Attributes: map[string]schema.Attribute{
							"name": schema.StringAttribute{
								Computed:            true,
								Description:         "application's name",
								MarkdownDescription: "application's name",
							},
						},
						Computed:            true,
						Description:         "integration's application",
						MarkdownDescription: "integration's application",
					},
				},
				Computed:            true,
				Description:         "integration",
				MarkdownDescription: "integration",
			},
			"maintainers": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Computed:            true,
							Description:         "type",
							MarkdownDescription: "",
						},
						"user": schema.SingleNestedAttribute{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Computed:            true,
									Description:         "id",
									MarkdownDescription: "",
								},
								"email": schema.StringAttribute{
									Computed:            true,
									Description:         "email",
									MarkdownDescription: "email",
								},
							},
							Computed:            true,
							Description:         "user",
							MarkdownDescription: "user",
						},
						"group": schema.SingleNestedAttribute{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Computed:            true,
									Description:         "",
									MarkdownDescription: "",
								},
								"email": schema.StringAttribute{
									Computed:            true,
									Description:         "email",
									MarkdownDescription: "email",
								},
							},
							Computed:            true,
							Description:         "group",
							MarkdownDescription: "group",
						},
					},
				},
				Computed:            true,
				Description:         "maintainers",
				MarkdownDescription: "maintainers",
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
	if resourceResp.JSON200.Result.Owner != nil {
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
		AllowRequests:          types.BoolValue(resourceResp.JSON200.Result.AllowRequests),
		AllowAsGrantMethod:     types.BoolValue(resourceResp.JSON200.Result.AllowAsGrantMethod),
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
