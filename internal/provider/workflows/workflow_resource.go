package workflows

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/entitleio/terraform-provider-entitle/internal/validators"

	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"

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

	"github.com/entitleio/terraform-provider-entitle/internal/client"
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
			"validated sets the actual approval process for the permissions request. ",
		Description: "A workflow in Entitle is a generic description of Just-In-Time permissions approval " +
			"process, which is triggered after the permissions were requested by a user. Who should approve by " +
			"approval order, to whom, and for how long. After the workflow is defined, it can be assigned to " +
			"multiple entities which are part of the Just-In-Time permissions approval process: integrations, " +
			"resources, roles and bundles." +
			"\n\nEvery workflow is comprised of multiple rules. Their order of is important, the first rule to be " +
			"validated sets the actual approval process for the permissions request. [Read more about workflows](https://docs.beyondtrust.com/entitle/docs/approval-workflows).",
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
				MarkdownDescription: "The human-readable name of the workflow. Must be between 2 and 50 characters.",
				Description:         "The human-readable name of the workflow. Must be between 2 and 50 characters.",
				Validators: []validator.String{
					validators.NewName(2, 50),
				},
			},
			"rules": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"any_schedule": schema.BoolAttribute{
							Computed:            true,
							Optional:            true,
							Default:             booldefault.StaticBool(true),
							Description:         "Indicates whether the rule applies at any schedule. Defaults to true.",
							MarkdownDescription: "Indicates whether the rule applies at any schedule. Defaults to true.",
						},
						"sort_order": schema.NumberAttribute{
							Computed:            true,
							Optional:            true,
							Description:         "Determines the evaluation priority of the rule.",
							MarkdownDescription: "Determines the evaluation priority of the rule.",
							Default:             numberdefault.StaticBigFloat(big.NewFloat(0)),
						},
						"under_duration": schema.NumberAttribute{
							Computed:            true,
							Optional:            true,
							Description:         "Maximum request duration (in seconds) for which the rule applies. Defaults to 3600 seconds (1 hour).",
							MarkdownDescription: "Maximum request duration (in seconds) for which the rule applies. Defaults to 3600 seconds (1 hour).",
							Default:             numberdefault.StaticBigFloat(big.NewFloat(3600)),
						},
						"in_groups": schema.SetNestedAttribute{
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										Required:            false,
										Optional:            true,
										Description:         "A unique identifier of the group",
										MarkdownDescription: "A unique identifier of the group",
									},
									"name": schema.StringAttribute{
										Computed:            true,
										Description:         "The name of the group.",
										MarkdownDescription: "The name of the group.",
									},
								},
							},
							Optional:            true,
							Description:         "List of user groups for which this rule is applicable.",
							MarkdownDescription: "List of user groups for which this rule is applicable.",
						},
						"in_schedules": schema.SetNestedAttribute{
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										Required:            false,
										Optional:            true,
										Description:         "A unique identifier of the schedule",
										MarkdownDescription: "A unique identifier of the schedule",
									},
									"name": schema.StringAttribute{
										Computed:            true,
										Description:         "The name of the schedule.",
										MarkdownDescription: "The name of the schedule.",
									},
								},
							},
							Optional:            true,
							Description:         "List of schedules during which this rule is valid.",
							MarkdownDescription: "List of schedules during which this rule is valid.",
						},
						"approval_flow": schema.SingleNestedAttribute{
							Attributes: map[string]schema.Attribute{
								"steps": schema.ListNestedAttribute{
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"operator": schema.StringAttribute{
												Computed:            true,
												Optional:            true,
												Description:         "Logical operator for combining approval entities.",
												MarkdownDescription: "Logical operator for combining approval entities.",
												Default:             stringdefault.StaticString("and"),
											},
											"sort_order": schema.NumberAttribute{
												Computed:            true,
												Optional:            true,
												Description:         "Order of the step within the approval flow. Lower numbers indicate earlier steps.",
												MarkdownDescription: "Order of the step within the approval flow. Lower numbers indicate earlier steps.",
												Default:             numberdefault.StaticBigFloat(big.NewFloat(0)),
											},
											"notified_entities": schema.SetNestedAttribute{
												NestedObject: schema.NestedAttributeObject{
													Attributes: map[string]schema.Attribute{
														"type": schema.StringAttribute{
															Optional:            true,
															Description:         "Type of notified entity",
															MarkdownDescription: "Type of notified entity",
														},
														"user": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"id": schema.StringAttribute{
																	Optional:            true,
																	Description:         "Unique identifier of the notified user.",
																	MarkdownDescription: "Unique identifier of the notified user.",
																},
																"email": schema.StringAttribute{
																	Computed:            true,
																	Description:         "Email address of the notified user.",
																	MarkdownDescription: "Email address of the notified user.",
																},
															},
															Optional:            true,
															Description:         "Represents an individual user who will be notified during this step of the approval process.",
															MarkdownDescription: "Represents an individual user who will be notified during this step of the approval process.",
														},
														"group": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"id": schema.StringAttribute{
																	Optional:            true,
																	Description:         "A unique identifier of the group",
																	MarkdownDescription: "A unique identifier of the group",
																},
																"name": schema.StringAttribute{
																	Computed:            true,
																	Description:         "Name of the notified group.",
																	MarkdownDescription: "Name of the notified group.",
																},
															},
															Optional:            true,
															Description:         "Represents a user group whose members will be notified during this step of the approval process.",
															MarkdownDescription: "Represents a user group whose members will be notified during this step of the approval process.",
														},
														"schedule": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"id": schema.StringAttribute{
																	Optional:            true,
																	Description:         "A unique identifier of the schedule",
																	MarkdownDescription: "A unique identifier of the schedule",
																},
																"name": schema.StringAttribute{
																	Computed:            true,
																	Description:         "A name of the schedule",
																	MarkdownDescription: "A name of the schedule",
																},
															},
															Optional:            true,
															Description:         "Schedule applied to the approval entity.",
															MarkdownDescription: "Schedule applied to the approval entity.",
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
												Description:         "List of users or groups to be notified during this approval step.",
												MarkdownDescription: "List of users or groups to be notified during this approval step.",
											},
											"approval_entities": schema.SetNestedAttribute{
												NestedObject: schema.NestedAttributeObject{
													Attributes: map[string]schema.Attribute{
														"type": schema.StringAttribute{
															Optional:            true,
															Description:         "Type of approval entity.",
															MarkdownDescription: "Type of approval entity.",
														},
														"user": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"id": schema.StringAttribute{
																	Optional:            true,
																	Description:         "Unique identifier of the approver.",
																	MarkdownDescription: "Unique identifier of the approver.",
																},
																"email": schema.StringAttribute{
																	Computed:            true,
																	Description:         "Email address of the approver.",
																	MarkdownDescription: "Email address of the approver.",
																},
															},
															Optional:            true,
															Description:         "Represents an individual user who is required to approve the permission request at this step.",
															MarkdownDescription: "Represents an individual user who is required to approve the permission request at this step.",
														},
														"group": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"id": schema.StringAttribute{
																	Optional:            true,
																	Description:         "Unique identifier of the approver group.",
																	MarkdownDescription: "Unique identifier of the approver group.",
																},
																"name": schema.StringAttribute{
																	Computed:            true,
																	Description:         "Name of the approver group.",
																	MarkdownDescription: "Name of the approver group.",
																},
															},
															Optional:            true,
															Description:         "Represents a group whose members are responsible for approving the permission request at this step.",
															MarkdownDescription: "Represents a group whose members are responsible for approving the permission request at this step.",
														},
														"schedule": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"id": schema.StringAttribute{
																	Computed:            true,
																	Description:         "Unique identifier of the schedule for the approval entity.",
																	MarkdownDescription: "Unique identifier of the schedule for the approval entity.",
																},
																"name": schema.StringAttribute{
																	Optional:            true,
																	Description:         "Name of the approval schedule.",
																	MarkdownDescription: "Name of the approval schedule.",
																},
															},
															Optional:            true,
															Description:         "Schedule applied to the approval entity.",
															MarkdownDescription: "Schedule applied to the approval entity.",
														},
														"value": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"approval": schema.StringAttribute{
																	Optional:            true,
																	Description:         "Specifies the approval condition or requirement for the entity in this step. For example, it could indicate whether the approval is mandatory, optional, or has a certain threshold. This field helps customize the approval logic at a granular level.",
																	MarkdownDescription: "Specifies the approval condition or requirement for the entity in this step. For example, it could indicate whether the approval is mandatory, optional, or has a certain threshold. This field helps customize the approval logic at a granular level.",
																},
															},
															Required:            true,
															Description:         "Holds additional metadata or configuration related to the entity’s role in the approval step. This can include specific rules, conditions, or statuses that influence how the approval or notification behaves.",
															MarkdownDescription: "Holds additional metadata or configuration related to the entity’s role in the approval step. This can include specific rules, conditions, or statuses that influence how the approval or notification behaves.",
														},
													},
												},
												Optional:            true,
												Description:         "List of users or groups that must approve in this step.",
												MarkdownDescription: "List of users or groups that must approve in this step.",
											},
										},
									},
									Optional:            true,
									Description:         "List of approval steps defining the sequence and conditions of approval.",
									MarkdownDescription: "List of approval steps defining the sequence and conditions of approval.",
								},
							},
							Optional:            true,
							Description:         "The approval process defined by one or more ordered steps. Each step includes approvers and conditions.",
							MarkdownDescription: "The approval process defined by one or more ordered steps. Each step includes approvers and conditions.",
						},
					},
				},
				Optional:            true,
				Description:         "A list of rules that determine how approvals should be handled based on specific conditions.",
				MarkdownDescription: "A list of rules that determine how approvals should be handled based on specific conditions.",
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

	name := plan.Name.ValueString()
	if name == "" {
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
		if workflowResp.HTTPResponse.StatusCode == http.StatusUnauthorized ||
			(workflowResp.HTTPResponse.StatusCode == http.StatusBadRequest && strings.Contains(errBody.GetMessage(), "is not a valid uuid")) {
			resp.Diagnostics.AddError(
				"Client Error",
				"unauthorized token, update the entitle token and retry please",
			)
			return
		}

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

	plan.ID = utils.TrimmedStringValue(workflowResp.JSON200.Result.Id.String())

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
		if workflowResp.HTTPResponse.StatusCode == http.StatusUnauthorized ||
			(workflowResp.HTTPResponse.StatusCode == http.StatusBadRequest && strings.Contains(errBody.GetMessage(), "is not a valid uuid")) {
			resp.Diagnostics.AddError(
				"Client Error",
				"unauthorized token, update the entitle token and retry please",
			)
			return
		}

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

	name := data.Name.ValueStringPointer()

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
		if workflowResp.HTTPResponse.StatusCode == http.StatusUnauthorized ||
			(workflowResp.HTTPResponse.StatusCode == http.StatusBadRequest && strings.Contains(errBody.GetMessage(), "is not a valid uuid")) {
			resp.Diagnostics.AddError(
				"Client Error",
				"unauthorized token, update the entitle token and retry please",
			)
			return
		}

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
