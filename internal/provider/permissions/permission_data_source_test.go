//go:build acceptance
// +build acceptance

package permissions_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestPermissionDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_permission" "my_permission" {
	id = "%s"
}
`, os.Getenv("ENTITLE_PERMISSION_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("data.entitle_permission.my_permission", "id", os.Getenv("ENTITLE_PERMISSION_ID")),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("data.entitle_permission.my_permission", "types.0"),
					resource.TestCheckResourceAttrSet("data.entitle_permission.my_permission", "path"),
					resource.TestCheckResourceAttrSet("data.entitle_permission.my_permission", "actor.id"),
					resource.TestCheckResourceAttrSet("data.entitle_permission.my_permission", "role.id"),
					resource.TestCheckResourceAttrSet("data.entitle_permission.my_permission", "role.name"),
					resource.TestCheckResourceAttrSet("data.entitle_permission.my_permission", "role.resource.id"),
					resource.TestCheckResourceAttrSet("data.entitle_permission.my_permission", "role.resource.name"),
					resource.TestCheckResourceAttrSet("data.entitle_permission.my_permission", "role.resource.integration.id"),
					resource.TestCheckResourceAttrSet("data.entitle_permission.my_permission", "role.resource.integration.name"),
					resource.TestCheckResourceAttrSet("data.entitle_permission.my_permission", "role.resource.integration.application.name"),
				),
			},
		},
	})
}
