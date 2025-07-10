//go:build acceptance
// +build acceptance

package accessRequestForwards_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccessRequestForwardResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`

resource "entitle_access_request_forward" "my_forward" {
	forwarder = {
		id = "%s"
	}
	target = {
		id = "%s"
	}
}
`, os.Getenv("ENTITLE_OWNER_ID"), os.Getenv("ENTITLE_OWNER_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("entitle_access_request_forward.my_forward", "forwarder.id", os.Getenv("ENTITLE_OWNER_ID")),
					resource.TestCheckResourceAttr("entitle_access_request_forward.my_forward", "target.id", os.Getenv("ENTITLE_OWNER_ID")),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("entitle_access_request_forward.my_forward", "target.email"),
					resource.TestCheckResourceAttrSet("entitle_access_request_forward.my_forward", "target.email"),
				),
			},
		},
	})
}
