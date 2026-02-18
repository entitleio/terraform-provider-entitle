//go:build acceptance

package workflows_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestWorkflowDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_workflow" "my_workflow" {
	id = "%s"
}
`, os.Getenv("ENTITLE_WORKFLOW_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("data.entitle_workflow.my_workflow", "id", os.Getenv("ENTITLE_WORKFLOW_ID")),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("data.entitle_workflow.my_workflow", "name"),
					resource.TestCheckResourceAttrSet("data.entitle_workflow.my_workflow", "rules.#"),
					resource.TestCheckResourceAttrSet("data.entitle_workflow.my_workflow", "rules.0.sort_order"),
				),
			},
		},
	})
}

func TestWorkflowDataSourceByName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_workflow" "my_workflow" {
	name = "%s"
}
`, os.Getenv("ENTITLE_WORKFLOW_NAME")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("data.entitle_workflow.my_workflow", "name", os.Getenv("ENTITLE_WORKFLOW_NAME")),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("data.entitle_workflow.my_workflow", "id"),
					resource.TestCheckResourceAttrSet("data.entitle_workflow.my_workflow", "rules.#"),
					resource.TestCheckResourceAttrSet("data.entitle_workflow.my_workflow", "rules.0.sort_order"),
				),
			},
		},
	})
}

func TestWorkflowDataSourceFail(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// ID and Name provided
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_workflow" "my_workflow" {
	id = "%s"
	name = "%s"
}
`, os.Getenv("ENTITLE_WORKFLOW_ID"), os.Getenv("ENTITLE_WORKFLOW_NAME")),
				ExpectError: regexp.MustCompile("Invalid Attribute Combination"),
			},
			// Name provided, but not found
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_workflow" "my_workflow" {
	name = "%sBLABLA"
}
`, os.Getenv("ENTITLE_WORKFLOW_NAME")),
				ExpectError: regexp.MustCompile("Failed to get the Workflow by the name"),
			},
			// None of ID and Name provided
			{
				Config: testhelpers.ProviderConfig + `
data "entitle_workflow" "my_workflow" {}
`,
				ExpectError: regexp.MustCompile("Invalid Attribute Combination"),
			},
		},
	})
}
