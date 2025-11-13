//go:build acceptance

package roles_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestRolesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_roles" "my_list" {
	resource_id = "%s"
}
`, os.Getenv("ENTITLE_RESOURCE_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("data.entitle_roles.my_list", "roles.0.id"),
					resource.TestCheckResourceAttrSet("data.entitle_roles.my_list", "roles.0.name"),
				),
			},
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_roles" "my_list" {
	resource_id = "%s"
	filter {
		search = "valuethathcouldnotbefound"		
	}
}
`, os.Getenv("ENTITLE_RESOURCE_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.entitle_roles.my_list", "roles.#", "0"),
				),
			},
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_roles" "my_list" {
	resource_id = "%s"
}
`, "00000000-0000-0000-0000-000000000000"),
				ExpectError: regexp.MustCompile("status code: 404"),
			},
		},
	})
}

func TestRolesDataSource_MissingResourceID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testhelpers.ProviderConfig + `
data "entitle_roles" "my_list" {
    # resource_id missing
}
`,
				ExpectError: regexp.MustCompile(`The argument "resource_id" is required`),
				PlanOnly:    true,
			},
		},
	})
}
