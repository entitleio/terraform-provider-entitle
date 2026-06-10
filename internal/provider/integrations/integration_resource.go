package integrations

import (
	"context"
	"maps"
	"strings"

	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"github.com/entitleio/terraform-provider-entitle/internal/validators"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/entitleio/terraform-provider-entitle/docs"
	"github.com/entitleio/terraform-provider-entitle/internal/client"
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
							validators.NewName(2, 50),
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

	newBase, appName, diags := ReadIntegration(ctx, r.client, data.BaseIntegrationResourceModel)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
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

	newBase, diags := UpdateIntegration(ctx, r.client, data.BaseIntegrationResourceModel, applicationName(data.Application.Name.ValueString()), parsedConnectionJson)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, IntegrationResourceModel{
		BaseIntegrationResourceModel: newBase,
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
