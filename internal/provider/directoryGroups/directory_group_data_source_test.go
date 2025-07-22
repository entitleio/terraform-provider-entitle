//go:build acceptance
// +build acceptance

package directoryGroups_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestAgentTokenResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_directory_group" "my_dg" {
	email = "%s"
}
`, os.Getenv("ENTITLE_DIRECTORY_GROUP_EMAIL")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("data.entitle_directory_group.my_dg", "email", os.Getenv("ENTITLE_OWNER_EMAIL")),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("data.entitle_directory_group.my_dg", "id"),
					resource.TestCheckResourceAttrSet("data.entitle_directory_group.my_dg", "name"),
					resource.TestCheckResourceAttrSet("data.entitle_directory_group.my_dg", "origin"),
				),
			},
		},
	})
}
