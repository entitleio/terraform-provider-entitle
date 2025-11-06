//go:build acceptance

package directoryGroups_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestDirectoryGroupsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testhelpers.ProviderConfig2 + fmt.Sprintf(`
data "entitle_directory_groups" "my_list" {
	filter {
		search = "%s"
	}
}
`, os.Getenv("ENTITLE_DIRECTORY_GROUP_EMAIL")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("data.entitle_directory_groups.my_list", "directory_groups.0.id"),
					resource.TestCheckResourceAttrSet("data.entitle_directory_groups.my_list", "directory_groups.0.name"),
					resource.TestCheckResourceAttrSet("data.entitle_directory_groups.my_list", "directory_groups.0.origin"),
					resource.TestCheckResourceAttrSet("data.entitle_directory_groups.my_list", "directory_groups.0.email"),
				),
			},
			{
				Config: testhelpers.ProviderConfig2 + fmt.Sprintf(`
data "entitle_directory_groups" "my_list" {
	filter {
		search = "%s"
	}
}
`, "nothingthatcouldbefound"),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttr("data.entitle_directory_groups.my_list", "directory_groups.#", "0"),
				),
			},
		},
	})
}
