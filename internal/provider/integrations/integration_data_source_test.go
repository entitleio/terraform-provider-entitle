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

func TestIntegrationDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_integration" "my_integration" {
	id = "%s"
}
`, os.Getenv("ENTITLE_INTEGRATION_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("data.entitle_integration.my_integration", "id", os.Getenv("ENTITLE_INTEGRATION_ID")),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "name"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "requestable"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "requestable_by_default"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "application.name"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "allowed_durations.0"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "auto_assign_recommended_maintainers"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "auto_assign_recommended_owners"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "allow_creating_accounts"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "notify_about_external_permission_changes"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "readonly"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "workflow.id"),
				),
			},
		},
	})
}

func TestIntegrationDataSourceByName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_integration" "my_integration" {
	name = "%s"
}
`, os.Getenv("ENTITLE_INTEGRATION_NAME")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("data.entitle_integration.my_integration", "name", os.Getenv("ENTITLE_INTEGRATION_NAME")),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "id"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "requestable"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "requestable_by_default"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "application.name"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "allowed_durations.0"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "auto_assign_recommended_maintainers"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "auto_assign_recommended_owners"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "allow_creating_accounts"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "notify_about_external_permission_changes"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "readonly"),
					resource.TestCheckResourceAttrSet("data.entitle_integration.my_integration", "workflow.id"),
				),
			},
		},
	})
}

func TestIntegrationDataSourceFail(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// ID and Name provided
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_integration" "my_integration" {
	id = "%s"
	name = "%s"
}
`, os.Getenv("ENTITLE_INTEGRATION_ID"), os.Getenv("ENTITLE_INTEGRATION_NAME")),
				ExpectError: regexp.MustCompile("Invalid Attribute Combination"),
			},
			// Name provided, but not found
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_integration" "my_integration" {
	name = "%sBLABLA"
}
`, os.Getenv("ENTITLE_INTEGRATION_NAME")),
				ExpectError: regexp.MustCompile("Failed to get the Integration by the name"),
			},
			// None of ID and Name provided
			{
				Config: testhelpers.ProviderConfig + `
data "entitle_integration" "my_integration" {}
`,
				ExpectError: regexp.MustCompile("Invalid Attribute Combination"),
			},
		},
	})
}
