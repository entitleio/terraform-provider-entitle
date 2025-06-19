package policies

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &PolicyResource{}
var _ resource.ResourceWithImportState = &PolicyResource{}

func NewPolicyResource() resource.Resource {
	return &PolicyResource{}
}

// PolicyResource defines the resource implementation.
type PolicyResource struct {
	client *client.ClientWithResponses
}

// PolicyResourceModel describes the resource data model.
type PolicyResourceModel struct {
	ID       types.String          `tfsdk:"id" json:"id"`
	Bundles  []*utils.IdNameModel  `tfsdk:"bundles" json:"bundles"`
	InGroups []*PolicyInGroupModel `tfsdk:"in_groups" json:"inGroups"`
	Roles    []*utils.Role         `tfsdk:"roles" json:"roles"`
}

func (r *PolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

func (r *PolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Entitle policy is a rule which manages users birthright permissions automatically, " +
			"a group of users is entitled to a set of permissions. When a user joins the group, e.g. upon joining " +
			"the organization, he will be granted with the permissions defined for the group automatically, and " +
			"upon leaving the group, e.g. leaving the organization, the permissions will be revoked automatically. " +
			"[Read more about policies](https://docs.beyondtrust.com/entitle/docs/birthright-policies).",
		Description: "Entitle policy is a rule which manages users birthright permissions automatically, " +
			"a group of users is entitled to a set of permissions. When a user joins the group, e.g. upon joining " +
			"the organization, he will be granted with the permissions defined for the group automatically, and " +
			"upon leaving the group, e.g. leaving the organization, the permissions will be revoked automatically. " +
			"[Read more about policies](https://docs.beyondtrust.com/entitle/docs/birthright-policies).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Entitle Policy identifier in uuid format",
				Description:         "Entitle Policy identifier in uuid format",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"roles": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Optional:            true,
							Required:            false,
							Description:         "The identifier of the role to be granted by the policy.",
							MarkdownDescription: "The identifier of the role to be granted by the policy.",
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
				Optional:            true,
				Description:         "A list of roles that the policy assigns to users. Each role grants access to a specific resource.",
				MarkdownDescription: "A list of roles that the policy assigns to users. Each role grants access to a specific resource.",
			},
			"bundles": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Optional:            true,
							Required:            false,
							Description:         "The identifier of the bundle to be assigned.",
							MarkdownDescription: "The identifier of the bundle to be assigned.",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							Description:         "The name of the bundle.",
							MarkdownDescription: "The name of the bundle.",
						},
					},
				},
				Optional:            true,
				Description:         "A list of bundles (collections of entitlements) to be assigned by the policy.",
				MarkdownDescription: "A list of bundles (collections of entitlements) to be assigned by the policy.",
			},
			"in_groups": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Optional:            true,
							Required:            false,
							Description:         "The unique identifier or email address of the IdP group.",
							MarkdownDescription: "The unique identifier or email address of the IdP group.",
						},
						"type": schema.StringAttribute{
							Optional:            false,
							Required:            true,
							Description:         "The type of group source (e.g. \"google\", \"okta\", etc.).",
							MarkdownDescription: "The type of group source (e.g. \"google\", \"okta\", etc.).",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							Description:         "The name of the group.",
							MarkdownDescription: "The name of the group.",
						},
					},
				},
				Optional:            false,
				Required:            true,
				Description:         "The list of identity provider (IdP) groups that the policy applies to. Users in these groups receive the defined roles or bundles.",
				MarkdownDescription: "The list of identity provider (IdP) groups that the policy applies to. Users in these groups receive the defined roles or bundles.",
			},
		},
	}
}

func (r *PolicyResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
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

// Create this function is responsible for creating a new resource of type Entitle Policy.
//
// Its reads the Terraform plan data provided in req.Plan and maps it to the PolicyResourceModel.
// And sends a request to the Entitle API to create the resource using API requests.
// If the creation is successful, it saves the resource's data into Terraform state.
func (r *PolicyResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var err error
	var plan PolicyResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	roles, diags := getRolesFromPlan(plan.Roles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	bundles, diags := getBundlesFromPlan(plan.Bundles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	inGroups := getInGroupsFromPlan(plan.InGroups)
	policyResp, err := r.client.PoliciesCreateWithResponse(ctx, client.PolicyCreateSchema{
		Bundles:  bundles,
		InGroups: inGroups,
		Roles:    roles,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to crete the policy, got error: %v", err),
		)
		return
	}

	if policyResp.HTTPResponse.StatusCode != 200 {
		errBody, _ := utils.GetErrorBody(policyResp.Body)
		if policyResp.HTTPResponse.StatusCode == http.StatusUnauthorized ||
			(policyResp.HTTPResponse.StatusCode == http.StatusBadRequest && strings.Contains(errBody.GetMessage(), "is not a valid uuid")) {
			resp.Diagnostics.AddError(
				"Client Error",
				"unauthorized token, update the entitle token and retry please",
			)
			return
		}

		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"failed to create the policy, status code: %d%s",
				policyResp.HTTPResponse.StatusCode,
				errBody.GetMessage(),
			),
		)
		return
	}

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a entitle policy resource")

	plan.ID = utils.TrimmedStringValue(policyResp.JSON200.Result[0].Id.String())
	plan, diags = convertFullPolicyResultResponseSchemaToModel(
		ctx,
		bundles,
		inGroups,
		roles,
		&policyResp.JSON200.Result[0],
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read this function is used to read an existing resource of type Entitle Policy.
//
// It retrieves the resource's data from the provider API requests.
// The retrieved data is then mapped to the PolicyResourceModel,
// and the data is saved to Terraform state.
func (r *PolicyResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data PolicyResourceModel

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
			fmt.Sprintf("failed to parse the resource id (%s) to UUID, got error: %s", data.ID.String(), err),
		)
		return
	}

	policyResp, err := r.client.PoliciesShowWithResponse(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to get the policy by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	if policyResp.HTTPResponse.StatusCode != 200 {
		errBody, _ := utils.GetErrorBody(policyResp.Body)
		if policyResp.HTTPResponse.StatusCode == http.StatusUnauthorized ||
			(policyResp.HTTPResponse.StatusCode == http.StatusBadRequest && strings.Contains(errBody.GetMessage(), "is not a valid uuid")) {
			resp.Diagnostics.AddError(
				"Client Error",
				"unauthorized token, update the entitle token and retry please",
			)
			return
		}

		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"failed to get the policy by the id (%s), status code: %d%s",
				uid.String(),
				policyResp.HTTPResponse.StatusCode,
				errBody.GetMessage(),
			),
		)
		return
	}

	data, diags = convertFullPolicyResultResponseSchemaToModel(ctx, nil, nil, nil, &policyResp.JSON200.Result[0])
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update this function handles updates to an existing resource of type Entitle Policy.
//
// It reads the updated Terraform plan data provided in req.Plan and maps it to the PolicyResourceModel.
// And sends a request to the Entitle API to update the resource using API requests.
// If the update is successful, it saves the updated resource data into Terraform state.
func (r *PolicyResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var data PolicyResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	uid, err := uuid.Parse(data.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("failed to parse the resource id (%s) to UUID, got error: %s", data.ID.String(), err),
		)
		return
	}

	roles, diags := getRolesFromPlan(data.Roles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	bundles, diags := getBundlesFromPlan(data.Bundles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	inGroups := getInGroupsFromPlan(data.InGroups)
	policyResp, err := r.client.PoliciesUpdateWithResponse(ctx, uid, client.PolicyUpdateSchema{
		Bundles:  &bundles,
		InGroups: &inGroups,
		Roles:    &roles,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to update the policy by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	if policyResp.HTTPResponse.StatusCode != 200 {
		errBody, _ := utils.GetErrorBody(policyResp.Body)
		if policyResp.HTTPResponse.StatusCode == http.StatusUnauthorized ||
			(policyResp.HTTPResponse.StatusCode == http.StatusBadRequest && strings.Contains(errBody.GetMessage(), "is not a valid uuid")) {
			resp.Diagnostics.AddError(
				"Client Error",
				"unauthorized token, update the entitle token and retry please",
			)
			return
		}

		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"failed to update the policy by the id (%s), status code: %d%s",
				uid.String(),
				policyResp.HTTPResponse.StatusCode,
				errBody.GetMessage(),
			),
		)
		return
	}

	data, diags = convertFullPolicyResultResponseSchemaToModel(
		ctx,
		bundles,
		inGroups,
		roles,
		&policyResp.JSON200.Result[0],
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete this function is responsible for deleting an existing resource of type
//
// It reads the resource's data from Terraform state, extracts the unique identifier,
// and sends a request to delete the resource using API requests.
// If the deletion is successful, it removes the resource from Terraform state.
func (r *PolicyResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data PolicyResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	parsedUUID, err := uuid.Parse(data.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to parse uuid of the policy, id: (%s), got error: %v", data.ID.String(), err),
		)
		return
	}

	httpResp, err := r.client.PoliciesDestroyWithResponse(ctx, parsedUUID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to delete policy, id: (%s), got error: %v", data.ID.String(), err),
		)
		return
	}

	if httpResp.HTTPResponse.StatusCode != 200 {
		errBody, _ := utils.GetErrorBody(httpResp.Body)
		if httpResp.HTTPResponse.StatusCode == http.StatusUnauthorized ||
			(httpResp.HTTPResponse.StatusCode == http.StatusBadRequest && strings.Contains(errBody.GetMessage(), "is not a valid uuid")) {
			resp.Diagnostics.AddError(
				"Client Error",
				"unauthorized token, update the entitle token and retry please",
			)
			return
		}

		if errBody.ID == "resource.notFound" {
			return
		}

		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"failed to delete the policy by the id (%s), status code: %d%s",
				data.ID.String(),
				httpResp.HTTPResponse.StatusCode,
				errBody.GetMessage(),
			),
		)
		return
	}
}

// ImportState this function is used to import an existing resource's state into Terraform.
//
// It extracts the resource's identifier from the import request and sets
// it in Terraform state using resource.ImportStatePassthroughID.
func (r *PolicyResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// convertFullPolicyResultResponseSchemaToModel is a utility function used to convert the API response data
// (of type client.FullPolicyResultResponseSchema) to a Terraform resource model (of type PolicyResourceModel).
//
// It extracts and transforms data from the API response into a format that can be stored in Terraform state.
// It returns the converted model and any diagnostic information if there are errors during the conversion.
func convertFullPolicyResultResponseSchemaToModel(
	ctx context.Context,
	planBundles []client.IdParamsSchema,
	planInGroups []client.InGroupSchema,
	planRoles []client.IdParamsSchema,
	data *client.FullPolicyResultResponseSchema,
) (PolicyResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Check if the API response data is nil
	if data == nil {
		diags.AddError(
			"No data",
			"Failed: the given schema data is nil",
		)

		return PolicyResourceModel{}, diags
	}

	var roles []*utils.Role
	var diagsRoles diag.Diagnostics
	if planRoles == nil {
		roles, diagsRoles = getRoles(ctx, data.Roles)
	} else {
		roles, diagsRoles = getRolesAsPlanned(ctx, planRoles, data.Roles)
	}

	diags.Append(diagsRoles...)
	if diags.HasError() {
		return PolicyResourceModel{}, diags
	}

	var bundles []*utils.IdNameModel
	if planBundles == nil {
		bundles = getBundles(data.Bundles)
	} else {
		bundles = getBundlesAsPlanned(planBundles, data.Bundles)
	}

	var inGroups []*PolicyInGroupModel
	if planInGroups == nil {
		inGroups = getInGroups(data.InGroups)
	} else {
		inGroups = getInGroupsAsPlanned(planInGroups, data.InGroups)
	}

	// Create the Terraform resource model using the extracted data
	return PolicyResourceModel{
		ID:       utils.TrimmedStringValue(data.Id.String()),
		Roles:    roles,
		Bundles:  bundles,
		InGroups: inGroups,
	}, diags
}
