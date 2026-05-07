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
								type = "ResourceOwner"
							},
							{
								type = "IntegrationOwner"
							}
						]
					},
					{
						sort_order = 2
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
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.approval_flow.steps.0.approval_entities.0.type", "ResourceOwner"),
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.approval_flow.steps.0.approval_entities.1.type", "IntegrationOwner"),
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
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.approval_flow.steps.0.approval_entities.0.type", "ResourceOwner"),
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.approval_flow.steps.0.approval_entities.1.type", "IntegrationOwner"),
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

func TestWorkflowResourceWithWebhook(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create workflow with webhook as approval entity
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`

resource "entitle_workflow" "webhook_workflow" {
	name = "Webhook Workflow CI"
	rules = [
		{
			sort_order = 1
			approval_flow = {
				steps = [
					{
						sort_order = 1
						approval_entities = [
							{
								type = "Webhook"
								webhook = {
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
`, os.Getenv("ENTITLE_WEBHOOK_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("entitle_workflow.webhook_workflow", "name", "Webhook Workflow CI"),
					resource.TestCheckResourceAttr("entitle_workflow.webhook_workflow", "rules.0.sort_order", "1"),
					resource.TestCheckResourceAttr("entitle_workflow.webhook_workflow", "rules.0.approval_flow.steps.0.approval_entities.0.type", "Webhook"),
					resource.TestCheckResourceAttr("entitle_workflow.webhook_workflow", "rules.0.approval_flow.steps.0.approval_entities.0.webhook.id", os.Getenv("ENTITLE_WEBHOOK_ID")),
					resource.TestCheckResourceAttrSet("entitle_workflow.webhook_workflow", "id"),
					resource.TestCheckResourceAttrSet("entitle_workflow.webhook_workflow", "rules.0.approval_flow.steps.0.approval_entities.0.webhook.name"),
				),
			},
			// Update: add notified entity with webhook
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`

resource "entitle_workflow" "webhook_workflow" {
	name = "Webhook Workflow CI Updated"
	rules = [
		{
			sort_order = 1
			approval_flow = {
				steps = [
					{
						sort_order = 1
						notified_entities = [
							{
								type = "Webhook"
								webhook = {
									id = "%s"
								}
							}
						]
						approval_entities = [
							{
								type = "Webhook"
								webhook = {
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
`, os.Getenv("ENTITLE_WEBHOOK_ID"), os.Getenv("ENTITLE_WEBHOOK_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("entitle_workflow.webhook_workflow", "name", "Webhook Workflow CI Updated"),
					resource.TestCheckResourceAttr("entitle_workflow.webhook_workflow", "rules.0.approval_flow.steps.0.notified_entities.0.type", "Webhook"),
					resource.TestCheckResourceAttr("entitle_workflow.webhook_workflow", "rules.0.approval_flow.steps.0.notified_entities.0.webhook.id", os.Getenv("ENTITLE_WEBHOOK_ID")),
					resource.TestCheckResourceAttr("entitle_workflow.webhook_workflow", "rules.0.approval_flow.steps.0.approval_entities.0.type", "Webhook"),
					resource.TestCheckResourceAttr("entitle_workflow.webhook_workflow", "rules.0.approval_flow.steps.0.approval_entities.0.webhook.id", os.Getenv("ENTITLE_WEBHOOK_ID")),
				),
			},
		},
	})
}

func TestWorkflowResourceWithMultipleWebhooks(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`

resource "entitle_workflow" "multi_webhook_workflow" {
	name = "Multi Webhook Workflow CI"
	rules = [
		{
			sort_order = 1
			approval_flow = {
				steps = [
					{
						sort_order = 1
						operator = "and"
						approval_entities = [
							{
								type = "Webhook"
								webhook = {
									id = "%s"
								}
							},
							{
								type = "Webhook"
								webhook = {
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
`, os.Getenv("ENTITLE_WEBHOOK_ID"), os.Getenv("ENTITLE_WEBHOOK_ID_2")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("entitle_workflow.multi_webhook_workflow", "name", "Multi Webhook Workflow CI"),
					resource.TestCheckResourceAttr("entitle_workflow.multi_webhook_workflow", "rules.0.approval_flow.steps.0.operator", "and"),
					resource.TestCheckResourceAttr("entitle_workflow.multi_webhook_workflow", "rules.0.approval_flow.steps.0.approval_entities.#", "2"),
					resource.TestCheckResourceAttr("entitle_workflow.multi_webhook_workflow", "rules.0.approval_flow.steps.0.approval_entities.0.type", "Webhook"),
					resource.TestCheckResourceAttr("entitle_workflow.multi_webhook_workflow", "rules.0.approval_flow.steps.0.approval_entities.0.webhook.id", os.Getenv("ENTITLE_WEBHOOK_ID")),
					resource.TestCheckResourceAttr("entitle_workflow.multi_webhook_workflow", "rules.0.approval_flow.steps.0.approval_entities.1.type", "Webhook"),
					resource.TestCheckResourceAttr("entitle_workflow.multi_webhook_workflow", "rules.0.approval_flow.steps.0.approval_entities.1.webhook.id", os.Getenv("ENTITLE_WEBHOOK_ID_2")),
					resource.TestCheckResourceAttrSet("entitle_workflow.multi_webhook_workflow", "id"),
				),
			},
		},
	})
}

func TestWorkflowResource_Import(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				ResourceName:      "entitle_workflow.my_workflow",
				ImportState:       true,
				ImportStateId:     os.Getenv("ENTITLE_WORKFLOW_ID"),
				ImportStateVerify: false,
				// Config is not applied; it is only needed so the test framework
				// can initialise the working directory with the correct resource type.
				Config: testhelpers.ProviderConfig + `
resource "entitle_workflow" "my_workflow" {
  name = "placeholder"
}
`,
			},
		},
	})
}

func TestWorkflowResourceChange(t *testing.T) {
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
						approval_entities = [
							{
								type = "Automatic"
							}
						]
						sort_order = 1
					}
				]
			}
		}
	]
}
`),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "name", "My Workflow CI"),
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.sort_order", "1"),
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.approval_flow.steps.0.sort_order", "1"),
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.approval_flow.steps.0.approval_entities.0.type", "Automatic"),

					// Verify default values
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.approval_flow.steps.0.operator", "and"),
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.any_schedule", "true"),
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.under_duration", "3600"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("entitle_workflow.my_workflow", "id"),
				),
			},
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
						approval_entities = [
							{
								type = "User"
								user = {
									id = "%s"
								}
							}
						]
						sort_order = 1
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
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.approval_flow.steps.0.approval_entities.0.type", "User"),

					// Verify default values
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.approval_flow.steps.0.operator", "and"),
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.any_schedule", "true"),
					resource.TestCheckResourceAttr("entitle_workflow.my_workflow", "rules.0.under_duration", "3600"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("entitle_workflow.my_workflow", "id"),
				),
			},
		},
	})
}
