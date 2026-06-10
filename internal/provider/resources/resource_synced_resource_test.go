//go:build acceptance

package resources_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestResourceSyncedResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
			
			resource "entitle_resource_synced" "my_resource" {
			 	name                               = "%s"
				integration = {
				  id = "%s"
				}
			}
			`, os.Getenv("ENTITLE_RESOURCE_SYNCED_NAME"), os.Getenv("ENTITLE_INTEGRATION_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("entitle_resource_synced.my_resource", "name", os.Getenv("ENTITLE_RESOURCE_SYNCED_NAME")),
					resource.TestCheckResourceAttr("entitle_resource_synced.my_resource", "integration.id", os.Getenv("ENTITLE_INTEGRATION_ID")),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("entitle_resource_synced.my_resource", "id"),
					resource.TestCheckResourceAttrSet("entitle_resource_synced.my_resource", "external_id"),

					resource.TestCheckResourceAttrSet("entitle_resource_synced.my_resource", "requestable"),
				),
			},
			// Update testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
			
			resource "entitle_resource_synced" "my_resource" {
			 	name                               = "%s"
				integration = {
				  id = "%s"
				}
				requestable = false
			}
			`, os.Getenv("ENTITLE_RESOURCE_SYNCED_NAME"), os.Getenv("ENTITLE_INTEGRATION_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("entitle_resource_synced.my_resource", "name", os.Getenv("ENTITLE_RESOURCE_SYNCED_NAME")),
					resource.TestCheckResourceAttr("entitle_resource_synced.my_resource", "integration.id", os.Getenv("ENTITLE_INTEGRATION_ID")),
					resource.TestCheckResourceAttr("entitle_resource_synced.my_resource", "requestable", "false"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("entitle_resource_synced.my_resource", "id"),
					resource.TestCheckResourceAttrSet("entitle_resource_synced.my_resource", "external_id"),
				),
			},
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
			
			resource "entitle_resource_synced" "my_resource" {
			 	name                               = "%s"
				integration = {
				  id = "%s"
				}
				requestable = true
			}
			`, os.Getenv("ENTITLE_RESOURCE_SYNCED_NAME"), os.Getenv("ENTITLE_INTEGRATION_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("entitle_resource_synced.my_resource", "name", os.Getenv("ENTITLE_RESOURCE_SYNCED_NAME")),
					resource.TestCheckResourceAttr("entitle_resource_synced.my_resource", "integration.id", os.Getenv("ENTITLE_INTEGRATION_ID")),
					resource.TestCheckResourceAttr("entitle_resource_synced.my_resource", "requestable", "true"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("entitle_resource_synced.my_resource", "id"),
				),
			},
		},
	})
}

func TestResourceSyncedResourceByExternalID(t *testing.T) {
	if os.Getenv("ENTITLE_RESOURCE_SYNCED_EXTERNAL_ID") == "" {
		t.SkipNow()
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
			resource "entitle_resource_synced" "my_resource" {
			 	external_id                               = "%s"
				integration = {
				  id = "%s"
				}
			}
			`, os.Getenv("ENTITLE_RESOURCE_SYNCED_EXTERNAL_ID"), os.Getenv("ENTITLE_INTEGRATION_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("entitle_resource_synced.my_resource", "external_id", os.Getenv("ENTITLE_RESOURCE_SYNCED_EXTERNAL_ID")),
					resource.TestCheckResourceAttr("entitle_resource_synced.my_resource", "integration.id", os.Getenv("ENTITLE_INTEGRATION_ID")),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("entitle_resource_synced.my_resource", "id"),
					resource.TestCheckResourceAttrSet("entitle_resource_synced.my_resource", "name"),

					resource.TestCheckResourceAttrSet("entitle_resource_synced.my_resource", "requestable"),
				),
			},
		},
	})
}

func TestResourceSyncedResourceMissingParam(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
			resource "entitle_resource_synced" "my_resource" {
				integration = {
				  id = "%s"
				}
			}
			`, os.Getenv("ENTITLE_INTEGRATION_ID")),
				ExpectError: regexp.MustCompile("No attribute specified "),
			},
		},
	})
}
