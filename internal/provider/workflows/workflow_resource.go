package workflows

import (
	"context"
	"fmt"
	"math/big"

	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/numberdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &WorkflowResource{}
var _ resource.ResourceWithImportState = &WorkflowResource{}

func NewWorkflowResource() resource.Resource {
	return &WorkflowResource{}
}

// WorkflowResource defines the resource implementation.
type WorkflowResource struct {
	client *client.ClientWithResponses
}

// WorkflowResourceModel describes the resource data model.
type WorkflowResourceModel struct {
	ID    types.String          `tfsdk:"id" json:"id"`
	Name  types.String          `tfsdk:"name" json:"name"`
	Rules []*workflowRulesModel `tfsdk:"rules" json:"rules"`
}

func (r *WorkflowResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflow"
}

func (r *WorkflowResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A workflow in Entitle is a generic description of Just-In-Time permissions approval " +
			"process, which is triggered after the permissions were requested by a user. Who should approve by " +
			"approval order, to whom, and for how long. After the workflow is defined, it can be assigned to " +
			"multiple entities which are part of the Just-In-Time permissions approval process: integrations, " +
			"resources, roles and bundles." +
			"\n\nEvery workflow is comprised of multiple rules. Their order of is important, the first rule to be " +
			"validated sets the actual approval process for the permissions request.",
		Description: "A workflow in Entitle is a generic description of Just-In-Time permissions approval " +
			"process, which is triggered after the permissions were requested by a user. Who should approve by " +
			"approval order, to whom, and for how long. After the workflow is defined, it can be assigned to " +
			"multiple entities which are part of the Just-In-Time permissions approval process: integrations, " +
			"resources, roles and bundles." +
			"\n\nEvery workflow is comprised of multiple rules. Their order of is important, the first rule to be " +
			"validated sets the actual approval process for the permissions request.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Entitle Workflow identifier in uuid format",
				Description:         "Entitle Workflow identifier in uuid format",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Computed:            false,
				Required:            true,
				MarkdownDescription: "name",
				Description:         "name",
			},
			"rules": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"any_schedule": schema.BoolAttribute{
							Computed:            true,
							Optional:            true,
							Default:             booldefault.StaticBool(true),
							Description:         "any_schedule",
							MarkdownDescription: "any_schedule",
						},
						"sort_order": schema.NumberAttribute{
							Computed:            true,
							Optional:            true,
							Description:         "sort_order",
							MarkdownDescription: "sort_order",
							Default:             numberdefault.StaticBigFloat(big.NewFloat(0)),
						},
						"under_duration": schema.NumberAttribute{
							Computed:            true,
							Optional:            true,
							Description:         "under_duration",
							MarkdownDescription: "under_duration",
							Default:             numberdefault.StaticBigFloat(big.NewFloat(3600)),
						},
						"in_groups": schema.ListNestedAttribute{
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										Required:            false,
										Optional:            true,
										Description:         "",
										MarkdownDescription: "",
									},
									"name": schema.StringAttribute{
										Computed:            true,
										Description:         "name",
										MarkdownDescription: "name",
									},
								},
							},
							Optional:            true,
							Description:         "in_groups",
							MarkdownDescription: "in_groups",
						},
						"in_schedules": schema.ListNestedAttribute{
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										Required:            false,
										Optional:            true,
										Description:         "",
										MarkdownDescription: "",
									},
									"name": schema.StringAttribute{
										Computed:            true,
										Description:         "name",
										MarkdownDescription: "name",
									},
								},
							},
							Optional:            true,
							Description:         "in_schedules",
							MarkdownDescription: "in_schedules",
						},
						"approval_flow": schema.SingleNestedAttribute{
							Attributes: map[string]schema.Attribute{
								"steps": schema.ListNestedAttribute{
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"operator": schema.StringAttribute{
												Computed:            true,
												Optional:            true,
												Description:         "operator",
												MarkdownDescription: "operator",
												Default:             stringdefault.StaticString("and"),
											},
											"sort_order": schema.NumberAttribute{
												Computed:            true,
												Optional:            true,
												Description:         "sort_order",
												MarkdownDescription: "sort_order",
												Default:             numberdefault.StaticBigFloat(big.NewFloat(0)),
											},
											"notified_entities": schema.ListNestedAttribute{
												NestedObject: schema.NestedAttributeObject{
													Attributes: map[string]schema.Attribute{
														"type": schema.StringAttribute{
															Optional:            true,
															Description:         "",
															MarkdownDescription: "",
														},
														"user": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"id": schema.StringAttribute{
																	Optional:            true,
																	Description:         "",
																	MarkdownDescription: "",
																},
																"email": schema.StringAttribute{
																	Computed:            true,
																	Description:         "email",
																	MarkdownDescription: "email",
																},
															},
															Optional:            true,
															Description:         "user",
															MarkdownDescription: "user",
														},
														"group": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"id": schema.StringAttribute{
																	Optional:            true,
																	Description:         "",
																	MarkdownDescription: "",
																},
																"name": schema.StringAttribute{
																	Computed:            true,
																	Description:         "",
																	MarkdownDescription: "",
																},
															},
															Optional:            true,
															Description:         "group",
															MarkdownDescription: "group",
														},
														"schedule": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"id": schema.StringAttribute{
																	Optional:            true,
																	Description:         "",
																	MarkdownDescription: "",
																},
																"name": schema.StringAttribute{
																	Computed:            true,
																	Description:         "",
																	MarkdownDescription: "",
																},
															},
															Optional:            true,
															Description:         "group",
															MarkdownDescription: "group",
														},
														"value": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"notified": schema.StringAttribute{
																	Optional:            true,
																	Description:         "",
																	MarkdownDescription: "",
																},
															},
															Optional:            true,
															Description:         "value",
															MarkdownDescription: "value",
														},
													},
												},
												Optional:            true,
												Description:         "in_groups",
												MarkdownDescription: "in_groups",
											},
											"approval_entities": schema.ListNestedAttribute{
												NestedObject: schema.NestedAttributeObject{
													Attributes: map[string]schema.Attribute{
														"type": schema.StringAttribute{
															Optional:            true,
															Description:         "",
															MarkdownDescription: "",
														},
														"user": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"id": schema.StringAttribute{
																	Optional:            true,
																	Description:         "",
																	MarkdownDescription: "",
																},
																"email": schema.StringAttribute{
																	Computed:            true,
																	Description:         "email",
																	MarkdownDescription: "email",
																},
															},
															Optional:            true,
															Description:         "user",
															MarkdownDescription: "user",
														},
														"group": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"id": schema.StringAttribute{
																	Optional:            true,
																	Description:         "",
																	MarkdownDescription: "",
																},
																"name": schema.StringAttribute{
																	Computed:            true,
																	Description:         "",
																	MarkdownDescription: "",
																},
															},
															Optional:            true,
															Description:         "group",
															MarkdownDescription: "group",
														},
														"schedule": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"id": schema.StringAttribute{
																	Computed:            true,
																	Description:         "",
																	MarkdownDescription: "",
																},
																"name": schema.StringAttribute{
																	Optional:            true,
																	Description:         "",
																	MarkdownDescription: "",
																},
															},
															Optional:            true,
															Description:         "group",
															MarkdownDescription: "group",
														},
														"value": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"approval": schema.StringAttribute{
																	Optional:            true,
																	Description:         "",
																	MarkdownDescription: "",
																},
															},
															Optional:            true,
															Description:         "value",
															MarkdownDescription: "value",
														},
													},
												},
												Optional:            true,
												Description:         "in_groups",
												MarkdownDescription: "in_groups",
											},
										},
									},
									Optional:            true,
									Description:         "steps",
									MarkdownDescription: "steps",
								},
							},
							Optional:            true,
							Description:         "approval_flow",
							MarkdownDescription: "approval_flow",
						},
					},
				},
				Optional:            true,
				Description:         "rules",
				MarkdownDescription: "rules",
			},
		},
	}
}

func (r *WorkflowResource) Configure(
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

// Create this function is responsible for creating a new resource of type Entitle Workflow.
//
// Its reads the Terraform plan data provided in req.Plan and maps it to the WorkflowResourceModel.
// And sends a request to the Entitle API to create the resource using API requests.
// If the creation is successful, it saves the resource's data into Terraform state.
func (r *WorkflowResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var err error
	var plan WorkflowResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var name string
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		if plan.Name.ValueString() != "" {
			name = plan.Name.ValueString()
		}
	} else {
		resp.Diagnostics.AddError(
			"Client Error",
			"failed create workflow resource required name variable",
		)

		return
	}

	rules, diags := getWorkflowsRules(ctx, plan.Rules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	workflowResp, err := r.client.WorkflowsCreateWithResponse(ctx, client.WorkflowCreateBodySchema{
		Name:  name,
		Rules: rules,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to crete the workflow, got error: %v", err),
		)
		return
	}

	if workflowResp.HTTPResponse.StatusCode != 200 {
		errBody, _ := utils.GetErrorBody(workflowResp.Body)
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"failed to create the workflow, status code: %d%s",
				workflowResp.HTTPResponse.StatusCode,
				errBody.GetMessage(),
			),
		)
		return
	}

	tflog.Trace(ctx, "created a entitle workflow resource")

	plan.ID = types.StringValue(workflowResp.JSON200.Result.Id.String())

	plan, diags = convertFullWorkflowResultResponseSchemaToModel(ctx, &workflowResp.JSON200.Result)
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

// Read this function is used to read an existing resource of type Entitle Workflow.
//
// It retrieves the resource's data from the provider API requests.
// The retrieved data is then mapped to the WorkflowResourceModel,
// and the data is saved to Terraform state.
func (r *WorkflowResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data WorkflowResourceModel

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

	workflowResp, err := r.client.WorkflowsShowWithResponse(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to get the workflow by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	if workflowResp.HTTPResponse.StatusCode != 200 {
		errBody, _ := utils.GetErrorBody(workflowResp.Body)
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"failed to get the workflow by the id (%s), status code: %d%s",
				uid.String(),
				workflowResp.HTTPResponse.StatusCode,
				errBody.GetMessage(),
			),
		)
		return
	}

	data, diags = convertFullWorkflowResultResponseSchemaToModel(ctx, &workflowResp.JSON200.Result)
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

// Update this function handles updates to an existing resource of type Entitle Workflow.
//
// It reads the updated Terraform plan data provided in req.Plan and maps it to the WorkflowResourceModel.
// And sends a request to the Entitle API to update the resource using API requests.
// If the update is successful, it saves the updated resource data into Terraform state.
func (r *WorkflowResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var data WorkflowResourceModel

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

	var name *string
	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		if data.Name.ValueString() != "" {
			name = data.Name.ValueStringPointer()
		}
	}

	rules, diags := getWorkflowsRules(ctx, data.Rules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	workflowResp, err := r.client.WorkflowsUpdateWithResponse(ctx, uid, client.WorkflowUpdatedBodySchema{
		Name:  name,
		Rules: utils.WorkflowRuleSchemaPointer(rules),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to update the workflow by the id (%s), got error: %s", uid.String(), err),
		)

		return
	}

	if workflowResp.HTTPResponse.StatusCode != 200 {
		errBody, _ := utils.GetErrorBody(workflowResp.Body)
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"failed to get the update by the id (%s), status code: %d%s",
				uid.String(),
				workflowResp.HTTPResponse.StatusCode,
				errBody.GetMessage(),
			),
		)
		return
	}

	data, diags = convertFullWorkflowResultResponseSchemaToModel(ctx, &workflowResp.JSON200.Result)
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
func (r *WorkflowResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data WorkflowResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	parsedUUID, err := uuid.Parse(data.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to parse uuid of the workflow, id: (%s), got error: %v", data.ID.String(), err),
		)
		return
	}

	httpResp, err := r.client.WorkflowsDestroyWithResponse(ctx, parsedUUID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to delete workflow, id: (%s), got error: %v", data.ID.String(), err),
		)
		return
	}

	if httpResp.HTTPResponse.StatusCode != 200 {
		errBody, _ := utils.GetErrorBody(httpResp.Body)
		if errBody.ID == "resource.notFound" {
			return
		}

		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"failed to delete workflow by the id (%s), status code: %d%s",
				parsedUUID.String(),
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
func (r *WorkflowResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
