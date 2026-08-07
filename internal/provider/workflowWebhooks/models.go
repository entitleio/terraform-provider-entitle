// Package workflowWebhooks provides the implementation of the Entitle Workflow Webhook
// resource and data source for Terraform.
//
// A workflow webhook is an HTTPS endpoint that Entitle calls as part of an approval
// workflow, for example to notify an external system that an access request requires
// attention. Webhooks are managed through the /public/v1/workflowsWebhooks endpoints
// and are referenced from approval workflow steps by their identifier.
package workflowWebhooks

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// WorkflowWebhookResourceModel describes the resource data model.
type WorkflowWebhookResourceModel struct {
	ID      types.String `tfsdk:"id" json:"id"`
	Name    types.String `tfsdk:"name" json:"name"`
	URL     types.String `tfsdk:"url" json:"url"`
	Headers types.Map    `tfsdk:"headers" json:"headers"`
}

// WorkflowWebhookDataSourceModel describes the data source data model.
type WorkflowWebhookDataSourceModel struct {
	ID      types.String `tfsdk:"id" json:"id"`
	Name    types.String `tfsdk:"name" json:"name"`
	URL     types.String `tfsdk:"url" json:"url"`
	Headers types.Map    `tfsdk:"headers" json:"headers"`
}
