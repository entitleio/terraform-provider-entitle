//go:build acceptance

package integrations_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestIntegrationsDataSource(t *testing.T) {
	integrationID := os.Getenv("ENTITLE_INTEGRATION_ID")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Every integration.
			{
				Config: testhelpers.ProviderConfig + `
data "entitle_integrations" "all" {}
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
			// A name that matches nothing.
			{
				Config: testhelpers.ProviderConfig + `
data "entitle_integrations" "none" {
  filter {
    names = ["valuethathcouldnotbefound"]
  }
}
`,
				Check: resource.TestCheckResourceAttr("data.entitle_integrations.none", "integrations.#", "0"),
			},
		},
	})
}

func TestIntegrationsDataSource_NameWithComma(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testhelpers.ProviderConfig + `
data "entitle_integrations" "x" {
  filter {
    names = ["a,b"]
  }
}
`,
				ExpectError: regexp.MustCompile(`must\s+not\s+contain\s+a\s+comma`),
				PlanOnly:    true,
			},
		},
	})
}
