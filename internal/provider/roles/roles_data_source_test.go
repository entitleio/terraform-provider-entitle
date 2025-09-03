//go:build acceptance
// +build acceptance

package roles_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestRolesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_roles" "my_list" {
	resource_id = "%s"
}
`, os.Getenv("ENTITLE_RESOURCE_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("data.entitle_roles.my_list", "roles.0.id"),
					resource.TestCheckResourceAttrSet("data.entitle_roles.my_list", "roles.0.name"),
				),
			},
		},
	})
}
