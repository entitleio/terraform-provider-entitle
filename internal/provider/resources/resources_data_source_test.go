//go:build acceptance

package resources_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestResourcesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_resources" "my_resource" {
	integration_id = "%s"
}
`, os.Getenv("ENTITLE_INTEGRATION_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("data.entitle_resources.my_resource", "resources.0.id"),
					resource.TestCheckResourceAttrSet("data.entitle_resources.my_resource", "resources.0.name"),
					resource.TestCheckResourceAttrSet("data.entitle_resources.my_resource", "resources.0.integration.id"),
				),
			},
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_resources" "my_resource" {
	integration_id = "%s"
}
`, "00000000-0000-0000-0000-000000000000"),
				ExpectError: regexp.MustCompile("status code: 404"),
			},
		},
	})
}

func TestResourcesDataSource_MissingIntegrationID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testhelpers.ProviderConfig + `
data "entitle_resources" "my_list" {
    # integration_id missing
}
`,
				ExpectError: regexp.MustCompile(`The argument "integration_id" is required`),
				PlanOnly:    true,
			},
		},
	})
}
