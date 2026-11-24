//go:build acceptance

package roles_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

// TestRoleSettingsResource runs against a real tenant. Destroy leaves settings
// in place, so the test only re-applies the role's current requestability and
// workflow: it exercises the bulk endpoint, the batched list read and drift
// detection without changing the role. (A role without its own workflow yields
// a null workflow_id, which is simply not managed.)
func TestRoleSettingsResource(t *testing.T) {
	roleID := os.Getenv("ENTITLE_ROLE_ID")

	config := testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_role" "current" {
  id = "%s"
}

resource "entitle_role_settings" "test" {
  role_ids    = [data.entitle_role.current.id]
  requestable = data.entitle_role.current.requestable
  workflow_id = try(data.entitle_role.current.workflow.id, null)
}
`, roleID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("entitle_role_settings.test", "role_ids.#", "1"),
					resource.TestCheckTypeSetElemAttr("entitle_role_settings.test", "role_ids.*", roleID),
					resource.TestCheckResourceAttrPair("entitle_role_settings.test", "requestable",
						"data.entitle_role.current", "requestable"),
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
