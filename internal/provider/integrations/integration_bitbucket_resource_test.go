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

func TestIntegrationBitbucketResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
resource "entitle_integration_bitbucket" "my_bitbucket" {
  	name                               = "My Bitbucket Integration"
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
		email                   = "%s"
		app_token            = "%s"
  	}
}
`, os.Getenv("ENTITLE_OWNER_ID"), os.Getenv("ENTITLE_WORKFLOW_ID"), os.Getenv("ENTITLE_OWNER_ID"), os.Getenv("ENTITLE_ROLE_ID"), os.Getenv("BITBUCKET_EMAIL"), os.Getenv("BITBUCKET_APP_TOKEN")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "name", "My Bitbucket Integration"),
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "allow_changing_account_permissions", "true"),
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "requestable", "true"),
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "requestable_by_default", "true"),
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "allowed_durations.0", "-1"),
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "auto_assign_recommended_maintainers", "false"),
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "auto_assign_recommended_owners", "false"),
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "allow_creating_accounts", "false"),
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "notify_about_external_permission_changes", "true"),
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "owner.id", os.Getenv("ENTITLE_OWNER_ID")),
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "readonly", "false"),
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "workflow.id", os.Getenv("ENTITLE_WORKFLOW_ID")),
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "prerequisite_permissions.0.role.id", os.Getenv("ENTITLE_ROLE_ID")),
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "prerequisite_permissions.0.default", "true"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("entitle_integration_bitbucket.my_bitbucket", "id"),

					resource.TestCheckResourceAttrSet("entitle_integration_bitbucket.my_bitbucket", "prerequisite_permissions.0.role.name"),
					resource.TestCheckResourceAttrSet("entitle_integration_bitbucket.my_bitbucket", "prerequisite_permissions.0.role.resource.name"),
					resource.TestCheckResourceAttrSet("entitle_integration_bitbucket.my_bitbucket", "prerequisite_permissions.0.role.resource.integration.name"),
				),
			},
			// Update testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
resource "entitle_integration_bitbucket" "my_bitbucket" {
  	name                               = "My Bitbucket Integration"
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
		email                   = "%s"
		app_token            = "%s"
  	}
}
`, os.Getenv("ENTITLE_USER2_ID"), os.Getenv("ENTITLE_WORKFLOW_ID"), os.Getenv("ENTITLE_OWNER_ID"), os.Getenv("ENTITLE_ROLE_ID"), os.Getenv("BITBUCKET_EMAIL"), os.Getenv("BITBUCKET_APP_TOKEN")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "name", "My Bitbucket Integration"),
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "allow_changing_account_permissions", "true"),
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "requestable", "true"),
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "requestable_by_default", "true"),
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "allowed_durations.0", "-1"),
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "auto_assign_recommended_maintainers", "false"),
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "auto_assign_recommended_owners", "false"),
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "allow_creating_accounts", "false"),
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "notify_about_external_permission_changes", "true"),
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "owner.id", os.Getenv("ENTITLE_USER2_ID")),
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "readonly", "false"),
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "workflow.id", os.Getenv("ENTITLE_WORKFLOW_ID")),
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "prerequisite_permissions.0.role.id", os.Getenv("ENTITLE_ROLE_ID")),
					resource.TestCheckResourceAttr("entitle_integration_bitbucket.my_bitbucket", "prerequisite_permissions.0.default", "true"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("entitle_integration_bitbucket.my_bitbucket", "id"),

					resource.TestCheckResourceAttrSet("entitle_integration_bitbucket.my_bitbucket", "prerequisite_permissions.0.role.name"),
					resource.TestCheckResourceAttrSet("entitle_integration_bitbucket.my_bitbucket", "prerequisite_permissions.0.role.resource.name"),
					resource.TestCheckResourceAttrSet("entitle_integration_bitbucket.my_bitbucket", "prerequisite_permissions.0.role.resource.integration.name"),
				),
			},
		},
	})
}

func TestIntegrationBitbucketResourceAllowCreatingAccountsValidation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
resource "entitle_integration_bitbucket" "my_bitbucket" {
  	name                               = "My Bitbucket Integration"
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
		email                   = "%s"
		app_token            = "%s"
  	}
}
`, os.Getenv("ENTITLE_OWNER_ID"), os.Getenv("ENTITLE_WORKFLOW_ID"), os.Getenv("ENTITLE_OWNER_ID"), os.Getenv("ENTITLE_ROLE_ID"), os.Getenv("BITBUCKET_EMAIL"), os.Getenv("BITBUCKET_APP_TOKEN")),
				ExpectError: regexp.MustCompile("Attribute allow_creating_accounts Value must be \"false\""),
			},
		},
	})
}

func TestIntegrationBitbucketResourceAllowChangingAccountPermissionsValidation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
resource "entitle_integration_bitbucket" "my_bitbucket" {
  	name                               = "My Bitbucket Integration"
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
		email                   = "%s"
		app_token            = "%s"
  	}
}
`, os.Getenv("ENTITLE_OWNER_ID"), os.Getenv("ENTITLE_WORKFLOW_ID"), os.Getenv("ENTITLE_OWNER_ID"), os.Getenv("ENTITLE_ROLE_ID"), os.Getenv("BITBUCKET_EMAIL"), os.Getenv("BITBUCKET_APP_TOKEN")),
				ExpectError: regexp.MustCompile("Attribute allow_changing_account_permissions Value must be \"true\""),
			},
		},
	})
}
