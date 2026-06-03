// Package resources provides the implementation of the Entitle Resource Synced resource for Terraform.
// It defines the resource type, its schema, and the CRUD operations for managing synced Resources.
package resources

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/docs"
	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"github.com/entitleio/terraform-provider-entitle/internal/validators"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ResourceSyncedResource{}
var _ resource.ResourceWithImportState = &ResourceSyncedResource{}

// NewResourceSyncedResource creates a new instance of the ResourceSyncedResource.
func NewResourceSyncedResource() resource.Resource {
	return &ResourceSyncedResource{}
}

// ResourceSyncedResource defines the resource implementation.
type ResourceSyncedResource struct {
	client *client.ClientWithResponses
}

// Metadata sets the metadata for the resource.
func (r *ResourceSyncedResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_synced"
}

// Schema sets the schema for the resource.
func (r *ResourceSyncedResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: docs.ResourceSyncedResourceMarkdownDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Entitle Resource identifier in UUID format",
				Description:         "Entitle Resource identifier in UUID format",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The display name of the resource. Used together with integration.id to look up the existing synced resource.",
				Description:         "The display name of the resource. Used together with integration.id to look up the existing synced resource.",
				Validators: []validator.String{
					validators.NewName(2, 50),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"integration": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Required:            true,
						Description:         "The integration's id. Used together with name to look up the existing synced resource.",
						MarkdownDescription: "The integration's id. Used together with name to look up the existing synced resource.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"name": schema.StringAttribute{
						Computed:            true,
						Description:         "The integration's name.",
						MarkdownDescription: "The integration's name.",
					},
				},
				Required:            true,
				Description:         "The integration this resource belongs to. Required to locate the synced resource.",
				MarkdownDescription: "The integration this resource belongs to. Required to locate the synced resource.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
			"allowed_durations": schema.SetAttribute{
				ElementType:         types.NumberType,
				Optional:            true,
				Computed:            true,
				Description:         "As the admin, you can set different durations for the resource, compared to the workflow linked to it.  \nAllowed values:\n  - 1800 - 30min\n  - 3600 - 1 hour\n  - 10800 - 3 hours\n  - 21600 - 6 hours\n  - 43200 - 12 hours\n  - 57600 - 16 hours\n  - 86400 - 24 hours\n  - 259200 - 3 days\n  - 604800 - 7 days\n  - 2628000  - ~30,4 days\n  - 7884000 - 91,25 days\n  - 15768000 - 182,5 days\n  - 31536000 - 365 days\n  - 63072000 - 730 days\n  - -1 - unlimited",
				MarkdownDescription: "As the admin, you can set different durations for the resource, compared to the workflow linked to it.  \nAllowed values:\n  - 1800 - 30min\n  - 3600 - 1 hour\n  - 10800 - 3 hours\n  - 21600 - 6 hours\n  - 43200 - 12 hours\n  - 57600 - 16 hours\n  - 86400 - 24 hours\n  - 259200 - 3 days\n  - 604800 - 7 days\n  - 2628000  - ~30,4 days\n  - 7884000 - 91,25 days\n  - 15768000 - 182,5 days\n  - 31536000 - 365 days\n  - 63072000 - 730 days\n  - -1 - unlimited",
				Validators: []validator.Set{
					validators.NewSetMinLength(1),
				},
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"workflow": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "The workflow's id.",
						MarkdownDescription: "The workflow's id.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"name": schema.StringAttribute{
						Computed:            true,
						Description:         "The workflow's name.",
						MarkdownDescription: "The workflow's name.",
					},
				},
				Optional:            true,
				Computed:            true,
				Description:         "The default approval workflow for entitlements for this resource.",
				MarkdownDescription: "The default approval workflow for entitlements for this resource.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
			"requestable": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Indicates if the resource is requestable.",
				MarkdownDescription: "Indicates if the resource is requestable.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"owner": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "The owner's id.",
						MarkdownDescription: "The owner's id.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"email": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "The owner's email (lowercase) is used when id was not provided.",
						MarkdownDescription: "The owner's email (lowercase) is used when id was not provided.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
				Optional:            true,
				Computed:            true,
				Description:         "The owner of the resource, used for administrative purposes and approval workflows.",
				MarkdownDescription: "The owner of the resource, used for administrative purposes and approval workflows.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
			"maintainers": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							Description:         "\"user\" or \"group\" (default: \"user\")",
							MarkdownDescription: "\"user\" or \"group\" (default: \"user\")",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"entity": schema.SingleNestedAttribute{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Optional:            true,
									Computed:            true,
									Description:         "Maintainer's unique identifier.",
									MarkdownDescription: "Maintainer's unique identifier.",
								},
								"email": schema.StringAttribute{
									Computed:            true,
									Description:         "Maintainer's email.",
									MarkdownDescription: "Maintainer's email.",
								},
							},
							Optional:            true,
							Computed:            true,
							Description:         "Maintainer's entity.",
							MarkdownDescription: "Maintainer's entity.",
						},
					},
				},
				Optional:            true,
				Computed:            true,
				Description:         "Secondary owners of the resource. Can be users or IDP groups.",
				MarkdownDescription: "Secondary owners of the resource. Can be users or IDP groups.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"tags": schema.SetAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				Description:         "System-generated searchable tags.",
				MarkdownDescription: "System-generated searchable tags.",
			},
			"user_defined_tags": schema.SetAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Description:         "User-defined searchable metadata tags.",
				MarkdownDescription: "User-defined searchable metadata tags.",
				Validators: []validator.Set{
					validators.NewSetMinLength(1),
				},
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"user_defined_description": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					validators.NewName(2, 2048),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"prerequisite_permissions": schema.ListNestedAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Users granted any role from this resource through a request will automatically receive the permissions to the roles selected below.",
				MarkdownDescription: "Users granted any role from this resource through a request will automatically receive the permissions to the roles selected below.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"default": schema.BoolAttribute{
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(false),
							Description:         "Indicates whether this prerequisite permission should be automatically granted as a default permission. When set to true, users will receive this permission by default when accessing the associated resource (default: false).",
							MarkdownDescription: "Indicates whether this prerequisite permission should be automatically granted as a default permission. When set to true, users will receive this permission by default when accessing the associated resource (default: false).",
							PlanModifiers: []planmodifier.Bool{
								boolplanmodifier.UseStateForUnknown(),
							},
						},
						"role": schema.SingleNestedAttribute{
							Optional: true,
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Optional:            true,
									Computed:            true,
									Description:         "The identifier of the role to be granted.",
									MarkdownDescription: "The identifier of the role to be granted.",
									PlanModifiers: []planmodifier.String{
										stringplanmodifier.UseStateForUnknown(),
									},
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
		},
	}
}

// Configure configures the resource with the provided client.
func (r *ResourceSyncedResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ClientWithResponses, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = c
}

// Create handles the "creation" of an entitle_resource_synced resource.
//
// Since synced resources already exist in Entitle (managed by the integration),
// this operation looks up the resource by name + integration.id and imports it into state.
// No resource is created; no DELETE will be sent on destroy.
func (r *ResourceSyncedResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Use a plan-read struct with framework types for optional+computed fields so that
	// unknown values on the first apply do not cause a conversion error.
	var createPlan struct {
		Name        types.String       `tfsdk:"name"`
		Integration *utils.IdNameModel `tfsdk:"integration"`
		// Remaining computed/optional fields accept unknown values via framework types.
		ID                      types.String `tfsdk:"id"`
		AllowedDurations        types.Set    `tfsdk:"allowed_durations"`
		Workflow                types.Object `tfsdk:"workflow"`
		Requestable             types.Bool   `tfsdk:"requestable"`
		Owner                   types.Object `tfsdk:"owner"`
		Maintainers             types.List   `tfsdk:"maintainers"`
		Tags                    types.Set    `tfsdk:"tags"`
		UserDefinedTags         types.Set    `tfsdk:"user_defined_tags"`
		UserDefinedDescription  types.String `tfsdk:"user_defined_description"`
		PrerequisitePermissions types.List   `tfsdk:"prerequisite_permissions"`
	}

	diags := req.Plan.Get(ctx, &createPlan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := createPlan.Name.ValueString()
	integrationID := createPlan.Integration.ID.ValueString()

	resourceID, err := r.getResourceIDByName(ctx, integrationID, name)
	if err != nil {
		resp.Diagnostics.AddError("Resource not found", fmt.Sprintf(
			"Failed to get the Resource by name (%s) and integration (%s): %s",
			name, integrationID, err.Error(),
		))
		return
	}

	apiResp, err := r.client.ResourcesShowWithResponse(ctx, *resourceID)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to get the resource. If you want to create a new one use entitle_resource. Got error: %v", err),
		)
		return
	}

	err = utils.HTTPResponseToError(apiResp.StatusCode(), apiResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf("Failed to get the resource: %s", err.Error()),
		)
		return
	}

	if !utils.IsApplicationWithSyncedResources(apiResp.JSON200.Result.Integration.Application.Name) {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			"The matched resource belongs to a manual or virtual integration. Use entitle_resource instead.",
		)
		return
	}

	state, diags := convertFullResourceResultResponseSchemaToModel(ctx, &apiResp.JSON200.Result)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

// Read retrieves the current state of an entitle_resource_synced resource.
func (r *ResourceSyncedResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResourceResourceModel

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	uid, err := uuid.Parse(data.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Failed to parse the resource id (%s) to UUID, got error: %s", data.ID.String(), err),
		)
		return
	}

	resourceResp, err := r.client.ResourcesShowWithResponse(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to get the resource by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(resourceResp.HTTPResponse.StatusCode, resourceResp.Body)
	if err != nil {
		if errors.Is(err, utils.ErrNotFound) {
			tflog.Debug(ctx, "Resource no longer exists, removing from state")
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf("Failed to get the resource by the id (%s), %s", uid.String(), err.Error()),
		)
		return
	}

	if !utils.IsApplicationWithSyncedResources(resourceResp.JSON200.Result.Integration.Application.Name) {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			"The resource belongs to a manual or virtual integration. Use entitle_resource instead.",
		)
		return
	}

	data, diags = convertFullResourceResultResponseSchemaToModel(ctx, &resourceResp.JSON200.Result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

// Update handles updates to an entitle_resource_synced resource.
// Only mutable Entitle settings are updated; the underlying synced resource is not recreated.
func (r *ResourceSyncedResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ResourceResourceModel

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	uid, err := uuid.Parse(data.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Failed to parse the given id to UUID format, got error: %v", err),
		)
		return
	}

	var allowedDurations *[]client.EnumAllowedDurations
	if !data.AllowedDurations.IsNull() && !data.AllowedDurations.IsUnknown() {
		aDurations, diags := utils.ConvertTerraformSetToAllowedDurations(ctx, data.AllowedDurations)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		if len(aDurations) > 0 {
			allowedDurations = &aDurations
		}
	}

	var workflow *client.IdParamsSchema
	if data.Workflow != nil && !data.Workflow.ID.IsNull() && !data.Workflow.ID.IsUnknown() {
		workflow = new(client.IdParamsSchema)
		workflow.Id, err = uuid.Parse(data.Workflow.ID.String())
		if err != nil {
			resp.Diagnostics.AddError(
				"Client Error",
				fmt.Sprintf("Failed to parse given workflow id to UUID, got error: %v", err),
			)
			return
		}
	}

	var owner *client.UserEntitySchema
	if data.Owner != nil {
		o := &client.UserEntitySchema{}
		if v := data.Owner.Id.ValueString(); v != "" {
			o.Id = utils.TrimPrefixSuffix(v)
		} else if v := data.Owner.Email.ValueString(); v != "" {
			o.Id = utils.TrimPrefixSuffix(v)
		}
		if o.Id != "" {
			owner = o
		}
	}

	maintainers, buildErr := buildUpdateMaintainers(ctx, data.Maintainers)
	if buildErr != nil {
		resp.Diagnostics.AddError("Client Error", buildErr.Error())
		return
	}

	var prerequisitePermissions *[][]client.IntegrationResourcesUpdateBodySchema_PrerequisitePermissions_Item
	if len(data.PrerequisitePermissions) > 0 {
		ppData := make([][]client.IntegrationResourcesUpdateBodySchema_PrerequisitePermissions_Item, 0, len(data.PrerequisitePermissions))
		for _, pp := range data.PrerequisitePermissions {
			if pp.Role.ID.IsNull() || pp.Role.ID.IsUnknown() {
				continue
			}

			item := client.IntegrationResourcesUpdateBodySchema_PrerequisitePermissions_Item{}
			mergeErr := item.MergePrerequisitePermissionCreateBodySchema(client.PrerequisitePermissionCreateBodySchema{
				Default: pp.Default.ValueBool(),
				Role: map[string]interface{}{
					"id": pp.Role.ID.ValueString(),
				},
			})
			if mergeErr != nil {
				resp.Diagnostics.AddError(
					"Client Error",
					fmt.Sprintf("Failed to merge prerequisite permission data, error: %v", mergeErr),
				)
			}

			ppData = append(ppData, []client.IntegrationResourcesUpdateBodySchema_PrerequisitePermissions_Item{item})
		}
		prerequisitePermissions = &ppData
	}

	request := client.ResourcesUpdateJSONRequestBody{
		PrerequisitePermissions: prerequisitePermissions,
		UserDefinedDescription:  data.UserDefinedDescription.ValueStringPointer(),
	}

	if !data.Requestable.IsNull() && !data.Requestable.IsUnknown() {
		request.Requestable = data.Requestable.ValueBoolPointer()
	}

	if len(maintainers) > 0 {
		request.Maintainers = &maintainers
	}

	if owner != nil {
		request.Owner = owner
	}

	if allowedDurations != nil {
		request.AllowedDurations = allowedDurations
	}

	if workflow != nil {
		request.Workflow = workflow
	}

	if !data.UserDefinedTags.IsNull() && !data.UserDefinedTags.IsUnknown() {
		var userDefinedTags []string
		for _, tag := range data.UserDefinedTags.Elements() {
			if sv, ok := tag.(types.String); ok {
				userDefinedTags = append(userDefinedTags, sv.ValueString())
			}
		}
		if len(userDefinedTags) > 0 {
			request.UserDefinedTags = &userDefinedTags
		}
	}

	resourceResp, err := r.client.ResourcesUpdateWithResponse(ctx, uid, request)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to update the resource by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(resourceResp.HTTPResponse.StatusCode, resourceResp.Body)
	if err != nil {
		if errors.Is(err, utils.ErrNotFound) {
			tflog.Debug(ctx, "Resource no longer exists, removing from state")
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf("Failed to update the resource by the id (%s), %s", uid.String(), err.Error()),
		)
		return
	}

	data, diags = convertFullResourceResultResponseSchemaToModel(ctx, &resourceResp.JSON200.Result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

// Delete is a no-op for synced resources — Terraform state is removed but no DELETE
// request is sent to Entitle, because synced resources are owned by the upstream integration.
func (r *ResourceSyncedResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Read state to satisfy the framework, then return without making any API call.
	var data ResourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
}

// ImportState imports an existing entitle_resource_synced by its UUID.
func (r *ResourceSyncedResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// getResourceIDByName paginates through ResourcesIndex to find a resource by exact name
// within the given integrationId.
//
// IntegrationResourceListItemResponseSchema does not implement the NameableID interface
// (no GetName/GetID methods), so we cannot use the generic FindIDByName helper here.
func (r *ResourceSyncedResource) getResourceIDByName(ctx context.Context, integrationID string, name string) (*uuid.UUID, error) {
	page := 1
	for {
		params := client.ResourcesIndexParams{
			PerPage:       utils.IntPointer(100),
			Page:          utils.IntPointer(page),
			Search:        utils.StringPointer(name),
			IntegrationId: integrationID,
		}

		resp, err := r.client.ResourcesIndexWithResponse(ctx, &params)
		if err != nil {
			return nil, fmt.Errorf("failed to list resources: %w", err)
		}

		if resp.HTTPResponse.StatusCode >= http.StatusBadRequest {
			return nil, fmt.Errorf("API returned status %d while listing resources (page %d)", resp.HTTPResponse.StatusCode, page)
		}

		if resp.JSON200 == nil || resp.JSON200.Result == nil {
			return nil, fmt.Errorf("received invalid response structure (page %d)", page)
		}

		for _, item := range resp.JSON200.Result {
			if item.Name == name {
				id := item.Id
				return &id, nil
			}
		}

		if page >= int(resp.JSON200.Pagination.TotalPages) {
			break
		}
		page++
	}

	return nil, fmt.Errorf("resource with name %q not found in integration %s", name, integrationID)
}
