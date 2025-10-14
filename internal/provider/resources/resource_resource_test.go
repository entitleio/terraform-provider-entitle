//go:build acceptance
// +build acceptance

package resources_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestResourceResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`

resource "entitle_resource" "my_resource" {
  	name                               = "My Resource"
	user_defined_description = "test description"
    requestable                     = true
	allowed_durations = [-1]
    owner = {
      id    = "%s"
    }
    workflow = {
      id   = "%s"
    }
	integration = {
	  id = "%s"
	}
	maintainers = [
		{
			type = "user"
			entity = {
				id = "%s"
			}
		}
	]
	user_defined_tags = [
		"test1",
		"test2"
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
`, os.Getenv("ENTITLE_OWNER_ID"), os.Getenv("ENTITLE_WORKFLOW_ID"), os.Getenv("ENTITLE_MANUAL_INTEGRATION_ID"), os.Getenv("ENTITLE_OWNER_ID"), os.Getenv("ENTITLE_ROLE_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("entitle_resource.my_resource", "name", "My Resource"),
					resource.TestCheckResourceAttr("entitle_resource.my_resource", "user_defined_description", "test description"),
					resource.TestCheckResourceAttr("entitle_resource.my_resource", "requestable", "true"),
					resource.TestCheckResourceAttr("entitle_resource.my_resource", "allowed_durations.0", "-1"),
					resource.TestCheckResourceAttr("entitle_resource.my_resource", "owner.id", os.Getenv("ENTITLE_OWNER_ID")),
					resource.TestCheckResourceAttr("entitle_resource.my_resource", "workflow.id", os.Getenv("ENTITLE_WORKFLOW_ID")),
					resource.TestCheckResourceAttr("entitle_resource.my_resource", "integration.id", os.Getenv("ENTITLE_MANUAL_INTEGRATION_ID")),
					resource.TestCheckResourceAttr("entitle_resource.my_resource", "user_defined_tags.0", "test1"),
					resource.TestCheckResourceAttr("entitle_resource.my_resource", "user_defined_tags.1", "test2"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("entitle_resource.my_resource", "id"),

					resource.TestCheckResourceAttrSet("entitle_resource.my_resource", "prerequisite_permissions.0.role.name"),
					resource.TestCheckResourceAttrSet("entitle_resource.my_resource", "prerequisite_permissions.0.role.resource.name"),
					resource.TestCheckResourceAttrSet("entitle_resource.my_resource", "prerequisite_permissions.0.role.resource.integration.name"),
					resource.TestCheckResourceAttrSet("entitle_resource.my_resource", "prerequisite_permissions.0.role.resource.integration.application.name"),
				),
			},
			// Update testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`

resource "entitle_resource" "my_resource" {
  	name                               = "My Resource"
	user_defined_description = "test description UPDATED"
    requestable                     = true
	allowed_durations = [-1]
    owner = {
      id    = "%s"
    }
    workflow = {
      id   = "%s"
    }
	integration = {
	  id = "%s"
	}
	maintainers = [
		{
			type = "user"
			entity = {
				id = "%s"
			}
		}
	]
	user_defined_tags = [
		"test1",
		"test2"
	]
}
`, os.Getenv("ENTITLE_OWNER_ID"), os.Getenv("ENTITLE_WORKFLOW_ID"), os.Getenv("ENTITLE_MANUAL_INTEGRATION_ID"), os.Getenv("ENTITLE_OWNER_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("entitle_resource.my_resource", "name", "My Resource"),
					resource.TestCheckResourceAttr("entitle_resource.my_resource", "user_defined_description", "test description UPDATED"),
					resource.TestCheckResourceAttr("entitle_resource.my_resource", "requestable", "true"),
					resource.TestCheckResourceAttr("entitle_resource.my_resource", "allowed_durations.0", "-1"),
					resource.TestCheckResourceAttr("entitle_resource.my_resource", "owner.id", os.Getenv("ENTITLE_OWNER_ID")),
					resource.TestCheckResourceAttr("entitle_resource.my_resource", "workflow.id", os.Getenv("ENTITLE_WORKFLOW_ID")),
					resource.TestCheckResourceAttr("entitle_resource.my_resource", "integration.id", os.Getenv("ENTITLE_MANUAL_INTEGRATION_ID")),
					resource.TestCheckResourceAttr("entitle_resource.my_resource", "user_defined_tags.0", "test1"),
					resource.TestCheckResourceAttr("entitle_resource.my_resource", "user_defined_tags.1", "test2"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("entitle_resource.my_resource", "id"),

					//resource.TestCheckResourceAttrSet("entitle_resource.my_resource", "prerequisite_permissions.0.role.name"),
					//resource.TestCheckResourceAttrSet("entitle_resource.my_resource", "prerequisite_permissions.0.role.resource.name"),
					//resource.TestCheckResourceAttrSet("entitle_resource.my_resource", "prerequisite_permissions.0.role.resource.integration.name"),
					//resource.TestCheckResourceAttrSet("entitle_resource.my_resource", "prerequisite_permissions.0.role.resource.integration.application.name"),
				),
			},
		},
	})
}

// TestResourceResourceNullFields tests that null values for optional fields like
// allowed_durations and maintainers don't cause drift when imported or read from API
func TestResourceResourceNullFields(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create resource with minimal required fields (no allowed_durations, no maintainers)
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
resource "entitle_resource" "minimal_resource" {
	name        = "Minimal Resource"
	requestable = false
	integration = {
		id = "%s"
	}
}
`, os.Getenv("ENTITLE_MANUAL_INTEGRATION_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify basic attributes
					resource.TestCheckResourceAttr("entitle_resource.minimal_resource", "name", "Minimal Resource"),
					resource.TestCheckResourceAttr("entitle_resource.minimal_resource", "requestable", "false"),
					resource.TestCheckResourceAttr("entitle_resource.minimal_resource", "integration.id", os.Getenv("ENTITLE_MANUAL_INTEGRATION_ID")),
					// Verify ID is set
					resource.TestCheckResourceAttrSet("entitle_resource.minimal_resource", "id"),
					// Verify integration name is populated from API
					resource.TestCheckResourceAttrSet("entitle_resource.minimal_resource", "integration.name"),
				),
			},
			// Apply same config again to verify no drift with null fields
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
resource "entitle_resource" "minimal_resource" {
	name        = "Minimal Resource"
	requestable = false
	integration = {
		id = "%s"
	}
}
`, os.Getenv("ENTITLE_MANUAL_INTEGRATION_ID")),
				PlanOnly: true,
			},
			// Import the resource and verify no drift
			{
				ResourceName:      "entitle_resource.minimal_resource",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
