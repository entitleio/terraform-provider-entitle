//go:build acceptance

package bundles_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestBundleResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`

resource "entitle_bundle" "my_bundle" {
	name = "My Bundle"
	description = "Description of my bundle"
	allowed_durations = [ 1800 ]
	tags = [
		"test1", 
		"test2"
	]
	workflow = {
		id = "%s"
	}
	roles = [
		{
			id = "%s"
		}
	]
}
`, os.Getenv("ENTITLE_WORKFLOW_ID"), os.Getenv("ENTITLE_ROLE_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("entitle_bundle.my_bundle", "name", "My Bundle"),
					resource.TestCheckResourceAttr("entitle_bundle.my_bundle", "description", "Description of my bundle"),
					resource.TestCheckResourceAttr("entitle_bundle.my_bundle", "allowed_durations.0", "1800"),
					resource.TestCheckResourceAttr("entitle_bundle.my_bundle", "tags.#", "2"),
					resource.TestCheckResourceAttr("entitle_bundle.my_bundle", "workflow.id", os.Getenv("ENTITLE_WORKFLOW_ID")),
					resource.TestCheckResourceAttr("entitle_bundle.my_bundle", "roles.0.id", os.Getenv("ENTITLE_ROLE_ID")),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("entitle_bundle.my_bundle", "id"),
					resource.TestCheckResourceAttrSet("entitle_bundle.my_bundle", "workflow.name"),
					resource.TestCheckResourceAttrSet("entitle_bundle.my_bundle", "roles.0.name"),
					resource.TestCheckResourceAttrSet("entitle_bundle.my_bundle", "roles.0.resource.id"),
				),
			},
			// Update testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`

resource "entitle_bundle" "my_bundle" {
	name = "My Bundle UPDATED"
	description = "Description of my bundle"
	allowed_durations = [ 1800 ]
	tags = [
		"test1", 
		"test2"
	]
	workflow = {
		id = "%s"
	}
	roles = [
		{
			id = "%s"
		}
	]
}
`, os.Getenv("ENTITLE_WORKFLOW_ID"), os.Getenv("ENTITLE_ROLE_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("entitle_bundle.my_bundle", "name", "My Bundle UPDATED"),
				),
			},
		},
	})
}
