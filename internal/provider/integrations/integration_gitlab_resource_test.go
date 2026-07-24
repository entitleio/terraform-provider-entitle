//go:build acceptance

package integrations_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/entitleio/terraform-provider-entitle/internal/provider/integrations"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestIntegrationGitlabResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
resource "entitle_integration_gitlab" "my_gitlab" {
  	name                               = "My Gitlab Integration"
    requestable                     = true
    requestable_by_default                     = true
	allowed_durations = [-1]
    auto_assign_recommended_maintainers      = false
    auto_assign_recommended_owners           = false
    allow_creating_accounts           = false
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
    connection_data = {
		domain                   = "https://gitlab.com"
		private_token            = "%s"
  	}
}
`, os.Getenv("ENTITLE_OWNER_ID"), os.Getenv("ENTITLE_WORKFLOW_ID"), os.Getenv("ENTITLE_OWNER_ID"), os.Getenv("ENTITLE_ROLE_ID"), os.Getenv("GITLAB_ACCESS_TOKEN")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "name", "My Gitlab Integration"),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "allow_changing_account_permissions", "true"),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "requestable", "true"),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "requestable_by_default", "true"),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "allowed_durations.0", "-1"),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "auto_assign_recommended_maintainers", "false"),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "auto_assign_recommended_owners", "false"),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "allow_creating_accounts", "false"),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "notify_about_external_permission_changes", "true"),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "owner.id", os.Getenv("ENTITLE_OWNER_ID")),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "readonly", "false"),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "workflow.id", os.Getenv("ENTITLE_WORKFLOW_ID")),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "prerequisite_permissions.0.role.id", os.Getenv("ENTITLE_ROLE_ID")),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "prerequisite_permissions.0.default", "true"),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "connection_data.domain", integrations.GitlabDefaultDomain),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("entitle_integration_gitlab.my_gitlab", "id"),

					resource.TestCheckResourceAttrSet("entitle_integration_gitlab.my_gitlab", "prerequisite_permissions.0.role.name"),
					resource.TestCheckResourceAttrSet("entitle_integration_gitlab.my_gitlab", "prerequisite_permissions.0.role.resource.name"),
					resource.TestCheckResourceAttrSet("entitle_integration_gitlab.my_gitlab", "prerequisite_permissions.0.role.resource.integration.name"),
				),
			},
			// Update testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
resource "entitle_integration_gitlab" "my_gitlab" {
  	name                               = "My Gitlab Integration"
    requestable                     = true
    requestable_by_default                     = true
	allowed_durations = [-1]
    auto_assign_recommended_maintainers      = false
    auto_assign_recommended_owners           = false
    allow_creating_accounts           = false
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
    connection_data = {
		domain                   = "https://gitlab.com"
		private_token            = "%s"
  	}
}
`, os.Getenv("ENTITLE_USER2_ID"), os.Getenv("ENTITLE_WORKFLOW_ID"), os.Getenv("ENTITLE_OWNER_ID"), os.Getenv("ENTITLE_ROLE_ID"), os.Getenv("GITLAB_ACCESS_TOKEN")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "name", "My Gitlab Integration"),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "allow_changing_account_permissions", "true"),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "requestable", "true"),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "requestable_by_default", "true"),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "allowed_durations.0", "-1"),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "auto_assign_recommended_maintainers", "false"),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "auto_assign_recommended_owners", "false"),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "allow_creating_accounts", "false"),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "notify_about_external_permission_changes", "true"),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "owner.id", os.Getenv("ENTITLE_USER2_ID")),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "readonly", "false"),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "workflow.id", os.Getenv("ENTITLE_WORKFLOW_ID")),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "prerequisite_permissions.0.role.id", os.Getenv("ENTITLE_ROLE_ID")),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "prerequisite_permissions.0.default", "true"),
					resource.TestCheckResourceAttr("entitle_integration_gitlab.my_gitlab", "connection_data.domain", integrations.GitlabDefaultDomain),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("entitle_integration_gitlab.my_gitlab", "id"),

					resource.TestCheckResourceAttrSet("entitle_integration_gitlab.my_gitlab", "prerequisite_permissions.0.role.name"),
					resource.TestCheckResourceAttrSet("entitle_integration_gitlab.my_gitlab", "prerequisite_permissions.0.role.resource.name"),
					resource.TestCheckResourceAttrSet("entitle_integration_gitlab.my_gitlab", "prerequisite_permissions.0.role.resource.integration.name"),
				),
			},
		},
	})
}

func TestIntegrationGitlabResourceAllowCreatingAccountsValidation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
resource "entitle_integration_gitlab" "my_gitlab" {
  	name                               = "My Gitlab Integration"
    requestable                     = true
    requestable_by_default                     = true
	allowed_durations = [-1]
    auto_assign_recommended_maintainers      = false
    auto_assign_recommended_owners           = false
    notify_about_external_permission_changes = true
	allow_creating_accounts = true
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
    connection_data = {
		domain                   = "https://gitlab.com"
		private_token            = "%s"
  	}
}
`, os.Getenv("ENTITLE_OWNER_ID"), os.Getenv("ENTITLE_WORKFLOW_ID"), os.Getenv("ENTITLE_OWNER_ID"), os.Getenv("ENTITLE_ROLE_ID"), os.Getenv("GITLAB_ACCESS_TOKEN")),
				ExpectError: regexp.MustCompile("Attribute allow_creating_accounts Value must be \"false\""),
			},
		},
	})
}

func TestIntegrationGitlabResourceAllowChangingAccountPermissionsValidation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
resource "entitle_integration_gitlab" "my_gitlab" {
  	name                               = "My Gitlab Integration"
    requestable                     = true
    requestable_by_default                     = true
	allowed_durations = [-1]
    auto_assign_recommended_maintainers      = false
    auto_assign_recommended_owners           = false
    notify_about_external_permission_changes = true
	allow_changing_account_permissions = false
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
    connection_data = {
		domain                   = "https://gitlab.com"
		private_token            = "%s"
  	}
}
`, os.Getenv("ENTITLE_OWNER_ID"), os.Getenv("ENTITLE_WORKFLOW_ID"), os.Getenv("ENTITLE_OWNER_ID"), os.Getenv("ENTITLE_ROLE_ID"), os.Getenv("GITLAB_ACCESS_TOKEN")),
				ExpectError: regexp.MustCompile("Attribute allow_changing_account_permissions Value must be \"true\""),
			},
		},
	})
}
