package permissions

//
//import (
//	"context"
//	"encoding/json"
//	"fmt"
//	"net/http"
//	"strings"
//
//	"github.com/google/uuid"
//	"github.com/hashicorp/terraform-plugin-framework/diag"
//	"github.com/hashicorp/terraform-plugin-framework/path"
//	"github.com/hashicorp/terraform-plugin-framework/resource"
//	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
//	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
//	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
//	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
//	"github.com/hashicorp/terraform-plugin-framework/types"
//	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
//	"github.com/hashicorp/terraform-plugin-log/tflog"
//
//	"github.com/entitleio/terraform-provider-entitle/internal/client"
//	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
//	"github.com/entitleio/terraform-provider-entitle/internal/validators"
//)
//
//// Ensure provider defined types fully satisfy framework interfaces.
//var _ resource.Resource = &Permission{}
//var _ resource.ResourceWithImportState = &Permission{}
//
//func NewPermission() resource.Resource {
//	return &Permission{}
//}
//
//// Permission defines the resource implementation.
//type Permission struct {
//	client *client.ClientWithResponses
//}
//
//// PermissionModel describes the resource data model.
//type PermissionModel struct {
//	ID    types.String        `tfsdk:"id"`
//	Actor *utils.IdEmailModel `tfsdk:"actor"`
//	Role  utils.Role          `tfsdk:"role"`
//	Path  types.String        `tfsdk:"path"`
//	Types types.Set           `tfsdk:"types"`
//}
//
//func (r *Permission) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
//	resp.TypeName = req.ProviderTypeName + "_permission"
//}
//
//func (r *Permission) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
//	resp.Schema = schema.Schema{
//		MarkdownDescription: "A specific instance or integration with an \"Application\". Integration includes the " +
//			"configuration needed to connect Entitle including credentials, as well as all the users permissions information. [Read more about integrations](https://docs.beyondtrust.com/entitle/docs/integrations-resources-roles).",
//		Description: "A specific instance or integration with an \"Application\". Integration includes the " +
//			"configuration needed to connect Entitle including credentials, as well as all the users permissions information. [Read more about integrations](https://docs.beyondtrust.com/entitle/docs/integrations-resources-roles).",
//		Attributes: map[string]schema.Attribute{
//			"id": schema.StringAttribute{
//				Computed:            true,
//				MarkdownDescription: "Entitle Permission identifier in uuid format",
//				Description:         "Entitle Permission identifier in uuid format",
//				PlanModifiers: []planmodifier.String{
//					stringplanmodifier.UseStateForUnknown(),
//				},
//			},
//			"actor": schema.SingleNestedAttribute{
//				Attributes: map[string]schema.Attribute{
//					"id": schema.StringAttribute{
//						Required:            false,
//						Optional:            true,
//						Description:         "the owner's id",
//						MarkdownDescription: "the owner's id",
//					},
//					"email": schema.StringAttribute{
//						Computed:            true,
//						Description:         "the owner's email",
//						MarkdownDescription: "the owner's email",
//					},
//				},
//				Required: false,
//				Optional: true,
//				Description: "Define the owner of the integration, which will be used for administrative " +
//					"purposes and approval workflows.",
//				MarkdownDescription: "Define the owner of the integration, which will be used for administrative " +
//					"purposes and approval workflows.",
//			},
//			"role": schema.SingleNestedAttribute{
//				Attributes: map[string]schema.Attribute{
//					"name": schema.StringAttribute{
//						Required:            false,
//						Optional:            true,
//						Description:         "The application's name",
//						MarkdownDescription: "The application's name",
//					},
//				},
//				Required: false,
//				Optional: true,
//				Description: "The application the integration connects to must be chosen from the list " +
//					"of supported applications.",
//				MarkdownDescription: "The application the integration connects to must be chosen from the list " +
//					"of supported applications.",
//			},
//			"path": schema.StringAttribute{
//				Required:            true,
//				Optional:            false,
//				Description:         "go to https://app.entitle.io/integrations and provide the latest schema.",
//				MarkdownDescription: "go to https://app.entitle.io/integrations and provide the latest schema.",
//			},
//			"types": schema.SetAttribute{
//				ElementType: types.NumberType,
//				Optional:    true,
//				Description: "As the admin, you can set different durations for the integration, " +
//					"compared to the workflow linked to it.",
//				MarkdownDescription: "As the admin, you can set different durations for the integration, " +
//					"compared to the workflow linked to it.",
//			},
//		},
//	}
//}
//
//func (r *Permission) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
//	// Prevent panic if the provider has not been configured.
//	if req.ProviderData == nil {
//		return
//	}
//
//	c, ok := req.ProviderData.(*client.ClientWithResponses)
//
//	if !ok {
//		resp.Diagnostics.AddError(
//			"Unexpected Resource Configure Type",
//			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
//		)
//
//		return
//	}
//
//	r.client = c
//}
//
//// Create this function is responsible for creating a new resource of type Entitle Permission.
////
//// It reads the Terraform plan data provided in req.Plan and maps it to the PermissionModel.
//// And sends a request to the Entitle API to create the resource using API requests.
//// If the creation is successful, it saves the resource's data into Terraform state.
//func (r *Permission) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
//	var err error
//	var plan PermissionModel
//
//	// Read Terraform plan data into the model
//	diags := req.Plan.Get(ctx, &plan)
//	resp.Diagnostics.Append(diags...)
//	if resp.Diagnostics.HasError() {
//		return
//	}
//
//	// TODO return error
//}
//
//// Read this function is used to read an existing resource of type Entitle Permission.
////
//// It retrieves the resource's data from the provider API requests.
//// The retrieved data is then mapped to the PermissionModel,
//// and the data is saved to Terraform state.
//func (r *Permission) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
//	var data PermissionModel
//
//	// Read Terraform prior state data into the model
//	diags := req.State.Get(ctx, &data)
//	resp.Diagnostics.Append(diags...)
//	if resp.Diagnostics.HasError() {
//		return
//	}
//
//	uid, err := uuid.Parse(data.ID.String())
//	if err != nil {
//		resp.Diagnostics.AddError(
//			"Client Error",
//			fmt.Sprintf("failed to parse the permission id (%s) to UUID, got error: %s", data.ID.String(), err),
//		)
//		return
//	}
//
//	apiResp, err := r.client.PermissionsIndexWithResponse(ctx, nil)
//	if err != nil {
//		resp.Diagnostics.AddError(
//			"Client Error",
//			fmt.Sprintf("Unable to get the permission by the id (%s), got error: %s", uid.String(), err),
//		)
//		return
//	}
//
//	//TODO:
//	if apiResp.HTTPResponse.StatusCode != 200 {
//		errBody, _ := utils.GetErrorBody(integrationResp.Body)
//		if integrationResp.HTTPResponse.StatusCode == http.StatusUnauthorized ||
//			(integrationResp.HTTPResponse.StatusCode == http.StatusBadRequest && strings.Contains(errBody.GetMessage(), "is not a valid uuid")) {
//			resp.Diagnostics.AddError(
//				"Client Error",
//				"unauthorized token, update the entitle token and retry please",
//			)
//			return
//		}
//
//		resp.Diagnostics.AddError(
//			"Client Error",
//			fmt.Sprintf(
//				"failed to get the integration by the id (%s), status code: %d%s",
//				uid.String(),
//				integrationResp.HTTPResponse.StatusCode,
//				errBody.GetMessage(),
//			),
//		)
//		return
//	}
//
//	if len(apiResp.JSON200.Result[0]) < 1 {
//		//TODO: error
//	}
//
//	data, diags = convertFullPermissionResultResponseSchemaToModel(
//		ctx,
//		&apiResp.JSON200.Result[0],
//	)
//	resp.Diagnostics.Append(diags...)
//	if resp.Diagnostics.HasError() {
//		return
//	}
//
//	// Save updated data into Terraform state
//	diags = resp.State.Set(ctx, &data)
//	resp.Diagnostics.Append(diags...)
//	if resp.Diagnostics.HasError() {
//		return
//	}
//}
//
//// Update this function handles updates to an existing resource of type Entitle Permission.
////
//// It reads the updated Terraform plan data provided in req.Plan and maps it to the PermissionModel.
//// And sends a request to the Entitle API to update the resource using API requests.
//// If the update is successful, it saves the updated resource data into Terraform state.
//func (r *Permission) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
//	var data PermissionModel
//
//	// Read Terraform plan data into the model
//	diags := req.Plan.Get(ctx, &data)
//	resp.Diagnostics.Append(diags...)
//	if resp.Diagnostics.HasError() {
//		return
//	}
//
//	//TODO: not allowed
//}
//
//// Delete this function is responsible for deleting an existing resource of type
////
//// It reads the resource's data from Terraform state, extracts the unique identifier,
//// and sends a request to delete the resource using API requests.
//// If the deletion is successful, it removes the resource from Terraform state.
//func (r *Permission) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
//	var data PermissionModel
//
//	// Read Terraform prior state data into the model
//	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
//
//	if resp.Diagnostics.HasError() {
//		return
//	}
//
//	parsedUUID, err := uuid.Parse(data.ID.String())
//	if err != nil {
//		return
//	}
//
//	httpResp, err := r.client.IntegrationsDestroyWithResponse(ctx, parsedUUID)
//	if err != nil {
//		resp.Diagnostics.AddError(
//			"Client Error",
//			fmt.Sprintf("Unable to delete integrations, id: (%s), got error: %v", data.ID.String(), err),
//		)
//		return
//	}
//
//	if httpResp.HTTPResponse.StatusCode != 200 {
//		errBody, _ := utils.GetErrorBody(httpResp.Body)
//		if httpResp.HTTPResponse.StatusCode == http.StatusUnauthorized ||
//			(httpResp.HTTPResponse.StatusCode == http.StatusBadRequest && strings.Contains(errBody.GetMessage(), "is not a valid uuid")) {
//			resp.Diagnostics.AddError(
//				"Client Error",
//				"unauthorized token, update the entitle token and retry please",
//			)
//			return
//		}
//
//		if errBody.ID == "resource.notFound" {
//			return
//		}
//
//		resp.Diagnostics.AddError(
//			"Client Error",
//			fmt.Sprintf(
//				"Unable to delete integrations, id: (%s), status code: %d%s",
//				data.ID.String(),
//				httpResp.HTTPResponse.StatusCode,
//				errBody.GetMessage(),
//			),
//		)
//		return
//	}
//}
//
//// ImportState this function is used to import an existing resource's state into Terraform.
////
//// It extracts the resource's identifier from the import request and sets
//// it in Terraform state using resource.ImportStatePassthroughID.
//func (r *Permission) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
//	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
//}
//
//// convertFullPermissionResultResponseSchemaToModel is a utility function used to convert the API response data
//// (of type client.IntegrationResultSchema) to a Terraform resource model (of type PermissionModel).
////
//// It extracts and transforms data from the API response into a format that can be stored in Terraform state.
//// It returns the converted model and any diagnostic information if there are errors during the conversion.
//func convertFullPermissionResultResponseSchemaToModel(
//	ctx context.Context,
//	data *client.PermissionSchema,
//) (PermissionModel, diag.Diagnostics) {
//	var diags diag.Diagnostics
//
//	// Check if the API response data is nil
//	if data == nil {
//		diags.AddError(
//			"No data",
//			"Failed: the given schema data is nil",
//		)
//
//		return PermissionModel{}, diags
//	}
//
//	// Extract and convert types from the API response
//	allowedTypes, advDiags := GetTypesFromResponse(data.Types)
//	if advDiags.HasError() {
//		diags.Append(advDiags...)
//		return PermissionModel{}, diags
//	}
//
//	// Create the Terraform permission model using the extracted data
//	return PermissionModel{
//		ID: utils.TrimmedStringValue(data.PermissionId.String()),
//		Actor: &utils.IdEmailModel{
//			Id:    utils.TrimmedStringValue(data.Account.Id.String()),
//			Email: utils.TrimmedStringValue(data.Account.Email),
//		},
//		Role: utils.Role{
//			ID:       utils.TrimmedStringValue(data.Role.Id.String()),
//			Name:     utils.TrimmedStringValue(data.Role.Name),
//			Resource: types.Object{}, //TODO:
//		},
//		Path:  utils.TrimmedStringValue(string(data.Path)),
//		Types: allowedTypes,
//	}, diags
//}
