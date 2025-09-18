//go:build acceptance
// +build acceptance

package permissions_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestPermissionResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_permissions" "my_integration_permissions" {
	filter {
		integration_id = "%s"
	}
}

resource "entitle_permission" "my_integration_permission" {
	id = data.entitle_permissions.my_integration_permissions.permissions[0].id
}
`, "c5f7a1ad-bc4f-44b0-9744-e027dfa9431d"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.entitle_permissions.my_integration_permissions", "permissions.0.id"),
					resource.TestCheckResourceAttrSet("entitle_permission.my_integration_permission", "id"),
				),
			},
		},
	})
}
