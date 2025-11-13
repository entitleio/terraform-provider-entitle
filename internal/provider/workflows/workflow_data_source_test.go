//go:build acceptance

package workflows_test

import (
	"fmt"
	"os"
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
