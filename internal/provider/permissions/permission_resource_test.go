//go:build acceptance
// +build acceptance

package permissions_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestPermissionResource(t *testing.T) {
	resourceName := "entitle_permission.test"
	permissionID := os.Getenv("ENTITLE_PERMISSION_ID")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Import the existing permission
			{
				ResourceName:  resourceName,
				ImportState:   true,
				ImportStateId: permissionID,
				Config: testhelpers.ProviderConfig + `
resource "entitle_permission" "test" {
  id = "` + permissionID + `"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", permissionID),
					resource.TestCheckResourceAttrSet(resourceName, "actor.id"),
					resource.TestCheckResourceAttrSet(resourceName, "role.id"),
					resource.TestCheckResourceAttrSet(resourceName, "path"),
				),
			},
		},
	})
}
