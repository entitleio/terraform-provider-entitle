package resources_test

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

// Unit tests for entitle_resource_settings against the fake Entitle API.

const (
	settingsWF1   = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	settingsWF2   = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	settingsGroup = "cccccccc-cccc-cccc-cccc-cccccccccccc"
	settingsAlice = "alice@example.com"
	settingsBob   = "bob@example.com"
	settingsDave  = "dave@example.com"
)

func settingsConfig(ids []string, body string) string {
	return tagsProviderConfig + fmt.Sprintf(`
resource "entitle_resource_settings" "test" {
  resource_ids = %s
%s
}
`, hclList(ids), body)
}

func newResources(ids ...string) map[string][]string {
	out := map[string][]string{}
	for _, id := range ids {
		out[id] = nil
	}
	return out
}

// --- request expectations ----------------------------------------------------

// expectUpdates checks the bulk requests sent since *mark. want maps each
// resource ID to a check of the update it received (exactly once).
func expectUpdates(f *fakeEntitle, mark *int, want map[string]func(u client.BulkResourceUpdate) error) resource.TestCheckFunc {
	return func(*terraform.State) error {
		got := map[string][]client.BulkResourceUpdate{}
		for _, call := range f.callsSince(*mark) {
			for _, id := range call.Ids {
				got[id] = append(got[id], call.Update)
			}
		}
		for id := range got {
			if _, ok := want[id]; !ok {
				return fmt.Errorf("unexpected update for %s: %+v", id, got[id])
			}
		}
		for id, check := range want {
			if len(got[id]) != 1 {
				return fmt.Errorf("resource %s: got %d updates, want 1", id, len(got[id]))
			}
			if err := check(got[id][0]); err != nil {
				return fmt.Errorf("resource %s: %w", id, err)
			}
		}
		return nil
	}
}

// sent describes which top-level fields an update must (and must not) contain.
type sent struct {
	requestable *bool
	workflow    string
	owner       string
	maintainers bool
}

func (s sent) check(u client.BulkResourceUpdate) error {
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
	case u.Tags != nil:
		return fmt.Errorf("tags must never be sent by entitle_resource_settings")
	}
	return nil
}

func boolp(v bool) *bool { return &v }

// --- resource expectations ---------------------------------------------------

func expectRequestable(want bool) func(fakeResource) error {
	return func(r fakeResource) error {
		if r.Requestable != want {
			return fmt.Errorf("requestable = %v, want %v", r.Requestable, want)
		}
		return nil
	}
}

func expectWorkflow(want string) func(fakeResource) error {
	return func(r fakeResource) error {
		if r.WorkflowID != want {
			return fmt.Errorf("workflow = %q, want %q", r.WorkflowID, want)
		}
		return nil
	}
}

func expectOwner(ref string) func(fakeResource) error {
	return func(r fakeResource) error {
		if r.Owner == nil || r.Owner.ID != resolveEntity(ref).ID {
			return fmt.Errorf("owner = %+v, want %s", r.Owner, ref)
		}
		return nil
	}
}

// expectMaintainers checks the exact maintainer set; entries are "type:ref".
func expectMaintainers(want ...string) func(fakeResource) error {
	return func(r fakeResource) error {
		var got, exp []string
		for _, m := range r.Maintainers {
			got = append(got, m.String())
		}
		for _, w := range want {
			typ, ref, _ := strings.Cut(w, ":")
			exp = append(exp, typ+":"+resolveEntity(ref).ID)
		}
		sort.Strings(got)
		sort.Strings(exp)
		if !slices.Equal(got, exp) {
			return fmt.Errorf("maintainers = %v, want %v", got, exp)
		}
		return nil
	}
}

func all(checks ...func(fakeResource) error) func(fakeResource) error {
	return func(r fakeResource) error {
		for _, c := range checks {
			if err := c(r); err != nil {
				return err
			}
		}
		return nil
	}
}

// --- tests -------------------------------------------------------------------

// Only configured attributes are sent; everything else is left untouched.
func TestUnitResourceSettingsResource_CreateSendsOnlyConfigured(t *testing.T) {
	fake := setupFakeEntitle(t, newResources(tagsR1, tagsR2))
	fake.update(tagsR1, func(r *fakeResource) { o := resolveEntity(settingsDave); r.Owner = &o })
	var mark int

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() { mark = fake.callCount() },
				Config: settingsConfig([]string{tagsR1, tagsR2}, fmt.Sprintf(`
  requestable = false
  workflow_id = %q
`, settingsWF1)),
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.checkResource(tagsR1, all(expectRequestable(false), expectWorkflow(settingsWF1), expectOwner(settingsDave))),
					fake.checkResource(tagsR2, all(expectRequestable(false), expectWorkflow(settingsWF1))),
					expectUpdates(fake, &mark, map[string]func(client.BulkResourceUpdate) error{
						tagsR1: sent{requestable: boolp(false), workflow: settingsWF1}.check,
						tagsR2: sent{requestable: boolp(false), workflow: settingsWF1}.check,
					}),
				),
			},
		},
	})
}

// New resources get every configured setting; resources that stay get only what changed.
func TestUnitResourceSettingsResource_UpdateSendsOnlyChanges(t *testing.T) {
	fake := setupFakeEntitle(t, newResources(tagsR1, tagsR2, tagsR3))
	var mark int

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: settingsConfig([]string{tagsR1, tagsR2}, fmt.Sprintf(`
  requestable = false
  workflow_id = %q
`, settingsWF1)),
			},
			{
				PreConfig: func() { mark = fake.callCount() },
				Config: settingsConfig([]string{tagsR1, tagsR2, tagsR3}, fmt.Sprintf(`
  requestable = false
  workflow_id = %q
`, settingsWF2)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("entitle_resource_settings.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.checkResource(tagsR1, all(expectRequestable(false), expectWorkflow(settingsWF2))),
					fake.checkResource(tagsR3, all(expectRequestable(false), expectWorkflow(settingsWF2))),
					expectUpdates(fake, &mark, map[string]func(client.BulkResourceUpdate) error{
						tagsR1: sent{workflow: settingsWF2}.check,
						tagsR2: sent{workflow: settingsWF2}.check,
						tagsR3: sent{requestable: boolp(false), workflow: settingsWF2}.check,
					}),
				),
			},
		},
	})
}

// An attribute removed from configuration is no longer managed: nothing is
// sent, its value stays, and later changes to it are not drift.
func TestUnitResourceSettingsResource_RemovedAttributeBecomesUnmanaged(t *testing.T) {
	fake := setupFakeEntitle(t, newResources(tagsR1))
	var mark int

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: settingsConfig([]string{tagsR1}, fmt.Sprintf(`
  requestable = false
  owner       = %q
`, settingsAlice)),
			},
			{
				PreConfig: func() { mark = fake.callCount() },
				Config:    settingsConfig([]string{tagsR1}, fmt.Sprintf("  owner = %q\n", settingsAlice)),
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.checkResource(tagsR1, all(expectRequestable(false), expectOwner(settingsAlice))),
					expectNoCalls(fake, &mark),
				),
			},
			{
				PreConfig: func() { fake.update(tagsR1, func(r *fakeResource) { r.Requestable = true }) },
				Config:    settingsConfig([]string{tagsR1}, fmt.Sprintf("  owner = %q\n", settingsAlice)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
		},
	})
}

// A managed setting changed outside Terraform is detected on refresh, and
// every configured setting is re-sent to that resource only.
func TestUnitResourceSettingsResource_Drift(t *testing.T) {
	fake := setupFakeEntitle(t, newResources(tagsR1, tagsR2))
	config := settingsConfig([]string{tagsR1, tagsR2}, fmt.Sprintf(`
  requestable = false
  workflow_id = %q
`, settingsWF1))
	var mark int

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{Config: config},
			{
				PreConfig: func() {
					fake.update(tagsR1, func(r *fakeResource) { r.WorkflowID = settingsWF2 })
					mark = fake.callCount()
				},
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("entitle_resource_settings.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.checkResource(tagsR1, expectWorkflow(settingsWF1)),
					expectUpdates(fake, &mark, map[string]func(client.BulkResourceUpdate) error{
						tagsR1: sent{requestable: boolp(false), workflow: settingsWF1}.check,
					}),
				),
			},
		},
	})
}

// The owner can be given as an email or a UUID; both match what the API returns.
func TestUnitResourceSettingsResource_OwnerByEmailOrID(t *testing.T) {
	fake := setupFakeEntitle(t, newResources(tagsR1))
	bobID := resolveEntity(settingsBob).ID

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: settingsConfig([]string{tagsR1}, fmt.Sprintf("  owner = %q\n", strings.ToUpper(settingsAlice))),
				Check:  fake.checkResource(tagsR1, expectOwner(settingsAlice)),
			},
			{
				Config: settingsConfig([]string{tagsR1}, fmt.Sprintf("  owner = %q\n", bobID)),
				Check:  fake.checkResource(tagsR1, expectOwner(settingsBob)),
			},
		},
	})
}

// maintainers replaces the whole list; extra maintainers added outside
// Terraform are drift; an empty list removes all maintainers.
func TestUnitResourceSettingsResource_MaintainersReplace(t *testing.T) {
	fake := setupFakeEntitle(t, newResources(tagsR1))
	fake.update(tagsR1, func(r *fakeResource) {
		r.Maintainers = []fakeMaintainer{{Type: "user", fakeEntity: resolveEntity(settingsDave)}}
	})
	config := settingsConfig([]string{tagsR1}, fmt.Sprintf(`
  maintainers = [
    { type = "user", id = %q },
    { type = "group", id = %q },
  ]
`, settingsAlice, settingsGroup))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  fake.checkResource(tagsR1, expectMaintainers("user:"+settingsAlice, "group:"+settingsGroup)),
			},
			{
				PreConfig: func() {
					fake.update(tagsR1, func(r *fakeResource) {
						r.Maintainers = append(r.Maintainers, fakeMaintainer{Type: "user", fakeEntity: resolveEntity(settingsBob)})
					})
				},
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("entitle_resource_settings.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: fake.checkResource(tagsR1, expectMaintainers("user:"+settingsAlice, "group:"+settingsGroup)),
			},
			{
				Config: settingsConfig([]string{tagsR1}, "  maintainers = []\n"),
				Check:  fake.checkResource(tagsR1, expectMaintainers()),
			},
		},
	})
}

// additional_maintainers works like entitle_resource_tags: it adds, keeps
// existing maintainers, and removes only the ones it added.
func TestUnitResourceSettingsResource_AdditionalMaintainers(t *testing.T) {
	fake := setupFakeEntitle(t, newResources(tagsR1, tagsR2))
	fake.update(tagsR1, func(r *fakeResource) {
		r.Maintainers = []fakeMaintainer{{Type: "user", fakeEntity: resolveEntity(settingsDave)}} // pre-existing
	})
	additional := func(refs ...string) string {
		var items []string
		for _, ref := range refs {
			items = append(items, fmt.Sprintf("    { type = \"user\", id = %q },", ref))
		}
		return "  additional_maintainers = [\n" + strings.Join(items, "\n") + "\n  ]\n"
	}
	var mark int

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: settingsConfig([]string{tagsR1, tagsR2}, additional(settingsDave, settingsAlice)),
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.checkResource(tagsR1, expectMaintainers("user:"+settingsDave, "user:"+settingsAlice)),
					fake.checkResource(tagsR2, expectMaintainers("user:"+settingsDave, "user:"+settingsAlice)),
				),
			},
			// Maintainers added outside Terraform are not drift.
			{
				PreConfig: func() {
					fake.update(tagsR2, func(r *fakeResource) {
						r.Maintainers = append(r.Maintainers, fakeMaintainer{Type: "user", fakeEntity: resolveEntity(settingsBob)})
					})
				},
				Config: settingsConfig([]string{tagsR1, tagsR2}, additional(settingsDave, settingsAlice)),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			// Taking dave out of the list removes him only where Terraform added him.
			{
				PreConfig: func() { mark = fake.callCount() },
				Config:    settingsConfig([]string{tagsR1, tagsR2}, additional(settingsAlice)),
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.checkResource(tagsR1, expectMaintainers("user:"+settingsDave, "user:"+settingsAlice)),
					fake.checkResource(tagsR2, expectMaintainers("user:"+settingsAlice, "user:"+settingsBob)),
					expectUpdates(fake, &mark, map[string]func(client.BulkResourceUpdate) error{
						tagsR2: sent{maintainers: true}.check,
					}),
				),
			},
			// Removing r2 from resource_ids removes the maintainers Terraform added to it.
			{
				Config: settingsConfig([]string{tagsR1}, additional(settingsAlice)),
				Check:  fake.checkResource(tagsR2, expectMaintainers("user:"+settingsBob)),
			},
		},
		// Destroy leaves everything in place, including additional maintainers.
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			fake.checkResource(tagsR1, expectMaintainers("user:"+settingsDave, "user:"+settingsAlice)),
		),
	})
}

// Switching from additional_maintainers to maintainers sends a single replace
// (the fake API fails the test if replace is combined with add/remove).
func TestUnitResourceSettingsResource_SwitchToReplace(t *testing.T) {
	fake := setupFakeEntitle(t, newResources(tagsR1))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: settingsConfig([]string{tagsR1}, fmt.Sprintf("  additional_maintainers = [{ type = \"user\", id = %q }]\n", settingsAlice)),
			},
			{
				Config: settingsConfig([]string{tagsR1}, fmt.Sprintf("  maintainers = [{ type = \"user\", id = %q }]\n", settingsBob)),
				Check:  fake.checkResource(tagsR1, expectMaintainers("user:"+settingsBob)),
			},
		},
	})
}

// Removing an ID or destroying leaves the settings in place and sends nothing.
func TestUnitResourceSettingsResource_RemoveAndDestroyLeaveSettings(t *testing.T) {
	fake := setupFakeEntitle(t, newResources(tagsR1, tagsR2))
	body := fmt.Sprintf(`
  requestable = false
  workflow_id = %q
`, settingsWF1)
	var mark int

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{Config: settingsConfig([]string{tagsR1, tagsR2}, body)},
			{
				PreConfig: func() { mark = fake.callCount() },
				Config:    settingsConfig([]string{tagsR1}, body),
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.checkResource(tagsR2, all(expectRequestable(false), expectWorkflow(settingsWF1))),
					expectNoCalls(fake, &mark),
					func(*terraform.State) error { mark = fake.callCount(); return nil },
				),
			},
		},
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			fake.checkResource(tagsR1, all(expectRequestable(false), expectWorkflow(settingsWF1))),
			expectNoCalls(fake, &mark),
		),
	})
}

// A failed create rolls back the additional maintainers it added, so the next
// attempt does not mistake them for pre-existing ones.
func TestUnitResourceSettingsResource_FailedCreateRollsBackAdditional(t *testing.T) {
	fake := setupFakeEntitle(t, newResources(tagsR1, tagsR2))
	config := settingsConfig([]string{tagsR1, tagsR2}, fmt.Sprintf("  additional_maintainers = [{ type = \"user\", id = %q }]\n", settingsAlice))

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
					// The failed job had applied its change; the rollback must have undone it.
					if err := fake.checkResource(tagsR1, expectMaintainers())(nil); err != nil {
						t.Errorf("rollback: %v", err)
					}
				},
				Config: config,
				Check:  fake.checkResource(tagsR1, expectMaintainers("user:"+settingsAlice)),
			},
			// alice is owned (not pre-existing), so removing r1 removes her.
			{
				Config: settingsConfig([]string{tagsR2}, fmt.Sprintf("  additional_maintainers = [{ type = \"user\", id = %q }]\n", settingsAlice)),
				Check:  fake.checkResource(tagsR1, expectMaintainers()),
			},
		},
	})
}

func TestUnitResourceSettingsResource_Validation(t *testing.T) {
	setupFakeEntitle(t, newResources())

	cases := map[string]struct {
		body string
		err  string
	}{
		"no settings": {
			body: "",
			err:  `At least one of these attributes must be configured`,
		},
		"both maintainer modes": {
			body: fmt.Sprintf("  maintainers = [{ type = \"user\", id = %q }]\n  additional_maintainers = [{ type = \"user\", id = %q }]\n", settingsAlice, settingsBob),
			err:  `cannot be configured together`,
		},
		"invalid workflow id": {
			body: "  workflow_id = \"not-a-uuid\"\n",
			err:  `UUID Validate failed`,
		},
		"invalid maintainer type": {
			body: fmt.Sprintf("  maintainers = [{ type = \"robot\", id = %q }]\n", settingsAlice),
			err:  `value must be one of`,
		},
		"empty additional maintainers": {
			body: "  additional_maintainers = []\n",
			err:  `must contain at least 1 elements`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: unitTestProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      settingsConfig([]string{tagsR1}, tc.body),
						PlanOnly:    true,
						ExpectError: regexp.MustCompile(strings.ReplaceAll(tc.err, " ", `\s+`)),
					},
				},
			})
		})
	}
}
