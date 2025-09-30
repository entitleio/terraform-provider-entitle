//go:build acceptance
// +build acceptance

package policies_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestPolicyDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testhelpers.ProviderConfig2 + fmt.Sprintf(`
data "entitle_policy" "my_policy" {
	id = "%s"
}
`, os.Getenv("ENTITLE_POLICY_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("data.entitle_policy.my_policy", "id", os.Getenv("ENTITLE_POLICY_ID")),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("data.entitle_policy.my_policy", "number"),
					resource.TestCheckResourceAttrSet("data.entitle_policy.my_policy", "sort_order"),
				),
			},
		},
	})
}
