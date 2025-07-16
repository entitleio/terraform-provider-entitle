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

func TestRoleResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`

resource "entitle_role" "my_role" {
	name = "My Role Example"
	resource = {
		id = "%s"
	}
	workflow = {
		id = "%s"
	}
	requestable = true
	allowed_durations = [-1]
}
`, os.Getenv("ENTITLE_RESOURCE_ID"), os.Getenv("ENTITLE_WORKFLOW_ID"), os.Getenv("ENTITLE_ROLE_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("entitle_role.my_role", "name", "My Role Example"),
					resource.TestCheckResourceAttr("entitle_role.my_role", "workflow.id", os.Getenv("ENTITLE_WORKFLOW_ID")),
					resource.TestCheckResourceAttr("entitle_role.my_role", "resource.id", os.Getenv("ENTITLE_RESOURCE_ID")),
					resource.TestCheckResourceAttr("entitle_role.my_role", "requestable", "true"),
					resource.TestCheckResourceAttr("entitle_role.my_role", "allowed_durations.0", "-1"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("entitle_role.my_role", "id"),
					resource.TestCheckResourceAttrSet("entitle_role.my_role", "workflow.name"),
					resource.TestCheckResourceAttrSet("entitle_role.my_role", "resource.name"),
				),
			},
			// Update testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`

resource "entitle_role" "my_role" {
	name = "My Role Example"
	resource = {
		id = "%s"
	}
	workflow = {
		id = "%s"
	}
	requestable = false
	allowed_durations = [3600]
}
`, os.Getenv("ENTITLE_RESOURCE_ID"), os.Getenv("ENTITLE_WORKFLOW_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("entitle_role.my_role", "name", "My Role Example"),
					resource.TestCheckResourceAttr("entitle_role.my_role", "workflow.id", os.Getenv("ENTITLE_WORKFLOW_ID")),
					resource.TestCheckResourceAttr("entitle_role.my_role", "resource.id", os.Getenv("ENTITLE_RESOURCE_ID")),
					resource.TestCheckResourceAttr("entitle_role.my_role", "requestable", "false"),
					resource.TestCheckResourceAttr("entitle_role.my_role", "allowed_durations.0", "3600"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("entitle_role.my_role", "id"),
					resource.TestCheckResourceAttrSet("entitle_role.my_role", "workflow.name"),
					resource.TestCheckResourceAttrSet("entitle_role.my_role", "resource.name"),
				),
			},
		},
	})
}
