//go:build acceptance

package users_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestUsersDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_users" "my_list" {
	filter {
		search = "%s"
	}
}
`, os.Getenv("ENTITLE_OWNER_EMAIL")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("data.entitle_users.my_list", "users.0.email", os.Getenv("ENTITLE_OWNER_EMAIL")),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("data.entitle_users.my_list", "users.0.id"),
					resource.TestCheckResourceAttrSet("data.entitle_users.my_list", "users.0.given_name"),
					resource.TestCheckResourceAttrSet("data.entitle_users.my_list", "users.0.family_name"),
					resource.TestCheckResourceAttrSet("data.entitle_users.my_list", "users.0.created_at"),
				),
			},
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_users" "my_list" {
	filter {
		search = "%s"
	}
}
`, "valuewhichcouldnotbefound"),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("data.entitle_users.my_list", "users.#", "0"),
				),
			},
			{
				Config: testhelpers.ProviderConfig + `
data "entitle_users" "my_list" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("data.entitle_users.my_list", "users.0.email"),
					resource.TestCheckResourceAttrSet("data.entitle_users.my_list", "users.0.id"),
					resource.TestCheckResourceAttrSet("data.entitle_users.my_list", "users.0.given_name"),
					resource.TestCheckResourceAttrSet("data.entitle_users.my_list", "users.0.family_name"),
					resource.TestCheckResourceAttrSet("data.entitle_users.my_list", "users.0.created_at"),
				),
			},
		},
	})
}
