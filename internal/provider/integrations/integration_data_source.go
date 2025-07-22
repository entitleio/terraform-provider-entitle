package integrations

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
var _ datasource.DataSource = &IntegrationDataSource{}

// IntegrationDataSource defines the data source implementation.
type IntegrationDataSource struct {
	client *client.ClientWithResponses
}

// NewIntegrationDataSource creates a new instance of IntegrationDataSource.
func NewIntegrationDataSource() datasource.DataSource {
	return &IntegrationDataSource{}
}

// IntegrationDataSourceModel describes the data source data model.
type IntegrationDataSourceModel struct {
	Id                                   types.String             `tfsdk:"id"`
	Name                                 types.String             `tfsdk:"name"`
	AllowedDurations                     types.Set                `tfsdk:"allowed_durations"`
	AllowChangingAccountPermissions      types.Bool               `tfsdk:"allow_changing_account_permissions"`
	AllowCreatingAccounts                types.Bool               `tfsdk:"allow_creating_accounts"`
	Readonly                             types.Bool               `tfsdk:"readonly"`
	Requestable                          types.Bool               `tfsdk:"requestable"`
	RequestableByDefault                 types.Bool               `tfsdk:"requestable_by_default"`
	AutoAssignRecommendedMaintainers     types.Bool               `tfsdk:"auto_assign_recommended_maintainers"`
	AutoAssignRecommendedOwners          types.Bool               `tfsdk:"auto_assign_recommended_owners"`
	NotifyAboutExternalPermissionChanges types.Bool               `tfsdk:"notify_about_external_permission_changes"`
	Application                          *utils.NameModel         `tfsdk:"application"`
	Workflow                             *utils.IdNameModel       `tfsdk:"workflow"`
	Maintainers                          []*utils.MaintainerModel `tfsdk:"maintainers"`
}

// Metadata sets the metadata for the data source.
func (d *IntegrationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration"
}

// Schema defines the data source schema.
func (d *IntegrationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Entitle Integration represents a connection to an external system that can be managed through Entitle. It includes configuration for permissions, maintainers, workflows, and access policies. [Read more about integrations](https://docs.beyondtrust.com/entitle/docs/integrations-resources-roles).",
		Description:         "Entitle Integration represents a connection to an external system that can be managed through Entitle. It includes configuration for permissions, maintainers, workflows, and access policies. [Read more about integrations](https://docs.beyondtrust.com/entitle/docs/integrations-resources-roles).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Entitle Integration identifier in uuid format",
				Description:         "Entitle Integration identifier in uuid format",
				Validators: []validator.String{
					validators.UUID{},
				},
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Entitle Integration name",
				Description:         "Entitle Integration name",
			},
			"allowed_durations": schema.SetAttribute{
				ElementType:         types.NumberType,
				Computed:            true,
				Description:         "List of allowed durations (in seconds) for this integration",
				MarkdownDescription: "List of allowed durations (in seconds) for this integration",
			},
			"maintainers": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Computed:            true,
							Description:         "The maintainer type (e.g., user or group)",
							MarkdownDescription: "The maintainer type (e.g., user or group)",
						},
						"user": schema.SingleNestedAttribute{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Computed:            true,
									Description:         "id",
									MarkdownDescription: "User's unique identifier",
								},
								"email": schema.StringAttribute{
									Computed:            true,
									Description:         "User's email address",
									MarkdownDescription: "User's email address",
								},
							},
							Computed:            true,
							Description:         "The user maintainer details",
							MarkdownDescription: "The user maintainer details",
						},
						"group": schema.SingleNestedAttribute{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Computed:            true,
									Description:         "Group's unique identifier",
									MarkdownDescription: "Group's unique identifier",
								},
								"email": schema.StringAttribute{
									Computed:            true,
									Description:         "Group's email address",
									MarkdownDescription: "Group's email address",
								},
							},
							Computed:            true,
							Description:         "The group maintainer details",
							MarkdownDescription: "The group maintainer details",
						},
					},
				},
				Computed:            true,
				Description:         "List of maintainers responsible for this integration",
				MarkdownDescription: "List of maintainers responsible for this integration",
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
				Description:         "Application associated with this integration",
				MarkdownDescription: "Application associated with this integration",
			},
			"allow_changing_account_permissions": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether changing account permissions is allowed (default: true)",
				MarkdownDescription: "Whether changing account permissions is allowed (default: true)",
			},
			"allow_creating_accounts": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether creating new accounts is allowed (default: true)",
				MarkdownDescription: "Whether creating new accounts is allowed (default: true)",
			},
			"readonly": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether the integration is read-only (default: true)",
				MarkdownDescription: "Whether the integration is read-only (default: true)",
			},
			"requestable": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether the integration is requestable (default: true)",
				MarkdownDescription: "Whether the integration is requestable (default: true)",
			},
			"requestable_by_default": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether the integration is requestable by default (default: true)",
				MarkdownDescription: "Whether the integration is requestable by default (default: true)",
			},
			"auto_assign_recommended_maintainers": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether recommended maintainers are auto-assigned (default: true)",
				MarkdownDescription: "Whether recommended maintainers are auto-assigned (default: true)",
			},
			"auto_assign_recommended_owners": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether recommended owners are auto-assigned (default: true)",
				MarkdownDescription: "Whether recommended owners are auto-assigned (default: true)",
			},
			"notify_about_external_permission_changes": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether to notify about external permission changes (default: true)",
				MarkdownDescription: "Whether to notify about external permission changes (default: true)",
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
				Description:         "Workflow associated with this integration",
				MarkdownDescription: "Workflow associated with this integration",
			},
		},
	}
}

// Configure configures the data source with the provided client.
func (d *IntegrationDataSource) Configure(
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
func (d *IntegrationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data IntegrationDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uid, err := uuid.Parse(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("failed to parse integration id to uuid format, got error: %s", err),
		)
		return
	}

	integrationResp, err := d.client.IntegrationsShowWithResponse(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to get the integration by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	if integrationResp.HTTPResponse.StatusCode != 200 {
		errBody, _ := utils.GetErrorBody(integrationResp.Body)
		if integrationResp.HTTPResponse.StatusCode == http.StatusUnauthorized ||
			(integrationResp.HTTPResponse.StatusCode == http.StatusBadRequest && strings.Contains(errBody.GetMessage(), "is not a valid uuid")) {
			resp.Diagnostics.AddError(
				"Client Error",
				"unauthorized token, update the entitle token and retry please",
			)
			return
		}

		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"failed to get the integration by the id (%s), status code: %d%s",
				uid.String(),
				integrationResp.HTTPResponse.StatusCode,
				errBody.GetMessage(),
			),
		)
		return
	}

	// Extract and convert allowed durations from the API response
	allowedDurationsValues, advDiags := utils.GetNumberSetFromAllowedDurations(integrationResp.JSON200.Result.AllowedDurations)
	if advDiags.HasError() {
		resp.Diagnostics.Append(advDiags...)
	}

	application := &utils.NameModel{
		Name: utils.TrimmedStringValue(integrationResp.JSON200.Result.Application.Name),
	}

	workflow := &utils.IdNameModel{
		ID:   utils.TrimmedStringValue(integrationResp.JSON200.Result.Workflow.Id.String()),
		Name: utils.TrimmedStringValue(integrationResp.JSON200.Result.Workflow.Name),
	}

	maintainers, diags := utils.GetMaintainers(ctx, integrationResp.JSON200.Result.Maintainers)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data = IntegrationDataSourceModel{
		Id:                                   utils.TrimmedStringValue(integrationResp.JSON200.Result.Id.String()),
		Name:                                 utils.TrimmedStringValue(integrationResp.JSON200.Result.Name),
		AllowedDurations:                     allowedDurationsValues,
		AllowChangingAccountPermissions:      types.BoolValue(integrationResp.JSON200.Result.AllowChangingAccountPermissions),
		AllowCreatingAccounts:                types.BoolValue(integrationResp.JSON200.Result.AllowCreatingAccounts),
		Readonly:                             types.BoolValue(integrationResp.JSON200.Result.Readonly),
		Requestable:                          types.BoolValue(integrationResp.JSON200.Result.Requestable),
		RequestableByDefault:                 types.BoolValue(integrationResp.JSON200.Result.RequestableByDefault),
		AutoAssignRecommendedMaintainers:     types.BoolValue(integrationResp.JSON200.Result.AutoAssignRecommendedMaintainers),
		AutoAssignRecommendedOwners:          types.BoolValue(integrationResp.JSON200.Result.AutoAssignRecommendedOwners),
		NotifyAboutExternalPermissionChanges: types.BoolValue(integrationResp.JSON200.Result.NotifyAboutExternalPermissionChanges),
		Application:                          application,
		Workflow:                             workflow,
		Maintainers:                          maintainers,
	}

	tflog.Trace(ctx, "read a entitle integration data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	diagsResult := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diagsResult...)
	if resp.Diagnostics.HasError() {
		return
	}
}
