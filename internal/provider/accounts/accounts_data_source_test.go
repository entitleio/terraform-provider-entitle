//go:build acceptance
// +build acceptance

package accounts_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestAccountsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_accounts" "my_list" {
	integration_id = "%s"
}
`, os.Getenv("ENTITLE_INTEGRATION_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("data.entitle_accounts.my_list", "accounts.0.id"),
					resource.TestCheckResourceAttrSet("data.entitle_accounts.my_list", "accounts.0.name"),
					resource.TestCheckResourceAttrSet("data.entitle_accounts.my_list", "accounts.0.euid"),
					resource.TestCheckResourceAttrSet("data.entitle_accounts.my_list", "accounts.0.integration.id"),
					resource.TestCheckResourceAttrSet("data.entitle_accounts.my_list", "accounts.0.integration.application.name"),
				),
			},
		},
	})
}
