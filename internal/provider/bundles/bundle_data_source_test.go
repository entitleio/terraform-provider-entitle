//go:build acceptance

package bundles_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestBundleDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_bundle" "my_bundle" {
	id = "%s"
}
`, os.Getenv("ENTITLE_BUNDLE_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("data.entitle_bundle.my_bundle", "id", os.Getenv("ENTITLE_BUNDLE_ID")),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("data.entitle_bundle.my_bundle", "name"),
					resource.TestCheckResourceAttrSet("data.entitle_bundle.my_bundle", "workflow.id"),
					resource.TestCheckResourceAttrSet("data.entitle_bundle.my_bundle", "workflow.name"),
					resource.TestCheckResourceAttrSet("data.entitle_bundle.my_bundle", "roles.0.resource.id"),
					resource.TestCheckResourceAttrSet("data.entitle_bundle.my_bundle", "roles.0.id"),
					resource.TestCheckResourceAttrSet("data.entitle_bundle.my_bundle", "roles.0.name"),
				),
			},
		},
	})
}
