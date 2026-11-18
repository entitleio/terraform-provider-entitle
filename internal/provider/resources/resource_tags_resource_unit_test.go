package resources_test

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/entitleio/terraform-provider-entitle/internal/provider"
)

// Unit tests for entitle_resource_tags. They run the real provider through
// Terraform against an in-memory fake of the Entitle API, so no credentials
// are needed (the Terraform CLI must be available or downloadable).

const (
	tagsR1 = "11111111-1111-1111-1111-111111111111"
	tagsR2 = "22222222-2222-2222-2222-222222222222"
	tagsR3 = "33333333-3333-3333-3333-333333333333"
)

var unitTestProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"entitle": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// chunk_size = 1 forces one bulk request per resource, exercising chunking.
const tagsProviderConfig = `
provider "entitle" {
  bulk {
    chunk_size  = 1
    parallelism = 1
  }
}
`

func tagsConfig(ids []string, tags []string) string {
	return tagsProviderConfig + fmt.Sprintf(`
resource "entitle_resource_tags" "test" {
  resource_ids = %s
  tags         = %s
}
`, hclList(ids), hclList(tags))
}

func hclList(items []string) string {
	quoted := make([]string, len(items))
	for i, it := range items {
		quoted[i] = fmt.Sprintf("%q", it)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func TestUnitResourceTagsResource_Lifecycle(t *testing.T) {
	fake := setupFakeEntitle(t, map[string][]string{
		tagsR1: {"keep"},
		tagsR2: {"keep"},
		tagsR3: {"keep"},
	})
	fake.conflictsLeft = 1 // first submission hits "another bulk operation is running"

	var mark int

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			// Create: tags added to every resource, existing tags preserved,
			// one request per chunk, 409 retried transparently.
			{
				Config: tagsConfig([]string{tagsR1, tagsR2}, []string{"pci", "sox"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("entitle_resource_tags.test", "resource_ids.#", "2"),
					resource.TestCheckTypeSetElemAttr("entitle_resource_tags.test", "resource_ids.*", tagsR1),
					resource.TestCheckTypeSetElemAttr("entitle_resource_tags.test", "resource_ids.*", tagsR2),
					resource.TestCheckResourceAttr("entitle_resource_tags.test", "tags.#", "2"),
					resource.TestCheckNoResourceAttr("entitle_resource_tags.test", "id"),
					fake.checkTags(tagsR1, "keep", "pci", "sox"),
					fake.checkTags(tagsR2, "keep", "pci", "sox"),
					fake.checkTags(tagsR3, "keep"),
					func(*terraform.State) error {
						if n := fake.callCount(); n != 2 {
							return fmt.Errorf("expected 2 accepted bulk requests (chunk_size = 1), got %d", n)
						}
						if left := fake.conflicts(); left != 0 {
							return fmt.Errorf("expected conflicts to be retried, %d left", left)
						}
						return nil
					},
				),
			},
			// Update: r1 leaves (tags removed), r2 stays (only tag diff applied),
			// r3 joins (all tags added); "sox" replaced by "hipaa".
			{
				PreConfig: func() { mark = fake.callCount() },
				Config:    tagsConfig([]string{tagsR2, tagsR3}, []string{"pci", "hipaa"}),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("entitle_resource_tags.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.checkTags(tagsR1, "keep"),
					fake.checkTags(tagsR2, "keep", "pci", "hipaa"),
					fake.checkTags(tagsR3, "keep", "pci", "hipaa"),
					func(*terraform.State) error {
						for _, call := range fake.callsSince(mark) {
							if slices.Contains(call.Ids, tagsR2) && call.Update.Tags.Add != nil &&
								slices.Contains(*call.Update.Tags.Add, "pci") {
								return fmt.Errorf("unchanged tag %q was re-sent for kept resource %s", "pci", tagsR2)
							}
						}
						return nil
					},
				),
			},
			// Drift: a tag removed outside Terraform is detected on refresh and re-added
			// with a request for the drifted resource only.
			{
				PreConfig: func() {
					fake.setTags(tagsR3, "keep", "hipaa")
					mark = fake.callCount()
				},
				Config: tagsConfig([]string{tagsR2, tagsR3}, []string{"pci", "hipaa"}),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("entitle_resource_tags.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.checkTags(tagsR3, "keep", "pci", "hipaa"),
					func(*terraform.State) error {
						calls := fake.callsSince(mark)
						if len(calls) != 1 || !slices.Equal(calls[0].Ids, []string{tagsR3}) {
							return fmt.Errorf("expected a single request for %s, got %+v", tagsR3, calls)
						}
						return nil
					},
				),
			},
			// A resource deleted outside Terraform but still in config: the API rejects
			// the request and the error names the offending ID.
			{
				PreConfig:   func() { fake.deleteResource(tagsR2) },
				Config:      tagsConfig([]string{tagsR2, tagsR3}, []string{"pci", "hipaa"}),
				ExpectError: regexp.MustCompile(tagsR2),
			},
			// Dropping the deleted resource from config converges without changes to r3.
			{
				Config: tagsConfig([]string{tagsR3}, []string{"pci", "hipaa"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("entitle_resource_tags.test", "resource_ids.#", "1"),
					fake.checkTags(tagsR3, "keep", "pci", "hipaa"),
				),
			},
		},
		// Destroy removes only the managed tags.
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			fake.checkTags(tagsR1, "keep"),
			fake.checkTags(tagsR3, "keep"),
		),
	})
}

func TestUnitResourceTagsResource_ToleratesPreexistingAndForeignTags(t *testing.T) {
	fake := setupFakeEntitle(t, map[string][]string{
		tagsR1: {"pci", "other"},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tagsConfig([]string{tagsR1}, []string{"pci", "sox"}),
				Check:  fake.checkTags(tagsR1, "pci", "other", "sox"),
			},
			// Tags added outside Terraform do not cause drift.
			{
				PreConfig: func() { fake.setTags(tagsR1, "pci", "other", "sox", "new") },
				Config:    tagsConfig([]string{tagsR1}, []string{"pci", "sox"}),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
		// "pci" existed before Terraform managed it, so destroy keeps it.
		CheckDestroy: fake.checkTags(tagsR1, "pci", "other", "new"),
	})
}

func TestUnitResourceTagsResource_Validation(t *testing.T) {
	setupFakeEntitle(t, map[string][]string{})

	cases := map[string]struct {
		config string
		err    string
	}{
		"invalid uuid": {
			config: tagsConfig([]string{"not-a-uuid"}, []string{"pci"}),
			err:    `UUID Validate failed`,
		},
		"empty resource_ids": {
			config: tagsConfig([]string{}, []string{"pci"}),
			err:    `resource_ids set must contain at least 1 elements`,
		},
		"empty tags": {
			config: tagsConfig([]string{tagsR1}, []string{}),
			err:    `tags set must contain at least 1 elements`,
		},
		"empty tag value": {
			config: tagsConfig([]string{tagsR1}, []string{""}),
			err:    `string length must be at least 1`,
		},
		"missing tags": {
			config: tagsProviderConfig + fmt.Sprintf(`
resource "entitle_resource_tags" "test" {
  resource_ids = [%q]
}
`, tagsR1),
			err: `The argument "tags" is required`,
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

// Reads go through the list endpoint, 100 IDs per request, never one GET per resource.
func TestUnitResourceTagsResource_ReadsInBatches(t *testing.T) {
	ids := make([]string, 150)
	initial := map[string][]string{}
	for i := range ids {
		ids[i] = fmt.Sprintf("%08x-0000-4000-8000-000000000000", i+1)
		initial[ids[i]] = nil
	}
	fake := setupFakeEntitle(t, initial)

	config := `
provider "entitle" {
  bulk {
    chunk_size  = 1000
    parallelism = 2
  }
}
` + fmt.Sprintf(`
resource "entitle_resource_tags" "test" {
  resource_ids = %s
  tags         = ["pci"]
}
`, hclList(ids))

	expectBatchedReads := func(*terraform.State) error {
		lists, singles := fake.requests()
		if singles != 0 {
			return fmt.Errorf("expected no single-resource GETs, got %d", singles)
		}
		if len(lists) < 2 {
			return fmt.Errorf("expected at least 2 list requests for 150 resources, got %d", len(lists))
		}
		for _, q := range lists {
			if n := len(strings.Split(q.Get("ids"), ",")); n > 100 {
				return fmt.Errorf("list request with %d ids (max 100)", n)
			}
			if q.Has("integrationId") {
				return fmt.Errorf("integrationId must not be sent with ids: %v", q)
			}
		}
		return nil
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("entitle_resource_tags.test", "resource_ids.#", "150"),
					fake.checkTags(ids[0], "pci"),
					fake.checkTags(ids[149], "pci"),
					expectBatchedReads,
				),
			},
			// Drift on one resource among 150 is still detected from the batched reads.
			{
				PreConfig: func() { fake.setTags(ids[120]) },
				Config:    config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("entitle_resource_tags.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.checkTags(ids[120], "pci"),
					expectBatchedReads,
				),
			},
		},
	})
}
