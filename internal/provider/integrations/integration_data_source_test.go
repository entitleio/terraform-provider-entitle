//go:build acceptance
// +build acceptance

package integrations_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestIntegrationDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_integration" "my_integration" {
	id = "%s"
}
`, os.Getenv("ENTITLE_INTEGRATION_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("data.entitle_integration.my_integration", "id", os.Getenv("ENTITLE_INTEGRATION_ID")),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "name"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "requestable"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "requestable_by_default"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "application.name"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "allowed_durations.0"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "auto_assign_recommended_maintainers"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "auto_assign_recommended_owners"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "allow_creating_accounts"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "notify_about_external_permission_changes"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "readonly"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "workflow.id"),
				),
			},
		},
	})
}
