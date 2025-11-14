//go:build acceptance

package integrations_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestIntegrationResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`

resource "entitle_integration" "my_gitlab" {
  	name                               = "My Gitlab Integration"
    requestable                     = true
    requestable_by_default                     = true
    application = {
   	  name = "gitlab"
    }
	allowed_durations = [-1]
    auto_assign_recommended_maintainers      = false
    auto_assign_recommended_owners           = false
    allow_creating_accounts           = false
    connection_json                          = jsonencode({
		domain                   = "https://gitlab.com"
		private_token            = "%s"
		configurationSchemaName = "Configuration "
	  })
    notify_about_external_permission_changes = true
    owner = {
      id    = "%s"
    }
    readonly = false
    workflow = {
      id   = "%s"
    }
	maintainers = [
		{
			type = "user"
			entity = {
				id = "%s"
			}
		}
	]
	prerequisite_permissions = [
		{
			default = true
			role = {
				id = "%s"
			}
		}
	]
}
`, os.Getenv("GITLAB_ACCESS_TOKEN"), os.Getenv("ENTITLE_OWNER_ID"), os.Getenv("ENTITLE_WORKFLOW_ID"), os.Getenv("ENTITLE_OWNER_ID"), os.Getenv("ENTITLE_ROLE_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("entitle_integration.my_gitlab", "name", "My Gitlab Integration"),
					resource.TestCheckResourceAttr("entitle_integration.my_gitlab", "requestable", "true"),
					resource.TestCheckResourceAttr("entitle_integration.my_gitlab", "requestable_by_default", "true"),
					resource.TestCheckResourceAttr("entitle_integration.my_gitlab", "application.name", "gitlab"),
					resource.TestCheckResourceAttr("entitle_integration.my_gitlab", "allowed_durations.0", "-1"),
					resource.TestCheckResourceAttr("entitle_integration.my_gitlab", "auto_assign_recommended_maintainers", "false"),
					resource.TestCheckResourceAttr("entitle_integration.my_gitlab", "auto_assign_recommended_owners", "false"),
					resource.TestCheckResourceAttr("entitle_integration.my_gitlab", "allow_creating_accounts", "false"),
					resource.TestCheckResourceAttr("entitle_integration.my_gitlab", "notify_about_external_permission_changes", "true"),
					resource.TestCheckResourceAttr("entitle_integration.my_gitlab", "owner.id", os.Getenv("ENTITLE_OWNER_ID")),
					resource.TestCheckResourceAttr("entitle_integration.my_gitlab", "readonly", "false"),
					resource.TestCheckResourceAttr("entitle_integration.my_gitlab", "workflow.id", os.Getenv("ENTITLE_WORKFLOW_ID")),
					resource.TestCheckResourceAttr("entitle_integration.my_gitlab", "prerequisite_permissions.0.role.id", os.Getenv("ENTITLE_ROLE_ID")),
					resource.TestCheckResourceAttrSet("entitle_integration.my_gitlab", "connection_json"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("entitle_integration.my_gitlab", "id"),

					resource.TestCheckResourceAttrSet("entitle_integration.my_gitlab", "prerequisite_permissions.0.role.name"),
					resource.TestCheckResourceAttrSet("entitle_integration.my_gitlab", "prerequisite_permissions.0.role.resource.name"),
					resource.TestCheckResourceAttrSet("entitle_integration.my_gitlab", "prerequisite_permissions.0.role.resource.integration.name"),
					resource.TestCheckResourceAttrSet("entitle_integration.my_gitlab", "prerequisite_permissions.0.role.resource.integration.application.name"),
				),
			},
			// Update testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`

resource "entitle_integration" "my_gitlab" {
  	name                               = "My Gitlab Integration UPDATED"
    requestable                     = true
    requestable_by_default                     = true
    application = {
   	  name = "gitlab"
    }
	allowed_durations = [-1]
    auto_assign_recommended_maintainers      = false
    auto_assign_recommended_owners           = false
    allow_creating_accounts           = false
    connection_json                          = jsonencode({
		domain                   = "https://gitlab.com"
		private_token            = "%s"
		configurationSchemaName = "Configuration "
	  })
    notify_about_external_permission_changes = true
    owner = {
      id    = "%s"
    }
    readonly = false
    workflow = {
      id   = "%s"
    }
	prerequisite_permissions = [
		{
			default = true
			role = {
				id = "%s"
			}
		}
	]
}
`, os.Getenv("GITLAB_ACCESS_TOKEN"), os.Getenv("ENTITLE_OWNER_ID"), os.Getenv("ENTITLE_WORKFLOW_ID"), os.Getenv("ENTITLE_ROLE_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Updated field assertions
					resource.TestCheckResourceAttr("entitle_integration.my_gitlab", "name", "My Gitlab Integration UPDATED"),
					// Not updated field assertions
					resource.TestCheckResourceAttr("entitle_integration.my_gitlab", "requestable", "true"),
					resource.TestCheckResourceAttr("entitle_integration.my_gitlab", "requestable_by_default", "true"),
					resource.TestCheckResourceAttr("entitle_integration.my_gitlab", "application.name", "gitlab"),
					resource.TestCheckResourceAttr("entitle_integration.my_gitlab", "allowed_durations.0", "-1"),
					resource.TestCheckResourceAttr("entitle_integration.my_gitlab", "auto_assign_recommended_maintainers", "false"),
					resource.TestCheckResourceAttr("entitle_integration.my_gitlab", "auto_assign_recommended_owners", "false"),
					resource.TestCheckResourceAttr("entitle_integration.my_gitlab", "allow_creating_accounts", "false"),
					resource.TestCheckResourceAttr("entitle_integration.my_gitlab", "notify_about_external_permission_changes", "true"),
					resource.TestCheckResourceAttr("entitle_integration.my_gitlab", "owner.id", os.Getenv("ENTITLE_OWNER_ID")),
					resource.TestCheckResourceAttr("entitle_integration.my_gitlab", "readonly", "false"),
					resource.TestCheckResourceAttr("entitle_integration.my_gitlab", "workflow.id", os.Getenv("ENTITLE_WORKFLOW_ID")),
					resource.TestCheckResourceAttr("entitle_integration.my_gitlab", "prerequisite_permissions.0.role.id", os.Getenv("ENTITLE_ROLE_ID")),
					resource.TestCheckResourceAttrSet("entitle_integration.my_gitlab", "connection_json"),
				),
			},
		},
	})
}
