//go:build acceptance

package bundles_test

import (
	"fmt"
	"os"
	"regexp"
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

func TestBundleDataSourceByName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_bundle" "my_bundle" {
	name = "%s"
}
`, os.Getenv("ENTITLE_BUNDLE_NAME")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("data.entitle_bundle.my_bundle", "name", os.Getenv("ENTITLE_BUNDLE_NAME")),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("data.entitle_bundle.my_bundle", "id"),
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

func TestBundleDataSourceFail(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// ID and Name provided
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_bundle" "my_bundle" {
	id = "%s"
	name = "%s"
}
`, os.Getenv("ENTITLE_BUNDLE_ID"), os.Getenv("ENTITLE_BUNDLE_NAME")),
				ExpectError: regexp.MustCompile("Invalid Attribute Combination"),
			},
			// Name provided, but not found
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_bundle" "my_bundle" {
	name = "%sBLABLA"
}
`, os.Getenv("ENTITLE_BUNDLE_NAME")),
				ExpectError: regexp.MustCompile("Failed to get the Bundle by the name"),
			},
			// None of ID and Name provided
			{
				Config: testhelpers.ProviderConfig + `
data "entitle_bundle" "my_bundle" {}
`,
				ExpectError: regexp.MustCompile("Invalid Attribute Combination"),
			},
		},
	})
}
