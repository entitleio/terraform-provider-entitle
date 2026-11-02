//go:build acceptance

package integrations_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestIntegrationsDataSource(t *testing.T) {
	integrationID := os.Getenv("ENTITLE_INTEGRATION_ID")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// First page of all integrations.
			{
				Config: testhelpers.ProviderConfig + `
data "entitle_integrations" "all" {
  filter {
    per_page = 10
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.entitle_integrations.all", "integrations.0.id"),
					resource.TestCheckResourceAttrSet("data.entitle_integrations.all", "integrations.0.name"),
					resource.TestCheckResourceAttrSet("data.entitle_integrations.all", "integrations.0.application.name"),
				),
			},
			// A single integration by ID.
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_integrations" "by_id" {
  filter {
    ids = ["%s"]
  }
}
`, integrationID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.entitle_integrations.by_id", "integrations.#", "1"),
					resource.TestCheckResourceAttr("data.entitle_integrations.by_id", "integrations.0.id", integrationID),
				),
			},
		},
	})
}
