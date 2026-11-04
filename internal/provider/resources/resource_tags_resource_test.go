//go:build acceptance

package resources_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

// TestResourceTagsResource runs against a real tenant. It uses random tag
// names so it never touches tags that already exist on ENTITLE_RESOURCE_ID.
func TestResourceTagsResource(t *testing.T) {
	resourceID := os.Getenv("ENTITLE_RESOURCE_ID")
	suffix := strings.ToLower(acctest.RandString(8))
	tagA, tagB, tagC := "tf-acc-a-"+suffix, "tf-acc-b-"+suffix, "tf-acc-c-"+suffix

	config := func(tags ...string) string {
		return testhelpers.ProviderConfig + fmt.Sprintf(`
resource "entitle_resource_tags" "test" {
  resource_ids = ["%s"]
  tags         = %s
}

# Read back after the tags have been applied.
data "entitle_resource" "check" {
  id         = "%s"
  depends_on = [entitle_resource_tags.test]
}
`, resourceID, hclList(tags), resourceID)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: config(tagA, tagB),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("entitle_resource_tags.test", "resource_ids.#", "1"),
					resource.TestCheckTypeSetElemAttr("entitle_resource_tags.test", "resource_ids.*", resourceID),
					resource.TestCheckResourceAttr("entitle_resource_tags.test", "tags.#", "2"),
					resource.TestCheckTypeSetElemAttr("data.entitle_resource.check", "user_defined_tags.*", tagA),
					resource.TestCheckTypeSetElemAttr("data.entitle_resource.check", "user_defined_tags.*", tagB),
				),
			},
			// Update: replace tagB with tagC
			{
				Config: config(tagA, tagC),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("entitle_resource_tags.test", "tags.#", "2"),
					resource.TestCheckTypeSetElemAttr("data.entitle_resource.check", "user_defined_tags.*", tagA),
					resource.TestCheckTypeSetElemAttr("data.entitle_resource.check", "user_defined_tags.*", tagC),
					testhelpers.TestCheckSetElemAbsent("data.entitle_resource.check", "user_defined_tags", tagB),
				),
			},
			// Update: remove tagC
			{
				Config: config(tagA),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("entitle_resource_tags.test", "tags.#", "1"),
					resource.TestCheckTypeSetElemAttr("data.entitle_resource.check", "user_defined_tags.*", tagA),
					testhelpers.TestCheckSetElemAbsent("data.entitle_resource.check", "user_defined_tags", tagB),
					testhelpers.TestCheckSetElemAbsent("data.entitle_resource.check", "user_defined_tags", tagC),
				),
			},
		},
	})
}
