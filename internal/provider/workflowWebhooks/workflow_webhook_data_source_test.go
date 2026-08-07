//go:build acceptance

package workflowWebhooks_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestWorkflowWebhookDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read by id testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_workflow_webhook" "by_id" {
	id = "%s"
}
`, os.Getenv("ENTITLE_WORKFLOW_WEBHOOK_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("data.entitle_workflow_webhook.by_id", "id", os.Getenv("ENTITLE_WORKFLOW_WEBHOOK_ID")),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("data.entitle_workflow_webhook.by_id", "name"),
					resource.TestCheckResourceAttrSet("data.entitle_workflow_webhook.by_id", "url"),
				),
			},
			// Read by name testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_workflow_webhook" "by_name" {
	name = "%s"
}
`, os.Getenv("ENTITLE_WORKFLOW_WEBHOOK_NAME")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("data.entitle_workflow_webhook.by_name", "name", os.Getenv("ENTITLE_WORKFLOW_WEBHOOK_NAME")),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("data.entitle_workflow_webhook.by_name", "id"),
					resource.TestCheckResourceAttrSet("data.entitle_workflow_webhook.by_name", "url"),
				),
			},
		},
	})
}
