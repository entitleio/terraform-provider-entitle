//go:build acceptance

package roles_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestRoleSyncedResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
resource "entitle_role_synced" "my_role" {
	name = "%s"
	resource = {
		id = "%s"
	}
}
`, os.Getenv("ENTITLE_ROLE_SYNCED_NAME"), os.Getenv("ENTITLE_RESOURCE_SYNCED_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("entitle_role_synced.my_role", "name", os.Getenv("ENTITLE_ROLE_SYNCED_NAME")),
					resource.TestCheckResourceAttr("entitle_role_synced.my_role", "resource.id", os.Getenv("ENTITLE_RESOURCE_SYNCED_ID")),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("entitle_role_synced.my_role", "id"),
					resource.TestCheckResourceAttrSet("entitle_role_synced.my_role", "requestable"),
					resource.TestCheckResourceAttrSet("entitle_role_synced.my_role", "resource.name"),
				),
			},
			// Update testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`

resource "entitle_role_synced" "my_role" {
	name = "%s"
	resource = {
		id = "%s"
	}
	requestable = false
}
`, os.Getenv("ENTITLE_ROLE_SYNCED_NAME"), os.Getenv("ENTITLE_RESOURCE_SYNCED_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("entitle_role_synced.my_role", "name", os.Getenv("ENTITLE_ROLE_SYNCED_NAME")),
					resource.TestCheckResourceAttr("entitle_role_synced.my_role", "resource.id", os.Getenv("ENTITLE_RESOURCE_SYNCED_ID")),
					resource.TestCheckResourceAttr("entitle_role_synced.my_role", "requestable", "false"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("entitle_role_synced.my_role", "id"),
					resource.TestCheckResourceAttrSet("entitle_role_synced.my_role", "resource.name"),
				),
			},
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
resource "entitle_role_synced" "my_role" {
	name = "%s"
	resource = {
		id = "%s"
	}
	requestable = true
}
`, os.Getenv("ENTITLE_ROLE_SYNCED_NAME"), os.Getenv("ENTITLE_RESOURCE_SYNCED_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("entitle_role_synced.my_role", "name", os.Getenv("ENTITLE_ROLE_SYNCED_NAME")),
					resource.TestCheckResourceAttr("entitle_role_synced.my_role", "resource.id", os.Getenv("ENTITLE_RESOURCE_SYNCED_ID")),
					resource.TestCheckResourceAttr("entitle_role_synced.my_role", "requestable", "true"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("entitle_role_synced.my_role", "id"),
					resource.TestCheckResourceAttrSet("entitle_role_synced.my_role", "resource.name"),
				),
			},
		},
	})
}
