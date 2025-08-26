package permissions

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &Permission{}
var _ resource.ResourceWithImportState = &Permission{}

func NewPermissionResource() resource.Resource {
	return &Permission{}
}

// Permission defines the resource implementation.
type Permission struct {
	client *client.ClientWithResponses
}

// PermissionModel describes the resource data model.
type PermissionModel struct {
	ID    types.String        `tfsdk:"id"`
	Actor *utils.IdEmailModel `tfsdk:"actor"`
	Role  *utils.Role         `tfsdk:"role"`
	Path  types.String        `tfsdk:"path"`
	Types types.Set           `tfsdk:"types"`
}

func (r *Permission) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_permission"
}

func (r *Permission) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A specific instance or integration with an \"Application\". Integration includes the " +
			"configuration needed to connect Entitle including credentials, as well as all the users permissions information. [Read more about integrations](https://docs.beyondtrust.com/entitle/docs/integrations-resources-roles).",
		Description: "A specific instance or integration with an \"Application\". Integration includes the " +
			"configuration needed to connect Entitle including credentials, as well as all the users permissions information. [Read more about integrations](https://docs.beyondtrust.com/entitle/docs/integrations-resources-roles).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Entitle Permission identifier in uuid format",
				Description:         "Entitle Permission identifier in uuid format",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"actor": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Required:            false,
						Optional:            true,
						Description:         "the owner's id",
						MarkdownDescription: "the owner's id",
					},
					"email": schema.StringAttribute{
						Computed:            true,
						Description:         "the owner's email",
						MarkdownDescription: "the owner's email",
					},
				},
				Required: false,
				Optional: true,
				Description: "Define the owner of the integration, which will be used for administrative " +
					"purposes and approval workflows.",
				MarkdownDescription: "Define the owner of the integration, which will be used for administrative " +
					"purposes and approval workflows.",
			},
			"role": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Required:            true,
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
				Required: false,
				Optional: true,
				Description: "The application the integration connects to must be chosen from the list " +
					"of supported applications.",
				MarkdownDescription: "The application the integration connects to must be chosen from the list " +
					"of supported applications.",
			},
			"path": schema.StringAttribute{
				Computed:            true,
				Description:         "go to https://app.entitle.io/integrations and provide the latest schema.",
				MarkdownDescription: "go to https://app.entitle.io/integrations and provide the latest schema.",
			},
			"types": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "As the admin, you can set different durations for the integration, " +
					"compared to the workflow linked to it.",
				MarkdownDescription: "As the admin, you can set different durations for the integration, " +
					"compared to the workflow linked to it.",
			},
		},
	}
}

func (r *Permission) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.ClientWithResponses)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = c
}

// Create this function is responsible for creating a new resource of type Entitle Permission.
//
// It reads the Terraform plan data provided in req.Plan and maps it to the PermissionModel.
// And sends a request to the Entitle API to create the resource using API requests.
// If the creation is successful, it saves the resource's data into Terraform state.
func (r *Permission) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PermissionModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.Create(ctx, req, resp)
	//resp.Diagnostics.AddError("Invalid Action", "Create is not supported")
}

// Read this function is used to read an existing resource of type Entitle Permission.
//
// It retrieves the resource's data from the provider API requests.
// The retrieved data is then mapped to the PermissionModel,
// and the data is saved to Terraform state.
func (r *Permission) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PermissionModel

	// Read Terraform prior state data into the model
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	uid, err := uuid.Parse(data.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("failed to parse the permission id (%s) to UUID, got error: %s", data.ID.String(), err),
		)
		return
	}

	apiResp, err := r.client.PermissionsIndexWithResponse(ctx, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to get the permission by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(apiResp.HTTPResponse.StatusCode, apiResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"Failed to get the Permission by the id (%s), status code: %d, %s",
				uid.String(),
				apiResp.HTTPResponse.StatusCode,
				err.Error(),
			),
		)
		return
	}

	for _, permission := range apiResp.JSON200.Result {
		if permission.PermissionId.String() == uid.String() {
			data, diags = convertFullPermissionResultResponseSchemaToModel(
				ctx,
				&permission,
			)

			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	if data.ID.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to get the permission by the id (%s), got error: %s", uid.String(), "not found"),
		)
	}

	// Save updated data into Terraform state
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update this function handles updates to an existing resource of type Entitle Permission.
//
// It reads the updated Terraform plan data provided in req.Plan and maps it to the PermissionModel.
// And sends a request to the Entitle API to update the resource using API requests.
// If the update is successful, it saves the updated resource data into Terraform state.
func (r *Permission) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PermissionModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.AddError("Invalid Action", "Update is not supported")
}

// Delete this function is responsible for deleting an existing resource of type
//
// It reads the resource's data from Terraform state, extracts the unique identifier,
// and sends a request to delete the resource using API requests.
// If the deletion is successful, it removes the resource from Terraform state.
func (r *Permission) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PermissionModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	parsedUUID, err := uuid.Parse(data.ID.String())
	if err != nil {
		return
	}

	httpResp, err := r.client.PermissionsRevokeWithResponse(ctx, parsedUUID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to delete Permission, id: (%s), got error: %v", data.ID.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(httpResp.HTTPResponse.StatusCode, httpResp.Body, utils.WithIgnoreNotFound(), utils.WithIgnorePending())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"Failed to delete the Permission by the id (%s), status code: %d, %s",
				data.ID.String(),
				httpResp.HTTPResponse.StatusCode,
				err.Error(),
			),
		)
		return
	}
}

// ImportState this function is used to import an existing resource's state into Terraform.
//
// It extracts the resource's identifier from the import request and sets
// it in Terraform state using resource.ImportStatePassthroughID.
func (r *Permission) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// convertFullPermissionResultResponseSchemaToModel is a utility function used to convert the API response data
// (of type client.IntegrationResultSchema) to a Terraform resource model (of type PermissionModel).
//
// It extracts and transforms data from the API response into a format that can be stored in Terraform state.
// It returns the converted model and any diagnostic information if there are errors during the conversion.
func convertFullPermissionResultResponseSchemaToModel(
	ctx context.Context,
	data *client.PermissionSchema,
) (PermissionModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Check if the API response data is nil
	if data == nil {
		diags.AddError(
			"No data",
			"Failed: the given schema data is nil",
		)

		return PermissionModel{}, diags
	}

	// Extract and convert types from the API response
	allowedTypes, advDiags := GetTypesFromResponse(data.Types)
	if advDiags.HasError() {
		diags.Append(advDiags...)
		return PermissionModel{}, diags
	}

	resource, rDiags := utils.RoleResource{
		Id:   utils.TrimmedStringValue(data.Role.Resource.Id.String()),
		Name: utils.TrimmedStringValue(data.Role.Resource.Name),
		Integration: utils.RoleResourceIntegration{
			Id:   utils.TrimmedStringValue(data.Role.Resource.Integration.Id.String()),
			Name: utils.TrimmedStringValue(data.Role.Resource.Integration.Name),
			Application: utils.NameModel{
				Name: utils.TrimmedStringValue(data.Role.Resource.Integration.Application.Name),
			},
		},
	}.AsObjectValue(ctx)
	if rDiags.HasError() {
		diags.Append(rDiags...)
		return PermissionModel{}, diags
	}

	// Create the Terraform permission model using the extracted data
	return PermissionModel{
		ID: utils.TrimmedStringValue(data.PermissionId.String()),
		Actor: &utils.IdEmailModel{
			Id:    utils.TrimmedStringValue(data.Account.Id.String()),
			Email: utils.TrimmedStringValue(data.Account.Email),
		},
		Role: &utils.Role{
			ID:       utils.TrimmedStringValue(data.Role.Id.String()),
			Name:     utils.TrimmedStringValue(data.Role.Name),
			Resource: resource,
		},
		Path:  utils.TrimmedStringValue(string(data.Path)),
		Types: allowedTypes,
	}, diags
}
