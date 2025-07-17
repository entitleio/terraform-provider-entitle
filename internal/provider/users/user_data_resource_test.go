//go:build acceptance
// +build acceptance

package users_test

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
data "entitle_user" "my_user" {
	email = "%s"
}
`, os.Getenv("ENTITLE_OWNER_EMAIL")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("data.entitle_user.my_user", "email", os.Getenv("ENTITLE_OWNER_EMAIL")),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("data.entitle_user.my_user", "id"),
					resource.TestCheckResourceAttrSet("data.entitle_user.my_user", "given_name"),
					resource.TestCheckResourceAttrSet("data.entitle_user.my_user", "family_name"),
					resource.TestCheckResourceAttrSet("data.entitle_user.my_user", "created_at"),
				),
			},
		},
	})
}
