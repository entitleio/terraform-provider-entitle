//go:build acceptance
// +build acceptance

package accessRequestForwards_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestAccessRequestForwardDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_access_request_forward" "my_forward" {
	id = "%s"
}
`, os.Getenv("ENTITLE_ACCESS_REQUEST_FORWARD_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("data.entitle_access_request_forward.my_forward", "id", os.Getenv("ENTITLE_ACCESS_REQUEST_FORWARD_ID")),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("data.entitle_access_request_forward.my_forward", "target.id"),
					resource.TestCheckResourceAttrSet("data.entitle_access_request_forward.my_forward", "target.email"),
					resource.TestCheckResourceAttrSet("data.entitle_access_request_forward.my_forward", "forwarder.id"),
					resource.TestCheckResourceAttrSet("data.entitle_access_request_forward.my_forward", "forwarder.email"),
				),
			},
		},
	})
}
