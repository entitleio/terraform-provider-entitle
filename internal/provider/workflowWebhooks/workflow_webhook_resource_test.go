//go:build acceptance

package workflowWebhooks_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestWorkflowWebhookResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testhelpers.ProviderConfig + `

resource "entitle_workflow_webhook" "my_webhook" {
	name = "My Workflow Webhook"
	url  = "https://hooks.example.com/entitle/approvals"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("entitle_workflow_webhook.my_webhook", "name", "My Workflow Webhook"),
					resource.TestCheckResourceAttr("entitle_workflow_webhook.my_webhook", "url", "https://hooks.example.com/entitle/approvals"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("entitle_workflow_webhook.my_webhook", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "entitle_workflow_webhook.my_webhook",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update testing
			{
				Config: testhelpers.ProviderConfig + `

resource "entitle_workflow_webhook" "my_webhook" {
	name = "My Workflow Webhook UPDATED"
	url  = "https://hooks.example.com/entitle/approvals-v2"

	headers = {
		"X-Env" = "staging"
	}
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("entitle_workflow_webhook.my_webhook", "name", "My Workflow Webhook UPDATED"),
					resource.TestCheckResourceAttr("entitle_workflow_webhook.my_webhook", "url", "https://hooks.example.com/entitle/approvals-v2"),
					resource.TestCheckResourceAttr("entitle_workflow_webhook.my_webhook", "headers.X-Env", "staging"),
				),
			},
		},
	})
}
