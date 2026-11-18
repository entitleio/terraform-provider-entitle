package resources_test

import (
	"fmt"
	"slices"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

// Tests for what happens to tags when an ID is removed from resource_ids and
// when the entitle_resource_tags block is removed from configuration.

// expectRemoveCalls checks that the bulk requests sent since *mark only remove
// tags (no additions) and that, per resource, exactly the expected tags were removed.
func expectRemoveCalls(f *fakeEntitle, mark *int, want map[string][]string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		got := map[string][]string{}
		for _, call := range f.callsSince(*mark) {
			if call.Update.Tags == nil {
				return fmt.Errorf("request without tags update: %+v", call)
			}
			if call.Update.Tags.Add != nil {
				return fmt.Errorf("unexpected tag additions %v for %v", *call.Update.Tags.Add, call.Ids)
			}
			for _, id := range call.Ids {
				got[id] = append(got[id], sortedTags(call.Update.Tags.Remove)...)
			}
		}

		if len(got) != len(want) {
			return fmt.Errorf("removals = %v, want %v", got, want)
		}
		for id, tags := range want {
			if g := sortedTags(ptr(got[id])); !slices.Equal(g, sortedTags(&tags)) {
				return fmt.Errorf("removed from %s = %v, want %v", id, g, sortedTags(&tags))
			}
		}
		return nil
	}
}

func ptr[T any](v T) *T { return &v }

func expectNoCalls(f *fakeEntitle, mark *int) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if calls := f.callsSince(*mark); len(calls) != 0 {
			return fmt.Errorf("expected no bulk requests, got %+v", calls)
		}
		return nil
	}
}

func sortedTags(p *[]string) []string {
	if p == nil {
		return nil
	}
	out := slices.Clone(*p)
	sort.Strings(out)
	return out
}

// Removing an ID from resource_ids removes the tags this resource added from
// that resource only. Pre-existing and unmanaged tags stay; the remaining
// resources are not touched.
func TestUnitResourceTagsResource_RemoveID(t *testing.T) {
	fake := setupFakeEntitle(t, map[string][]string{
		tagsR1: {"keep", "pci"}, // "pci" existed before Terraform managed it
		tagsR2: {"keep"},
		tagsR3: {"keep"},
	})
	var mark int

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tagsConfig([]string{tagsR1, tagsR2, tagsR3}, []string{"pci", "sox"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.checkTags(tagsR1, "keep", "pci", "sox"),
					fake.checkTags(tagsR2, "keep", "pci", "sox"),
					fake.checkTags(tagsR3, "keep", "pci", "sox"),
				),
			},
			{
				PreConfig: func() { mark = fake.callCount() },
				Config:    tagsConfig([]string{tagsR2, tagsR3}, []string{"pci", "sox"}),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("entitle_resource_tags.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("entitle_resource_tags.test", "resource_ids.#", "2"),
					// Only "sox" was added by Terraform; the pre-existing "pci" stays.
					fake.checkTags(tagsR1, "keep", "pci"),
					fake.checkTags(tagsR2, "keep", "pci", "sox"),
					fake.checkTags(tagsR3, "keep", "pci", "sox"),
					expectRemoveCalls(fake, &mark, map[string][]string{tagsR1: {"sox"}}),
				),
			},
		},
	})
}

// Removing an ID the bulk endpoint rejects with 404 (deleted, or part of an
// internal integration) does not fail the apply; the ID is skipped.
func TestUnitResourceTagsResource_RemoveID_SkipsRejectedResource(t *testing.T) {
	fake := setupFakeEntitle(t, map[string][]string{
		tagsR1: {"keep"},
		tagsR2: {"keep"},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tagsConfig([]string{tagsR1, tagsR2}, []string{"pci"}),
			},
			{
				PreConfig: func() { fake.hideFromBulk(tagsR1) },
				Config:    tagsConfig([]string{tagsR2}, []string{"pci"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("entitle_resource_tags.test", "resource_ids.#", "1"),
					fake.checkTags(tagsR1, "keep", "pci"), // could not be changed, skipped
					fake.checkTags(tagsR2, "keep", "pci"),
				),
			},
		},
	})
}

// Removing the block from configuration removes the tags this resource added
// from every resource in resource_ids, and nothing else.
func TestUnitResourceTagsResource_RemoveBlock(t *testing.T) {
	fake := setupFakeEntitle(t, map[string][]string{
		tagsR1: {"keep", "pci"}, // "pci" existed before Terraform managed it
		tagsR2: {"keep"},
		tagsR3: {"pci", "sox"}, // not managed by this block
	})
	var mark int

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tagsConfig([]string{tagsR1, tagsR2}, []string{"pci", "sox"}),
			},
			{
				PreConfig: func() { mark = fake.callCount() },
				Config:    tagsProviderConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("entitle_resource_tags.test", plancheck.ResourceActionDestroy),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.checkTags(tagsR1, "keep", "pci"), // pre-existing "pci" stays
					fake.checkTags(tagsR2, "keep"),
					fake.checkTags(tagsR3, "pci", "sox"),
					expectRemoveCalls(fake, &mark, map[string][]string{
						tagsR1: {"sox"},
						tagsR2: {"pci", "sox"},
					}),
				),
			},
		},
	})
}

// On destroy, a resource rejected with 404 is dropped from its chunk and the
// rest of the chunk is still processed.
func TestUnitResourceTagsResource_RemoveBlock_SkipsRejectedResource(t *testing.T) {
	fake := setupFakeEntitle(t, map[string][]string{
		tagsR1: {"keep"},
		tagsR2: {"keep"},
		tagsR3: {"keep"},
	})

	// All IDs in one chunk, so the 404 rejects the whole first request.
	config := func(withResource bool) string {
		cfg := `
provider "entitle" {
  bulk {
    chunk_size = 100
  }
}
`
		if withResource {
			cfg += fmt.Sprintf(`
resource "entitle_resource_tags" "test" {
  resource_ids = %s
  tags         = ["pci"]
}
`, hclList([]string{tagsR1, tagsR2, tagsR3}))
		}
		return cfg
	}

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(true),
			},
			{
				PreConfig: func() { fake.hideFromBulk(tagsR2) },
				Config:    config(false),
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.checkTags(tagsR1, "keep"),
					fake.checkTags(tagsR2, "keep", "pci"), // skipped
					fake.checkTags(tagsR3, "keep"),
				),
			},
		},
	})
}

// A removed block with destroy = false stops managing the tags without
// touching Entitle.
func TestUnitResourceTagsResource_RemovedBlockKeepsTags(t *testing.T) {
	fake := setupFakeEntitle(t, map[string][]string{
		tagsR1: {"keep"},
	})
	var mark int

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_7_0),
		},
		Steps: []resource.TestStep{
			{
				Config: tagsConfig([]string{tagsR1}, []string{"pci"}),
			},
			{
				PreConfig: func() { mark = fake.callCount() },
				Config: tagsProviderConfig + `
removed {
  from = entitle_resource_tags.test

  lifecycle {
    destroy = false
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.checkTags(tagsR1, "keep", "pci"),
					expectNoCalls(fake, &mark),
				),
			},
		},
	})
}

// Two blocks managing the same tag on the same resource. The block created
// second sees the tag as pre-existing, so removing it keeps the tag. Removing
// the block that added the tag strips it; the other block detects the drift,
// re-adds the tag and from then on owns it.
func TestUnitResourceTagsResource_OverlappingBlocks(t *testing.T) {
	fake := setupFakeEntitle(t, map[string][]string{
		tagsR1: {"keep"},
	})

	blockA := fmt.Sprintf(`
resource "entitle_resource_tags" "a" {
  resource_ids = [%q]
  tags         = ["pci"]
}
`, tagsR1)
	blockB := fmt.Sprintf(`
resource "entitle_resource_tags" "b" {
  resource_ids = [%q]
  tags         = ["pci", "sox"]
}
`, tagsR1)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tagsProviderConfig + blockA,
				Check:  fake.checkTags(tagsR1, "keep", "pci"),
			},
			// b is created after a: "pci" is pre-existing for b.
			{
				Config: tagsProviderConfig + blockA + blockB,
				Check:  fake.checkTags(tagsR1, "keep", "pci", "sox"),
			},
			// Removing b removes only "sox"; a stays in sync.
			{
				Config: tagsProviderConfig + blockA,
				Check:  fake.checkTags(tagsR1, "keep", "pci"),
			},
			// Bring b back ("pci" pre-existing again), then remove a, which owns "pci".
			{
				Config: tagsProviderConfig + blockA + blockB,
				Check:  fake.checkTags(tagsR1, "keep", "pci", "sox"),
			},
			{
				Config:             tagsProviderConfig + blockB,
				Check:              fake.checkTags(tagsR1, "keep", "sox"),
				ExpectNonEmptyPlan: true, // b has drifted
			},
			// b re-adds "pci" and now owns it.
			{
				Config: tagsProviderConfig + blockB,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("entitle_resource_tags.b", plancheck.ResourceActionUpdate),
					},
				},
				Check: fake.checkTags(tagsR1, "keep", "pci", "sox"),
			},
		},
		CheckDestroy: fake.checkTags(tagsR1, "keep"),
	})
}
