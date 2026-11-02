package roles_test

// Unit tests for entitle_role_settings, run through Terraform against the fake
// roles API (role_settings_fake_unit_test.go).

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	roleA = "11111111-aaaa-4aaa-8aaa-111111111111"
	roleB = "22222222-bbbb-4bbb-8bbb-222222222222"
	roleC = "33333333-cccc-4ccc-8ccc-333333333333"

	roleWF1 = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	roleWF2 = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
)

// chunk_size = 1 forces one bulk request per role.
func roleSettingsConfig(ids []string, body string) string {
	quoted := make([]string, len(ids))
	for i, id := range ids {
		quoted[i] = fmt.Sprintf("%q", id)
	}
	return fmt.Sprintf(`
provider "entitle" {
  bulk {
    chunk_size  = 1
    parallelism = 1
  }
}

resource "entitle_role_settings" "test" {
  role_ids = [%s]
%s
}
`, strings.Join(quoted, ", "), body)
}

// Only configured attributes are sent, to the roles bulk endpoint; reads use the roles list with ids.
func TestUnitRoleSettingsResource_CreateSendsOnlyConfigured(t *testing.T) {
	fake := setupFakeRoles(t, roleA, roleB)
	fake.update(roleA, func(r *fakeRole) { r.WorkflowID = roleWF2 }) // not managed below
	var mark int

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: roleSettingsProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() { mark = fake.callCount() },
				Config:    roleSettingsConfig([]string{roleA, roleB}, "  requestable = false\n"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("entitle_role_settings.test", "role_ids.#", "2"),
					fake.check(roleA, false, roleWF2),
					fake.check(roleB, false, ""),
					fake.expectUpdates(&mark, map[string][]string{
						roleA: {"requestable"},
						roleB: {"requestable"},
					}),
					fake.checkRequests(),
				),
			},
		},
	})
}

// New roles get every configured setting; roles that stay get only what changed.
func TestUnitRoleSettingsResource_UpdateSendsOnlyChanges(t *testing.T) {
	fake := setupFakeRoles(t, roleA, roleB, roleC)
	var mark int

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: roleSettingsProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: roleSettingsConfig([]string{roleA, roleB}, fmt.Sprintf("  requestable = false\n  workflow_id = %q\n", roleWF1)),
			},
			{
				PreConfig: func() { mark = fake.callCount() },
				Config:    roleSettingsConfig([]string{roleA, roleB, roleC}, fmt.Sprintf("  requestable = false\n  workflow_id = %q\n", roleWF2)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("entitle_role_settings.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.check(roleA, false, roleWF2),
					fake.check(roleC, false, roleWF2),
					fake.expectUpdates(&mark, map[string][]string{
						roleA: {"workflow"},
						roleB: {"workflow"},
						roleC: {"requestable", "workflow"},
					}),
					fake.checkRequests(),
				),
			},
		},
	})
}

// An attribute removed from configuration is no longer managed: nothing is
// sent, its value stays, and later changes to it are not drift.
func TestUnitRoleSettingsResource_RemovedAttributeBecomesUnmanaged(t *testing.T) {
	fake := setupFakeRoles(t, roleA)
	var mark int

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: roleSettingsProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: roleSettingsConfig([]string{roleA}, fmt.Sprintf("  requestable = false\n  workflow_id = %q\n", roleWF1)),
			},
			{
				PreConfig: func() { mark = fake.callCount() },
				Config:    roleSettingsConfig([]string{roleA}, fmt.Sprintf("  workflow_id = %q\n", roleWF1)),
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.check(roleA, false, roleWF1),
					fake.expectUpdates(&mark, map[string][]string{}),
				),
			},
			{
				PreConfig: func() { fake.update(roleA, func(r *fakeRole) { r.Requestable = true }) },
				Config:    roleSettingsConfig([]string{roleA}, fmt.Sprintf("  workflow_id = %q\n", roleWF1)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				// The value changed outside Terraform is left as it is.
				Check: fake.check(roleA, true, roleWF1),
			},
		},
	})
}

// A managed setting changed outside Terraform is detected on refresh and
// re-applied to that role only.
func TestUnitRoleSettingsResource_Drift(t *testing.T) {
	fake := setupFakeRoles(t, roleA, roleB)
	config := roleSettingsConfig([]string{roleA, roleB}, fmt.Sprintf("  requestable = false\n  workflow_id = %q\n", roleWF1))
	var mark int

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: roleSettingsProviderFactories,
		Steps: []resource.TestStep{
			{Config: config},
			{
				PreConfig: func() {
					fake.update(roleB, func(r *fakeRole) { r.Requestable = true })
					mark = fake.callCount()
				},
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("entitle_role_settings.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.check(roleB, false, roleWF1),
					fake.expectUpdates(&mark, map[string][]string{
						roleB: {"requestable", "workflow"},
					}),
				),
			},
		},
	})
}

// Removing an ID or destroying leaves the settings in place and sends nothing.
func TestUnitRoleSettingsResource_RemoveAndDestroyLeaveSettings(t *testing.T) {
	fake := setupFakeRoles(t, roleA, roleB)
	body := fmt.Sprintf("  requestable = false\n  workflow_id = %q\n", roleWF1)
	var mark int

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: roleSettingsProviderFactories,
		Steps: []resource.TestStep{
			{Config: roleSettingsConfig([]string{roleA, roleB}, body)},
			{
				PreConfig: func() { mark = fake.callCount() },
				Config:    roleSettingsConfig([]string{roleA}, body),
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.check(roleB, false, roleWF1),
					fake.expectUpdates(&mark, map[string][]string{}),
					func(*terraform.State) error { mark = fake.callCount(); return nil },
				),
			},
		},
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			fake.check(roleA, false, roleWF1),
			func(*terraform.State) error {
				if n := fake.callCount(); n != mark {
					return fmt.Errorf("destroy sent %d bulk requests, want 0", n-mark)
				}
				return nil
			},
		),
	})
}

// A deleted role still in configuration fails with an error naming it;
// dropping it from configuration converges.
func TestUnitRoleSettingsResource_MissingRole(t *testing.T) {
	fake := setupFakeRoles(t, roleA)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: roleSettingsProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      roleSettingsConfig([]string{roleA, roleB}, "  requestable = false\n"),
				ExpectError: regexp.MustCompile(roleB),
			},
			{
				Config: roleSettingsConfig([]string{roleA}, "  requestable = false\n"),
				Check:  fake.check(roleA, false, ""),
			},
		},
	})
}

// A failed job surfaces the API error; the next apply converges.
func TestUnitRoleSettingsResource_FailedJob(t *testing.T) {
	fake := setupFakeRoles(t, roleA)
	config := roleSettingsConfig([]string{roleA}, "  requestable = false\n")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: roleSettingsProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig:   fake.failNext,
				Config:      config,
				ExpectError: regexp.MustCompile(`simulated\s+failure`),
			},
			{
				Config: config,
				Check:  fake.check(roleA, false, ""),
			},
		},
	})
}

func TestUnitRoleSettingsResource_Validation(t *testing.T) {
	setupFakeRoles(t)

	cases := map[string]struct {
		config string
		err    string
	}{
		"no settings": {
			config: roleSettingsConfig([]string{roleA}, ""),
			err:    `At least one of these attributes must be configured`,
		},
		"empty role_ids": {
			config: roleSettingsConfig([]string{}, "  requestable = true\n"),
			err:    `role_ids set must contain at least 1 elements`,
		},
		"invalid role id": {
			config: roleSettingsConfig([]string{"not-a-uuid"}, "  requestable = true\n"),
			err:    `UUID Validate failed`,
		},
		"invalid workflow id": {
			config: roleSettingsConfig([]string{roleA}, "  workflow_id = \"not-a-uuid\"\n"),
			err:    `UUID Validate failed`,
		},
		"owner is not supported": {
			config: roleSettingsConfig([]string{roleA}, "  owner = \"alice@example.com\"\n"),
			err:    `An argument named "owner" is not expected here`,
		},
		"maintainers are not supported": {
			config: roleSettingsConfig([]string{roleA}, "  maintainers = []\n"),
			err:    `An argument named "maintainers" is not expected here`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: roleSettingsProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      tc.config,
						PlanOnly:    true,
						ExpectError: regexp.MustCompile(strings.ReplaceAll(tc.err, " ", `\s+`)),
					},
				},
			})
		})
	}
}
