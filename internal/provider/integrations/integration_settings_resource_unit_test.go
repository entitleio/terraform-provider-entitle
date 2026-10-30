package integrations_test

// Unit tests for entitle_integration_settings, run through Terraform against
// the fake integrations API (integration_settings_fake_unit_test.go).

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
)

const (
	intg1 = "44444444-4444-4444-4444-444444444444"
	intg2 = "55555555-5555-5555-5555-555555555555"
	intg3 = "66666666-6666-6666-6666-666666666666"

	wf1   = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	wf2   = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	group = "cccccccc-cccc-cccc-cccc-cccccccccccc"
	alice = "alice@example.com"
	bob   = "bob@example.com"
	dave  = "dave@example.com"
)

// chunk_size = 1 forces one bulk request per integration.
func settingsConfig(ids []string, body string) string {
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

resource "entitle_integration_settings" "test" {
  integration_ids = [%s]
%s
}
`, strings.Join(quoted, ", "), body)
}

// --- expectations ------------------------------------------------------------

// sent describes which fields an update must (and must not) contain.
type sent struct {
	requestable *bool
	workflow    string
	owner       string
	maintainers bool
}

func (s sent) check(u client.BulkSettingsUpdate) error {
	switch {
	case (s.requestable == nil) != (u.Requestable == nil):
		return fmt.Errorf("requestable sent = %v, want %v", u.Requestable != nil, s.requestable != nil)
	case s.requestable != nil && *u.Requestable != *s.requestable:
		return fmt.Errorf("requestable = %v, want %v", *u.Requestable, *s.requestable)
	case (s.workflow == "") != (u.Workflow == nil):
		return fmt.Errorf("workflow sent = %v, want %v", u.Workflow != nil, s.workflow != "")
	case s.workflow != "" && u.Workflow.Id != s.workflow:
		return fmt.Errorf("workflow = %s, want %s", u.Workflow.Id, s.workflow)
	case (s.owner == "") != (u.Owner == nil):
		return fmt.Errorf("owner sent = %v, want %v", u.Owner != nil, s.owner != "")
	case s.owner != "" && u.Owner.Id != s.owner:
		return fmt.Errorf("owner = %s, want %s", u.Owner.Id, s.owner)
	case s.maintainers != (u.Maintainers != nil):
		return fmt.Errorf("maintainers sent = %v, want %v", u.Maintainers != nil, s.maintainers)
	}
	return nil
}

func boolp(v bool) *bool { return &v }

func requestable(want bool) func(fakeIntegration) error {
	return func(in fakeIntegration) error {
		if in.Requestable != want {
			return fmt.Errorf("requestable = %v, want %v", in.Requestable, want)
		}
		return nil
	}
}

func workflow(want string) func(fakeIntegration) error {
	return func(in fakeIntegration) error {
		if in.WorkflowID != want {
			return fmt.Errorf("workflow = %q, want %q", in.WorkflowID, want)
		}
		return nil
	}
}

func owner(ref string) func(fakeIntegration) error {
	return func(in fakeIntegration) error {
		if in.Owner == nil || in.Owner.ID != resolve(ref).ID {
			return fmt.Errorf("owner = %+v, want %s", in.Owner, ref)
		}
		return nil
	}
}

// maintainers checks the exact maintainer set; entries are "type:ref".
func maintainers(want ...string) func(fakeIntegration) error {
	return func(in fakeIntegration) error {
		var got, exp []string
		for _, m := range in.Maintainers {
			got = append(got, m.String())
		}
		for _, w := range want {
			typ, ref, _ := strings.Cut(w, ":")
			exp = append(exp, typ+":"+resolve(ref).ID)
		}
		sort.Strings(got)
		sort.Strings(exp)
		if !slices.Equal(got, exp) {
			return fmt.Errorf("maintainers = %v, want %v", got, exp)
		}
		return nil
	}
}

func all(checks ...func(fakeIntegration) error) func(fakeIntegration) error {
	return func(in fakeIntegration) error {
		for _, c := range checks {
			if err := c(in); err != nil {
				return err
			}
		}
		return nil
	}
}

// --- tests -------------------------------------------------------------------

// Only configured attributes are sent, to the integrations bulk endpoint;
// reads use the integrations list with ids.
func TestUnitIntegrationSettingsResource_CreateSendsOnlyConfigured(t *testing.T) {
	fake := setupFake(t, intg1, intg2)
	fake.update(intg1, func(in *fakeIntegration) { o := resolve(dave); in.Owner = &o })
	var mark int

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() { mark = fake.callCount() },
				Config: settingsConfig([]string{intg1, intg2}, fmt.Sprintf(`
  requestable = false
  workflow_id = %q
`, wf1)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("entitle_integration_settings.test", "integration_ids.#", "2"),
					fake.check(intg1, all(requestable(false), workflow(wf1), owner(dave))),
					fake.check(intg2, all(requestable(false), workflow(wf1))),
					fake.expectUpdates(&mark, map[string]func(client.BulkSettingsUpdate) error{
						intg1: sent{requestable: boolp(false), workflow: wf1}.check,
						intg2: sent{requestable: boolp(false), workflow: wf1}.check,
					}),
					fake.checkRequests(),
				),
			},
		},
	})
}

// New integrations get every configured setting; integrations that stay get only what changed.
func TestUnitIntegrationSettingsResource_UpdateSendsOnlyChanges(t *testing.T) {
	fake := setupFake(t, intg1, intg2, intg3)
	var mark int

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: settingsConfig([]string{intg1, intg2}, fmt.Sprintf(`
  requestable = false
  owner       = %q
`, alice)),
			},
			{
				PreConfig: func() { mark = fake.callCount() },
				Config: settingsConfig([]string{intg1, intg2, intg3}, fmt.Sprintf(`
  requestable = false
  owner       = %q
`, bob)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("entitle_integration_settings.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.check(intg1, all(requestable(false), owner(bob))),
					fake.check(intg3, all(requestable(false), owner(bob))),
					fake.expectUpdates(&mark, map[string]func(client.BulkSettingsUpdate) error{
						intg1: sent{owner: bob}.check,
						intg2: sent{owner: bob}.check,
						intg3: sent{requestable: boolp(false), owner: bob}.check,
					}),
					fake.checkRequests(),
				),
			},
		},
	})
}

// An attribute removed from configuration is no longer managed: nothing is
// sent, its value stays, and later changes to it are not drift.
func TestUnitIntegrationSettingsResource_RemovedAttributeBecomesUnmanaged(t *testing.T) {
	fake := setupFake(t, intg1)
	var mark int

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: settingsConfig([]string{intg1}, fmt.Sprintf("  requestable = false\n  workflow_id = %q\n", wf1)),
			},
			{
				PreConfig: func() { mark = fake.callCount() },
				Config:    settingsConfig([]string{intg1}, fmt.Sprintf("  workflow_id = %q\n", wf1)),
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.check(intg1, all(requestable(false), workflow(wf1))),
					fake.expectNoUpdates(&mark),
				),
			},
			{
				PreConfig: func() { fake.update(intg1, func(in *fakeIntegration) { in.Requestable = true }) },
				Config:    settingsConfig([]string{intg1}, fmt.Sprintf("  workflow_id = %q\n", wf1)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// A managed setting changed outside Terraform is detected on refresh and
// re-applied to that integration only.
func TestUnitIntegrationSettingsResource_Drift(t *testing.T) {
	fake := setupFake(t, intg1, intg2)
	config := settingsConfig([]string{intg1, intg2}, fmt.Sprintf("  workflow_id = %q\n", wf1))
	var mark int

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{Config: config},
			{
				PreConfig: func() {
					fake.update(intg2, func(in *fakeIntegration) { in.WorkflowID = wf2 })
					mark = fake.callCount()
				},
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("entitle_integration_settings.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.check(intg2, workflow(wf1)),
					fake.expectUpdates(&mark, map[string]func(client.BulkSettingsUpdate) error{
						intg2: sent{workflow: wf1}.check,
					}),
				),
			},
		},
	})
}

// The owner can be given as an email or a UUID; both match what the API returns.
func TestUnitIntegrationSettingsResource_OwnerByEmailOrID(t *testing.T) {
	fake := setupFake(t, intg1)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: settingsConfig([]string{intg1}, fmt.Sprintf("  owner = %q\n", strings.ToUpper(alice))),
				Check:  fake.check(intg1, owner(alice)),
			},
			{
				Config: settingsConfig([]string{intg1}, fmt.Sprintf("  owner = %q\n", resolve(bob).ID)),
				Check:  fake.check(intg1, owner(bob)),
			},
		},
	})
}

// maintainers replaces the whole list; extra maintainers are drift; an empty list removes all.
func TestUnitIntegrationSettingsResource_MaintainersReplace(t *testing.T) {
	fake := setupFake(t, intg1)
	fake.update(intg1, func(in *fakeIntegration) { in.Maintainers = []maintainer{{Type: "user", identity: resolve(dave)}} })
	config := settingsConfig([]string{intg1}, fmt.Sprintf(`
  maintainers = [
    { type = "user", id = %q },
    { type = "group", id = %q },
  ]
`, alice, group))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  fake.check(intg1, maintainers("user:"+alice, "group:"+group)),
			},
			{
				PreConfig: func() {
					fake.update(intg1, func(in *fakeIntegration) {
						in.Maintainers = append(in.Maintainers, maintainer{Type: "user", identity: resolve(bob)})
					})
				},
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("entitle_integration_settings.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: fake.check(intg1, maintainers("user:"+alice, "group:"+group)),
			},
			{
				Config: settingsConfig([]string{intg1}, "  maintainers = []\n"),
				Check:  fake.check(intg1, maintainers()),
			},
		},
	})
}

// additional_maintainers adds, keeps existing maintainers, and removes only the ones it added.
func TestUnitIntegrationSettingsResource_AdditionalMaintainers(t *testing.T) {
	fake := setupFake(t, intg1, intg2)
	fake.update(intg1, func(in *fakeIntegration) { in.Maintainers = []maintainer{{Type: "user", identity: resolve(dave)}} }) // pre-existing
	additional := func(refs ...string) string {
		items := make([]string, len(refs))
		for i, ref := range refs {
			items[i] = fmt.Sprintf("    { type = \"user\", id = %q },", ref)
		}
		return "  additional_maintainers = [\n" + strings.Join(items, "\n") + "\n  ]\n"
	}
	var mark int

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: settingsConfig([]string{intg1, intg2}, additional(dave, alice)),
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.check(intg1, maintainers("user:"+dave, "user:"+alice)),
					fake.check(intg2, maintainers("user:"+dave, "user:"+alice)),
				),
			},
			// Maintainers added outside Terraform are not drift.
			{
				PreConfig: func() {
					fake.update(intg2, func(in *fakeIntegration) {
						in.Maintainers = append(in.Maintainers, maintainer{Type: "user", identity: resolve(bob)})
					})
				},
				Config: settingsConfig([]string{intg1, intg2}, additional(dave, alice)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			// Taking dave out of the list removes him only where Terraform added him.
			{
				PreConfig: func() { mark = fake.callCount() },
				Config:    settingsConfig([]string{intg1, intg2}, additional(alice)),
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.check(intg1, maintainers("user:"+dave, "user:"+alice)),
					fake.check(intg2, maintainers("user:"+alice, "user:"+bob)),
					fake.expectUpdates(&mark, map[string]func(client.BulkSettingsUpdate) error{
						intg2: sent{maintainers: true}.check,
					}),
				),
			},
			// Removing intg2 from integration_ids removes the maintainers Terraform added to it.
			{
				Config: settingsConfig([]string{intg1}, additional(alice)),
				Check:  fake.check(intg2, maintainers("user:"+bob)),
			},
			// Switching to maintainers sends a single replace (the fake fails the test if combined with add/remove).
			{
				Config: settingsConfig([]string{intg1}, fmt.Sprintf("  maintainers = [{ type = \"user\", id = %q }]\n", bob)),
				Check:  fake.check(intg1, maintainers("user:"+bob)),
			},
		},
	})
}

// Removing an ID or destroying leaves the settings in place and sends nothing.
func TestUnitIntegrationSettingsResource_RemoveAndDestroyLeaveSettings(t *testing.T) {
	fake := setupFake(t, intg1, intg2)
	body := fmt.Sprintf("  requestable = false\n  workflow_id = %q\n", wf1)
	var mark int

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{Config: settingsConfig([]string{intg1, intg2}, body)},
			{
				PreConfig: func() { mark = fake.callCount() },
				Config:    settingsConfig([]string{intg1}, body),
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.check(intg2, all(requestable(false), workflow(wf1))),
					fake.expectNoUpdates(&mark),
					func(*terraform.State) error { mark = fake.callCount(); return nil },
				),
			},
		},
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			fake.check(intg1, all(requestable(false), workflow(wf1))),
			fake.expectNoUpdates(&mark),
		),
	})
}

// A deleted integration still in configuration fails with an error naming it;
// dropping it from configuration converges.
func TestUnitIntegrationSettingsResource_MissingIntegration(t *testing.T) {
	fake := setupFake(t, intg1)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      settingsConfig([]string{intg1, intg2}, "  requestable = false\n"),
				ExpectError: regexp.MustCompile(intg2),
			},
			{
				Config: settingsConfig([]string{intg1}, "  requestable = false\n"),
				Check:  fake.check(intg1, requestable(false)),
			},
		},
	})
}

// A failed create rolls back the additional maintainers it added, so the next
// attempt does not mistake them for pre-existing ones.
func TestUnitIntegrationSettingsResource_FailedCreateRollsBackAdditional(t *testing.T) {
	fake := setupFake(t, intg1, intg2)
	config := settingsConfig([]string{intg1, intg2}, fmt.Sprintf("  additional_maintainers = [{ type = \"user\", id = %q }]\n", alice))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig:   fake.failNext,
				Config:      config,
				ExpectError: regexp.MustCompile(`simulated\s+failure`),
			},
			{
				PreConfig: func() {
					if err := fake.check(intg1, maintainers())(nil); err != nil {
						t.Errorf("rollback: %v", err)
					}
				},
				Config: config,
				Check:  fake.check(intg1, maintainers("user:"+alice)),
			},
			// alice is owned (not pre-existing), so removing intg1 removes her.
			{
				Config: settingsConfig([]string{intg2}, fmt.Sprintf("  additional_maintainers = [{ type = \"user\", id = %q }]\n", alice)),
				Check:  fake.check(intg1, maintainers()),
			},
		},
	})
}

func TestUnitIntegrationSettingsResource_Validation(t *testing.T) {
	setupFake(t)

	cases := map[string]struct {
		config string
		err    string
	}{
		"no settings": {
			config: settingsConfig([]string{intg1}, ""),
			err:    `At least one of these attributes must be configured`,
		},
		"empty integration_ids": {
			config: settingsConfig([]string{}, "  requestable = true\n"),
			err:    `integration_ids set must contain at least 1 elements`,
		},
		"invalid integration id": {
			config: settingsConfig([]string{"not-a-uuid"}, "  requestable = true\n"),
			err:    `UUID Validate failed`,
		},
		"invalid workflow id": {
			config: settingsConfig([]string{intg1}, "  workflow_id = \"not-a-uuid\"\n"),
			err:    `UUID Validate failed`,
		},
		"both maintainer modes": {
			config: settingsConfig([]string{intg1}, fmt.Sprintf(
				"  maintainers = [{ type = \"user\", id = %q }]\n  additional_maintainers = [{ type = \"user\", id = %q }]\n", alice, bob)),
			err: `cannot be configured together`,
		},
		"invalid maintainer type": {
			config: settingsConfig([]string{intg1}, fmt.Sprintf("  maintainers = [{ type = \"robot\", id = %q }]\n", alice)),
			err:    `value must be one of`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: unitTestProviderFactories,
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
