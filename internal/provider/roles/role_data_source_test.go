//go:build acceptance

package roles_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestRoleDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_role" "my_role" {
	id = "%s"
}
`, os.Getenv("ENTITLE_ROLE_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("data.entitle_role.my_role", "id", os.Getenv("ENTITLE_ROLE_ID")),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("data.entitle_role.my_role", "name"),
					resource.TestCheckResourceAttrSet("data.entitle_role.my_role", "resource.id"),
					resource.TestCheckResourceAttrSet("data.entitle_role.my_role", "requestable"),
				),
			},
		},
	})
}
