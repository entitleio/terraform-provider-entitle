//go:build acceptance

package roles_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestRolesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_roles" "my_list" {
	filter {
		resource_id = "%s"
	}
}
`, os.Getenv("ENTITLE_RESOURCE_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("data.entitle_roles.my_list", "roles.0.id"),
					resource.TestCheckResourceAttrSet("data.entitle_roles.my_list", "roles.0.name"),
					resource.TestCheckResourceAttrSet("data.entitle_roles.my_list", "roles.0.external_id"),
					resource.TestCheckResourceAttrSet("data.entitle_roles.my_list", "roles.0.requestable"),
					resource.TestCheckResourceAttrSet("data.entitle_roles.my_list", "roles.0.resource_id"),
				),
			},
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_roles" "my_list" {
	filter {
		resource_id = "%s"
		search      = "valuethathcouldnotbefound"
	}
}
`, os.Getenv("ENTITLE_RESOURCE_ID")),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.entitle_roles.my_list", "roles.#", "0"),
				),
			},
			// Deprecated top-level resource_id still works.
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_roles" "my_list" {
	resource_id = "%s"
}
`, os.Getenv("ENTITLE_RESOURCE_ID")),
				Check: resource.TestCheckResourceAttrSet("data.entitle_roles.my_list", "roles.0.id"),
			},
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_roles" "my_list" {
	filter {
		resource_id = "%s"
	}
}
`, "00000000-0000-0000-0000-000000000000"),
				ExpectError: regexp.MustCompile(utils.ErrNotFound.Error()),
			},
		},
	})
}

func TestRolesDataSource_MissingResourceID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testhelpers.ProviderConfig + `
data "entitle_roles" "my_list" {
    # resource_id and integration_id missing
}
`,
				ExpectError: regexp.MustCompile(`At\s+least\s+one\s+attribute\s+out\s+of`),
				PlanOnly:    true,
			},
			{
				Config: testhelpers.ProviderConfig + `
data "entitle_roles" "my_list" {
  filter {
    search = "admin" # no scope
  }
}
`,
				ExpectError: regexp.MustCompile(`At\s+least\s+one\s+attribute\s+out\s+of`),
				PlanOnly:    true,
			},
		},
	})
}

func TestRolesDataSource_DeprecatedScopeConflicts(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testhelpers.ProviderConfig + `
data "entitle_roles" "my_list" {
  resource_id = "00000000-0000-0000-0000-000000000001"
  filter {
    resource_id = "00000000-0000-0000-0000-000000000002"
  }
}
`,
				ExpectError: regexp.MustCompile(`cannot\s+be\s+specified\s+when`),
				PlanOnly:    true,
			},
		},
	})
}

func TestRolesDataSource_IDsFilter(t *testing.T) {
	roleID := os.Getenv("ENTITLE_ROLE_ID")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testhelpers.ProviderConfig + fmt.Sprintf(`
data "entitle_roles" "by_id" {
  filter {
    ids = ["%s"]
  }
}
`, roleID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.entitle_roles.by_id", "roles.#", "1"),
					resource.TestCheckResourceAttr("data.entitle_roles.by_id", "roles.0.id", roleID),
				),
			},
		},
	})
}
