package integrations

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	AllowedDurations                     types.List               `tfsdk:"allowed_durations"`
	AllowChangingAccountPermissions      types.Bool               `tfsdk:"allow_changing_account_permissions"`
	AllowCreatingAccounts                types.Bool               `tfsdk:"allow_creating_accounts"`
	Readonly                             types.Bool               `tfsdk:"readonly"`
	AllowRequests                        types.Bool               `tfsdk:"allow_requests"`
	AllowRequestsByDefault               types.Bool               `tfsdk:"allow_requests_by_default"`
	AllowAsGrantMethod                   types.Bool               `tfsdk:"allow_as_grant_method"`
	AllowAsGrantMethodByDefault          types.Bool               `tfsdk:"allow_as_grant_method_by_default"`
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
		MarkdownDescription: "Entitle Integration Description",
		Description:         "Entitle Integration Description",
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
			"allowed_durations": schema.ListAttribute{
				ElementType:         types.NumberType,
				Computed:            true,
				Description:         "allowedDurations",
				MarkdownDescription: "allowedDurations",
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
			"application": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Computed:            true,
						Description:         "name",
						MarkdownDescription: "name",
					},
				},
				Computed:            true,
				Description:         "application",
				MarkdownDescription: "application",
			},
			"allow_changing_account_permissions": schema.BoolAttribute{
				Computed:            true,
				Description:         "allowChangingAccountPermissions (default: true)",
				MarkdownDescription: "allowChangingAccountPermissions (default: true)",
			},
			"allow_creating_accounts": schema.BoolAttribute{
				Computed:            true,
				Description:         "allowCreatingAccounts (default: true)",
				MarkdownDescription: "allowCreatingAccounts (default: true)",
			},
			"readonly": schema.BoolAttribute{
				Computed:            true,
				Description:         "readonly (default: true)",
				MarkdownDescription: "readonly (default: true)",
			},
			"allow_requests": schema.BoolAttribute{
				Computed:            true,
				Description:         "allowRequests (default: true)",
				MarkdownDescription: "allowRequests (default: true)",
			},
			"allow_requests_by_default": schema.BoolAttribute{
				Computed:            true,
				Description:         "allowRequestsByDefault (default: true)",
				MarkdownDescription: "allowRequestsByDefault (default: true)",
			},
			"allow_as_grant_method": schema.BoolAttribute{
				Computed:            true,
				Description:         "allowAsGrantMethod (default: false)",
				MarkdownDescription: "allowAsGrantMethod (default: false)",
			},
			"allow_as_grant_method_by_default": schema.BoolAttribute{
				Computed:            true,
				Description:         "allowAsGrantMethodByDefault (default: false)",
				MarkdownDescription: "allowAsGrantMethodByDefault (default: false)",
			},
			"auto_assign_recommended_maintainers": schema.BoolAttribute{
				Computed:            true,
				Description:         "autoAssignRecommendedMaintainers (default: true)",
				MarkdownDescription: "autoAssignRecommendedMaintainers (default: true)",
			},
			"auto_assign_recommended_owners": schema.BoolAttribute{
				Computed:            true,
				Description:         "autoAssignRecommendedOwners (default: true)",
				MarkdownDescription: "autoAssignRecommendedOwners (default: true)",
			},
			"notify_about_external_permission_changes": schema.BoolAttribute{
				Computed:            true,
				Description:         "notifyAboutExternalPermissionChanges (default: true)",
				MarkdownDescription: "notifyAboutExternalPermissionChanges (default: true)",
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

	allowedDurationsValues := make([]attr.Value, len(integrationResp.JSON200.Result.AllowedDurations))
	for i, durations := range integrationResp.JSON200.Result.AllowedDurations {
		allowedDurationsValues[i] = types.NumberValue(big.NewFloat(float64(durations)))
	}

	allowedDurations, diags := types.ListValue(types.NumberType, allowedDurationsValues)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
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
		AllowedDurations:                     allowedDurations,
		AllowChangingAccountPermissions:      types.BoolValue(integrationResp.JSON200.Result.AllowChangingAccountPermissions),
		AllowCreatingAccounts:                types.BoolValue(integrationResp.JSON200.Result.AllowCreatingAccounts),
		Readonly:                             types.BoolValue(integrationResp.JSON200.Result.Readonly),
		AllowRequests:                        types.BoolValue(integrationResp.JSON200.Result.AllowRequests),
		AllowRequestsByDefault:               types.BoolValue(integrationResp.JSON200.Result.AllowRequestsByDefault),
		AllowAsGrantMethod:                   types.BoolValue(integrationResp.JSON200.Result.AllowAsGrantMethod),
		AllowAsGrantMethodByDefault:          types.BoolValue(integrationResp.JSON200.Result.AllowAsGrantMethodByDefault),
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
