package permissions

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
)

// Ensure the interface is satisfied
var _ resource.Resource = &PermissionResource{}
var _ resource.ResourceWithImportState = &PermissionResource{}

type PermissionResource struct {
	client *client.ClientWithResponses
}

// PermissionResourceModel defines the Terraform schema model
type PermissionResourceModel struct {
	ID    types.String `tfsdk:"id"`
	Actor types.Object `tfsdk:"actor"`
	Role  types.Object `tfsdk:"role"`
	Path  types.String `tfsdk:"path"`
	Types types.Set    `tfsdk:"types"`
}

func NewPermissionResource() resource.Resource {
	return &PermissionResource{}
}

func (r *PermissionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_permission"
}

func (r *PermissionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Entitle Permission.\n\n⚠️ This is an **import-only resource** — Terraform does not create permissions. " +
			"Existing permissions must be already created before being imported or managed.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The unique ID of the permission (UUID).",
			},
			"actor": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"id":    schema.StringAttribute{Computed: true},
					"email": schema.StringAttribute{Computed: true},
				},
			},
			"role": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"id":   schema.StringAttribute{Computed: true},
					"name": schema.StringAttribute{Computed: true},
					"resource": schema.SingleNestedAttribute{
						Computed: true,
						Attributes: map[string]schema.Attribute{
							"id":   schema.StringAttribute{Computed: true},
							"name": schema.StringAttribute{Computed: true},
							"integration": schema.SingleNestedAttribute{
								Computed: true,
								Attributes: map[string]schema.Attribute{
									"id":   schema.StringAttribute{Computed: true},
									"name": schema.StringAttribute{Computed: true},
									"application": schema.SingleNestedAttribute{
										Computed: true,
										Attributes: map[string]schema.Attribute{
											"name": schema.StringAttribute{Computed: true},
										},
									},
								},
							},
						},
					},
				},
			},
			"path": schema.StringAttribute{
				Computed: true,
			},
			"types": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *PermissionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ClientWithResponses, got: %T", req.ProviderData))
		return
	}
	r.client = c
}

// Create verifies the permission exists in the list and sets ID (import-only)
func (r *PermissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PermissionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ID.IsNull() || plan.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Missing ID", "You must provide an existing permission ID to manage.")
		return
	}

	perm, err := r.findPermissionByID(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Permission Not Found", err.Error())
		return
	}

	diags := mapPermissionToModel(ctx, &plan, perm)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read fetches the permission from the list and updates state
func (r *PermissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PermissionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	perm, err := r.findPermissionByID(ctx, state.ID.ValueString())
	if err != nil {
		// Permission no longer exists → remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	diags := mapPermissionToModel(ctx, &state, perm)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is not supported
func (r *PermissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddWarning(
		"Update Not Supported",
		"This resource is import-only. To change it, delete and re-import the correct permission.",
	)
	resp.Diagnostics.Append(req.State.Get(ctx, nil)...)
}

// Delete removes the permission via API
func (r *PermissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PermissionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	permissionID, err := uuid.Parse(state.ID.ValueString())
	tflog.Debug(ctx, "Deleting Entitle Permission", map[string]any{"id": permissionID})
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("failed to parse account id to uuid format, got error: %s", err),
		)
		return
	}
	apiResp, err := r.client.PermissionsRevokeWithResponse(ctx, permissionID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete permission %s: %s", permissionID, err))
		return
	}

	if err := utils.HTTPResponseToError(apiResp.StatusCode(), apiResp.Body, utils.WithIgnorePending()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to delete permission: %s", err))
		return
	}
}

// ImportState allows terraform import
func (r *PermissionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// --- Helper functions ---

// findPermissionByID searches the permission list for the given ID
func (r *PermissionResource) findPermissionByID(ctx context.Context, id string) (*client.PermissionSchema, error) {
	params := client.PermissionsIndexParams{
		PerPage: utils.Float32Pointer(10000000000000000), // no way to get all or by id, hence fetching the maximum limit
	}
	permissionsResp, err := r.client.PermissionsIndexWithResponse(ctx, &params)
	if err != nil {
		return nil, fmt.Errorf("unable to list permissions: %w", err)
	}
	if permissionsResp.JSON200 == nil || permissionsResp.JSON200.Result == nil {
		return nil, fmt.Errorf("no permissions found")
	}

	for _, perm := range permissionsResp.JSON200.Result {
		if perm.PermissionId.String() == id {
			return &perm, nil
		}
	}

	return nil, fmt.Errorf("permission %s not found", id)
}

// mapPermissionToModel maps API response into Terraform model
func mapPermissionToModel(ctx context.Context, model *PermissionResourceModel, perm *client.PermissionSchema) diag.Diagnostics {
	typesSet, diags := GetTypesFromResponse(perm.Types)
	if diags.HasError() {
		return diags
	}

	model.ID = utils.TrimmedStringValue(perm.PermissionId.String())
	actor := &utils.IdEmailModel{
		Id:    utils.TrimmedStringValue(perm.Account.Id.String()),
		Email: utils.TrimmedStringValue(perm.Account.Email),
	}
	model.Actor, diags = actor.AsObjectValue(ctx)
	if diags.HasError() {
		return diags
	}

	resource, diags := utils.RoleResource{
		Id:   utils.TrimmedStringValue(perm.Role.Resource.Id.String()),
		Name: utils.TrimmedStringValue(perm.Role.Resource.Name),
		Integration: utils.RoleResourceIntegration{
			Id:   utils.TrimmedStringValue(perm.Role.Resource.Integration.Id.String()),
			Name: utils.TrimmedStringValue(perm.Role.Resource.Integration.Name),
			Application: utils.NameModel{
				Name: utils.TrimmedStringValue(perm.Role.Resource.Integration.Application.Name),
			},
		},
	}.AsObjectValue(ctx)
	if diags.HasError() {
		return diags
	}

	role := &utils.Role{
		ID:       utils.TrimmedStringValue(perm.Role.Id.String()),
		Name:     utils.TrimmedStringValue(perm.Role.Name),
		Resource: resource,
	}
	model.Role, diags = role.AsObjectValue(ctx)
	if diags.HasError() {
		return diags
	}

	model.Path = utils.TrimmedStringValue(string(perm.Path))
	model.Types = typesSet

	return diags
}
