package integrations

import (
	"context"
	"maps"
	"strings"

	"github.com/entitleio/terraform-provider-entitle/docs"
	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"github.com/entitleio/terraform-provider-entitle/internal/validators"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &IntegrationResource{}
var _ resource.ResourceWithImportState = &IntegrationResource{}

func NewIntegrationResource() resource.Resource {
	return &IntegrationResource{}
}

// IntegrationResource defines the resource implementation.
type IntegrationResource struct {
	client *client.ClientWithResponses
}

// IntegrationResourceModel describes the resource data model.
type IntegrationResourceModel struct {
	BaseIntegrationResourceModel
	ConnectionJson types.String     `tfsdk:"connection_json"`
	Application    *utils.NameModel `tfsdk:"application"`
}

func (r *IntegrationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration"
}

func (r *IntegrationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: docs.IntegrationResourceMarkdownDescription,
		Attributes: func() map[string]schema.Attribute {
			m := maps.Clone(BaseIntegrationResourceAttributes)

			m["connection_json"] = schema.StringAttribute{
				Required:            true,
				Description:         "You can get it on [this page](https://docs.beyondtrust.com/entitle/docs/integrations) or using [web ui create form](https://app.entitle.io/integrations/create).",
				MarkdownDescription: "You can get it on [this page](https://docs.beyondtrust.com/entitle/docs/integrations) or using [web ui create form](https://app.entitle.io/integrations/create).",
			}

			m["application"] = schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required:            true,
						Description:         "The application's name (lowercase). Could be found using entitle_applications. More detailed info about integrations available on [this page](https://docs.beyondtrust.com/entitle/docs/integrations).",
						MarkdownDescription: "The application's name (lowercase). Could be found using entitle_applications. More detailed info about integrations available on [this page](https://docs.beyondtrust.com/entitle/docs/integrations).",
						Validators: []validator.String{
							stringvalidator.LengthBetween(2, 50),
							validators.Lowercase{},
						},
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
				},
				Required: true,
				Description: "The application the integration connects to must be chosen from the list " +
					"of supported applications.",
				MarkdownDescription: "The application the integration connects to must be chosen from the list " +
					"of supported applications.",
			}

			return m
		}(),
	}
}

func (r *IntegrationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureIntegrationResource(req.ProviderData, &r.client, &resp.Diagnostics)
}

// Create this function is responsible for creating a new resource of type Entitle Integration.
//
// Its reads the Terraform plan data provided in req.Plan and maps it to the IntegrationResourceModel.
// And sends a request to the Entitle API to create the resource using API requests.
// If the creation is successful, it saves the resource's data into Terraform state.
func (r *IntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan IntegrationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parsedConnectionJson, diags := ParseConnectionJson(plan.ConnectionJson.ValueString())
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	newBase, diags := CreateIntegration(ctx, r.client, plan.BaseIntegrationResourceModel, applicationName(plan.Application.Name.ValueString()), parsedConnectionJson)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, IntegrationResourceModel{
		BaseIntegrationResourceModel: newBase,
		ConnectionJson:               plan.ConnectionJson,
		Application: &utils.NameModel{
			Name: utils.TrimmedStringValue(strings.ToLower(plan.Application.Name.ValueString())),
		},
	})...)
}

// Read this function is used to read an existing resource of type Entitle Integration.
//
// It retrieves the resource's data from the provider API requests.
// The retrieved data is then mapped to the IntegrationResourceModel,
// and the data is saved to Terraform state.
func (r *IntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data IntegrationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	newBase, appName, ok := ReadIntegration(ctx, r.client, data.BaseIntegrationResourceModel, resp)
	if !ok {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, IntegrationResourceModel{
		BaseIntegrationResourceModel: newBase,
		ConnectionJson:               data.ConnectionJson,
		Application: &utils.NameModel{
			Name: utils.TrimmedStringValue(strings.ToLower(appName)),
		},
	})...)
}

// Update this function handles updates to an existing resource of type Entitle Integration.
//
// It reads the updated Terraform plan data provided in req.Plan and maps it to the IntegrationResourceModel.
// And sends a request to the Entitle API to update the resource using API requests.
// If the update is successful, it saves the updated resource data into Terraform state.
func (r *IntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data IntegrationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parsedConnectionJson, diags := ParseConnectionJson(data.ConnectionJson.ValueString())
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	newBase := UpdateIntegration(ctx, r.client, data.BaseIntegrationResourceModel, applicationName(data.Application.Name.ValueString()), parsedConnectionJson, resp)
	if newBase == nil {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, IntegrationResourceModel{
		BaseIntegrationResourceModel: *newBase,
		ConnectionJson:               data.ConnectionJson,
		Application: &utils.NameModel{
			Name: utils.TrimmedStringValue(strings.ToLower(data.Application.Name.ValueString())),
		},
	})...)
}

// Delete this function is responsible for deleting an existing resource of type
//
// It reads the resource's data from Terraform state, extracts the unique identifier,
// and sends a request to delete the resource using API requests.
// If the deletion is successful, it removes the resource from Terraform state.
func (r *IntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data IntegrationResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	DeleteIntegration(ctx, r.client, data.BaseIntegrationResourceModel, resp)
}

// ImportState this function is used to import an existing resource's state into Terraform.
//
// It extracts the resource's identifier from the import request and sets
// it in Terraform state using resource.ImportStatePassthroughID.
func (r *IntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// convertFullIntegrationResultResponseSchemaToModel is a utility function used to convert the API response data
// (of type client.IntegrationResultSchema) to a Terraform resource model (of type IntegrationResourceModel).
//
// It extracts and transforms data from the API response into a format that can be stored in Terraform state.
// It returns the converted model and any diagnostic information if there are errors during the conversion.
func convertFullIntegrationResultResponseSchemaToModel(
	ctx context.Context,
	data *client.IntegrationResultSchema,
	connectionJson, agentTokenName string,
) (IntegrationResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Check if the API response data is nil
	if data == nil {
		diags.AddError(
			"No data",
			"Failed: the given schema data is nil",
		)

		return IntegrationResourceModel{}, diags
	}

	// Extract and convert allowed durations from the API response
	allowedDurationsValues, advDiags := utils.GetNumberSetFromAllowedDurations(data.AllowedDurations)
	if advDiags.HasError() {
		diags.Append(advDiags...)
		return IntegrationResourceModel{}, diags
	}

	maintainers := make([]*utils.MaintainerModel, 0, len(data.Maintainers))
	for _, item := range data.Maintainers {
		var body utils.MaintainerCommonResponseSchema

		dataBytes, err := item.MarshalJSON()
		if err != nil {
			diags.AddError(
				"No data",
				"Failed to marshal the maintainer data",
			)

			return IntegrationResourceModel{}, diags
		}

		err = json.Unmarshal(dataBytes, &body)
		if err != nil {
			diags.AddError(
				"No data",
				fmt.Sprintf("Failed to unmarshal the maintainer data (%s), error: %s", dataBytes, err.Error()),
			)

			return IntegrationResourceModel{}, diags
		}

		switch strings.ToLower(body.Type) {
		case utils.MaintainerTypeUser:
			responseSchema, err := item.AsMaintainerUserResponseSchema()
			if err != nil {
				diags.AddError(
					"No data",
					fmt.Sprintf("Failed to convert response schema to user response schema, error: %v", err),
				)

				return IntegrationResourceModel{}, diags
			}

			u := &utils.IdEmailModel{
				Id:    utils.TrimmedStringValue(responseSchema.User.Id.String()),
				Email: utils.GetNullableEmailStringValue(responseSchema.User.Email),
			}

			uObject, diagsValues := u.AsObjectValue(ctx)
			if diagsValues.HasError() {
				diags.Append(diagsValues...)
				return IntegrationResourceModel{}, diags
			}

			maintainerUser := &utils.MaintainerModel{
				Type:   utils.TrimmedStringValue(body.Type),
				Entity: uObject,
			}

			maintainers = append(maintainers, maintainerUser)
		case utils.MaintainerTypeGroup:
			responseSchema, err := item.AsMaintainerGroupResponseSchema()
			if err != nil {
				diags.AddError(
					"No data",
					fmt.Sprintf("Failed to convert response schema to group response schema, error: %v", err),
				)

				return IntegrationResourceModel{}, diags
			}

			g := &utils.IdEmailModel{
				Id:    utils.TrimmedStringValue(responseSchema.Group.Id.String()),
				Email: utils.GetNullableEmailStringValue(responseSchema.Group.Email),
			}

			gObject, diagsValues := g.AsObjectValue(ctx)
			if diagsValues.HasError() {
				diags.Append(diagsValues...)
				return IntegrationResourceModel{}, diags
			}

			maintainerGroup := &utils.MaintainerModel{
				Type:   utils.TrimmedStringValue(body.Type),
				Entity: gObject,
			}

			maintainers = append(maintainers, maintainerGroup)
		default:
			diags.AddError("Failed invalid type for maintainer", body.Type)
			return IntegrationResourceModel{}, diags
		}
	}

	var agentToken *utils.NameModel
	if len(agentTokenName) != 0 {
		agentToken = &utils.NameModel{
			Name: utils.TrimmedStringValue(agentTokenName),
		}
	}

	var prerequisitePermissions []utils.PrerequisitePermissionModel
	if data.PrerequisitePermissions != nil {
		for _, pp := range *data.PrerequisitePermissions {
			for _, item := range pp {
				v, err := item.AsPrerequisiteRolePermissionResponseSchema()
				if err != nil {
					diags.AddError(
						"No data",
						fmt.Sprintf("Failed to unmarshal the prerequisite permissions data, err: %s", err.Error()),
					)
					return IntegrationResourceModel{}, diags
				}

				roleModel, diagsGetRoles := utils.GetRole(ctx, v.Role.Id.String(), v.Role.Name, v.Role.Resource)
				if diagsGetRoles.HasError() {
					diags.Append(diagsGetRoles...)
					return IntegrationResourceModel{}, diags
				}

				prerequisitePermissions = append(prerequisitePermissions,
					utils.PrerequisitePermissionModel{
						Default: types.BoolValue(v.Default),
						Role:    roleModel,
					},
				)
			}
		}
	}

	if len(maintainers) == 0 {
		maintainers = nil
	}

	// Create the Terraform resource model using the extracted data
	return IntegrationResourceModel{
		ID:                                   utils.TrimmedStringValue(data.Id.String()),
		Name:                                 utils.TrimmedStringValue(data.Name),
		AllowedDurations:                     allowedDurationsValues,
		AllowChangingAccountPermissions:      types.BoolValue(data.AllowChangingAccountPermissions),
		AllowCreatingAccounts:                types.BoolValue(data.AllowCreatingAccounts),
		Readonly:                             types.BoolValue(data.Readonly),
		Requestable:                          types.BoolValue(data.Requestable),
		RequestableByDefault:                 types.BoolValue(data.RequestableByDefault),
		AutoAssignRecommendedMaintainers:     types.BoolValue(data.AutoAssignRecommendedMaintainers),
		AutoAssignRecommendedOwners:          types.BoolValue(data.AutoAssignRecommendedOwners),
		NotifyAboutExternalPermissionChanges: types.BoolValue(data.NotifyAboutExternalPermissionChanges),
		Owner: &utils.IdEmailModel{
			Id:    utils.TrimmedStringValue(data.Owner.Id.String()),
			Email: utils.GetNullableEmailStringValue(data.Owner.Email),
		},
		Application: &utils.NameModel{
			Name: utils.TrimmedStringValue(strings.ToLower(data.Application.Name)),
		},
		AgentToken: agentToken,
		Workflow: &utils.IdNameModel{
			ID:   utils.TrimmedStringValue(data.Workflow.Id.String()),
			Name: utils.TrimmedStringValue(data.Workflow.Name),
		},
		Maintainers:             maintainers,
		ConnectionJson:          utils.TrimmedStringValue(connectionJson),
		PrerequisitePermissions: prerequisitePermissions,
	}, diags
}
