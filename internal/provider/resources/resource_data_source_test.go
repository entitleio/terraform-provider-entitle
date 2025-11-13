//go:build acceptance

package resources_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestResourceDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_resource" "my_resource" {
	id = "%s"
}
`, os.Getenv("ENTITLE_RESOURCE_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("data.entitle_resource.my_resource", "id", os.Getenv("ENTITLE_RESOURCE_ID")),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("data.entitle_resource.my_resource", "requestable"),
					resource.TestCheckResourceAttrSet("data.entitle_resource.my_resource", "user_defined_description"),
					resource.TestCheckResourceAttrSet("data.entitle_resource.my_resource", "name"),
					resource.TestCheckResourceAttrSet("data.entitle_resource.my_resource", "owner.id"),
					resource.TestCheckResourceAttrSet("data.entitle_resource.my_resource", "user_defined_tags.0"),
					resource.TestCheckResourceAttrSet("data.entitle_resource.my_resource", "integration.id"),
					resource.TestCheckResourceAttrSet("data.entitle_resource.my_resource", "workflow.id"),
				),
			},
		},
	})
}
