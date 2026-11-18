//go:build acceptance

package integrations_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

// TestIntegrationSettingsResource runs against a real tenant. Destroy leaves
// settings in place, so the test only re-applies the integration's current
// workflow and requestability: it exercises the bulk endpoint, the batched
// list read and drift detection without changing the integration.
func TestIntegrationSettingsResource(t *testing.T) {
	integrationID := os.Getenv("ENTITLE_INTEGRATION_ID")

	config := testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_integration" "current" {
  id = "%s"
}

resource "entitle_integration_settings" "test" {
  integration_ids = [data.entitle_integration.current.id]
  workflow_id     = data.entitle_integration.current.workflow.id
  requestable     = data.entitle_integration.current.requestable
}
`, integrationID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("entitle_integration_settings.test", "integration_ids.#", "1"),
					resource.TestCheckTypeSetElemAttr("entitle_integration_settings.test", "integration_ids.*", integrationID),
					resource.TestCheckResourceAttrPair("entitle_integration_settings.test", "workflow_id",
						"data.entitle_integration.current", "workflow.id"),
				),
			},
			// Re-applying the same configuration is a no-op (refresh finds no drift).
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}
