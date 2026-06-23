//go:build acceptance

package policies_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestPolicyResource(t *testing.T) {
	if os.Getenv("ENTITLE_DIRECTORY_GROUP_ID") == "" {
		t.SkipNow()
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
			
			resource "entitle_policy" "my_policy" {
				in_groups = [
					{
						id = "%s"
						type = "group"
					}
				]
				roles = [
					{
						id = "%s"
					}
				]
			}
			`, os.Getenv("ENTITLE_DIRECTORY_GROUP_ID"), os.Getenv("ENTITLE_ROLE_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("entitle_policy.my_policy", "in_groups.0.id", os.Getenv("ENTITLE_DIRECTORY_GROUP_ID")),
					resource.TestCheckResourceAttr("entitle_policy.my_policy", "in_groups.0.type", "group"),
					resource.TestCheckResourceAttr("entitle_policy.my_policy", "roles.0.id", os.Getenv("ENTITLE_ROLE_ID")),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("entitle_policy.my_policy", "id"),
				),
			},
		},
	})
}
