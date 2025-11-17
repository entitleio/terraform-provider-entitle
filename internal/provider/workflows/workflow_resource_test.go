//go:build acceptance

package workflows_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestWorkflowResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`

resource "entitle_workflow" "my_workflow" {
	name = "My Workflow CI"
	rules = [
		{
			sort_order = 1
			approval_flow = {
				steps = [
					{
						sort_order = 1
						notified_entities = [
							{
								type = "IntegrationOwner"
							}
						]
						approval_entities = [
							{
								type = "IntegrationOwner"
							},
							{
								type = "ResourceOwner"
							}
						]
					},
					{
						sort_order = 1
						approval_entities = [
							{
								type = "User"
								user = {
									id = "%s"
								}
							}
						]
					}
				]
			}
		}
	]
}
`, os.Getenv("ENTITLE_OWNER_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "name", "My Workflow CI"),
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.sort_order", "1"),
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.approval_flow.steps.0.sort_order", "1"),
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.approval_flow.steps.0.approval_entities.0.type", "IntegrationOwner"),
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.approval_flow.steps.0.approval_entities.1.type", "ResourceOwner"),
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.approval_flow.steps.1.approval_entities.0.type", "User"),
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.approval_flow.steps.1.approval_entities.0.user.id", os.Getenv("ENTITLE_OWNER_ID")),

					// Verify default values
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.approval_flow.steps.0.operator", "and"),
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.any_schedule", "true"),
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.under_duration", "3600"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("entitle_workflow.my_workflow", "id"),
					resource.TestCheckResourceAttrSet("entitle_workflow.my_workflow", "rules.0.approval_flow.steps.1.approval_entities.0.user.email"),
				),
			},
			// Update testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`

resource "entitle_workflow" "my_workflow" {
	name = "My Workflow CI UPDATED"
	rules = [
		{
			sort_order = 1
			approval_flow = {
				steps = [
					{
						sort_order = 1
						notified_entities = [
							{
								type = "IntegrationOwner"
								value = {
								  notified = null
								}
							}
						]
						approval_entities = [
							{
								type = "ResourceOwner"
							},
							{
								type = "IntegrationOwner"
							}
						]
					},
					{
						sort_order = 1
						approval_entities = [
							{
								type = "User"
								user = {
									id = "%s"
								}
							}
						]
					}
				]
			}
		}
	]
}
`, os.Getenv("ENTITLE_OWNER_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "name", "My Workflow CI UPDATED"),
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.sort_order", "1"),
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.approval_flow.steps.0.sort_order", "1"),
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.approval_flow.steps.0.approval_entities.0.type", "IntegrationOwner"),
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.approval_flow.steps.0.approval_entities.1.type", "ResourceOwner"),
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.approval_flow.steps.1.approval_entities.0.type", "User"),
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.approval_flow.steps.1.approval_entities.0.user.id", os.Getenv("ENTITLE_OWNER_ID")),

					// Verify default values
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.approval_flow.steps.0.operator", "and"),
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.any_schedule", "true"),
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.under_duration", "3600"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("entitle_workflow.my_workflow", "id"),
					resource.TestCheckResourceAttrSet("entitle_workflow.my_workflow", "rules.0.approval_flow.steps.1.approval_entities.0.user.email"),
				),
			},
		},
	})
}
