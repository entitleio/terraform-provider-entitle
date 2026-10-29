//go:build acceptance

package resources_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
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
					resource.TestCheckResourceAttrSet("data.entitle_resources.my_resource", "resources.0.external_id"),
				),
			},
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_resources" "my_resource" {
	integration_id = "%s"
}
`, "00000000-0000-0000-0000-000000000000"),
				ExpectError: regexp.MustCompile(utils.ErrNotFound.Error()),
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
				ExpectError: regexp.MustCompile(`Either\s+integration_id\s+or\s+filter.ids\s+must\s+be\s+set`),
				PlanOnly:    true,
			},
		},
	})
}

func TestResourcesDataSource_IDsFilter(t *testing.T) {
	resourceID := os.Getenv("ENTITLE_RESOURCE_ID")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// integration_id is not needed when ids is set.
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_resources" "by_id" {
  filter {
    ids = ["%s"]
  }
}
`, resourceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.entitle_resources.by_id", "resources.#", "1"),
					resource.TestCheckResourceAttr("data.entitle_resources.by_id", "resources.0.id", resourceID),
				),
			},
		},
	})
}
