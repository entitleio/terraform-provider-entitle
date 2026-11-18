package resources_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// Tests for ownership tracking: entitle_resource_tags only removes tags it
// added itself, never tags a resource already had.

// Removing a tag from `tags` removes it only where Terraform added it.
func TestUnitResourceTagsResource_Ownership_RemoveTagKeepsPreexisting(t *testing.T) {
	fake := setupFakeEntitle(t, map[string][]string{
		tagsR1: {"pci"}, // pre-existing
		tagsR2: {},
	})
	var mark int

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tagsConfig([]string{tagsR1, tagsR2}, []string{"pci", "sox"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.checkTags(tagsR1, "pci", "sox"),
					fake.checkTags(tagsR2, "pci", "sox"),
				),
			},
			{
				PreConfig: func() { mark = fake.callCount() },
				Config:    tagsConfig([]string{tagsR1, tagsR2}, []string{"sox"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.checkTags(tagsR1, "pci", "sox"),
					fake.checkTags(tagsR2, "sox"),
					expectRemoveCalls(fake, &mark, map[string][]string{tagsR2: {"pci"}}),
				),
			},
		},
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			fake.checkTags(tagsR1, "pci"),
			fake.checkTags(tagsR2),
		),
	})
}

// A tag added to `tags` later is also checked first: where it already exists it
// is recorded as pre-existing and survives destroy.
func TestUnitResourceTagsResource_Ownership_AddedTagChecksPreexisting(t *testing.T) {
	fake := setupFakeEntitle(t, map[string][]string{
		tagsR1: {},
		tagsR2: {},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tagsConfig([]string{tagsR1, tagsR2}, []string{"pci"}),
			},
			{
				PreConfig: func() { fake.setTags(tagsR1, "pci", "hipaa") }, // hipaa added outside Terraform
				Config:    tagsConfig([]string{tagsR1, tagsR2}, []string{"pci", "hipaa"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.checkTags(tagsR1, "pci", "hipaa"),
					fake.checkTags(tagsR2, "pci", "hipaa"),
				),
			},
		},
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			fake.checkTags(tagsR1, "hipaa"),
			fake.checkTags(tagsR2),
		),
	})
}

// A resource added to resource_ids later is also checked first.
func TestUnitResourceTagsResource_Ownership_AddedResourceChecksPreexisting(t *testing.T) {
	fake := setupFakeEntitle(t, map[string][]string{
		tagsR1: {},
		tagsR2: {"pci"},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tagsConfig([]string{tagsR1}, []string{"pci", "sox"}),
			},
			{
				Config: tagsConfig([]string{tagsR1, tagsR2}, []string{"pci", "sox"}),
				Check:  fake.checkTags(tagsR2, "pci", "sox"),
			},
		},
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			fake.checkTags(tagsR1),
			fake.checkTags(tagsR2, "pci"),
		),
	})
}

// Drift must not make Terraform forget which tags it added: after a tag is
// removed outside Terraform and re-added, the tags Terraform added earlier
// are still owned (not mistaken for pre-existing) and are removed on destroy.
func TestUnitResourceTagsResource_Ownership_SurvivesDrift(t *testing.T) {
	fake := setupFakeEntitle(t, map[string][]string{
		tagsR1: {"keep"},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tagsConfig([]string{tagsR1}, []string{"pci", "sox"}),
			},
			{
				PreConfig: func() { fake.setTags(tagsR1, "keep", "pci") }, // "sox" removed outside Terraform
				Config:    tagsConfig([]string{tagsR1}, []string{"pci", "sox"}),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("entitle_resource_tags.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: fake.checkTags(tagsR1, "keep", "pci", "sox"),
			},
		},
		CheckDestroy: fake.checkTags(tagsR1, "keep"),
	})
}

// Drift on a resource that is removed from configuration at the same time.
// Read already dropped the drifted ID from state, so config and state match and
// Terraform plans no change. The resource stays managed in private state and
// is cleaned up by the next apply that changes anything, or by destroy; its
// pre-existing tags are kept either way.
func TestUnitResourceTagsResource_Ownership_DriftedResourceRemovedFromConfig(t *testing.T) {
	fake := setupFakeEntitle(t, map[string][]string{
		tagsR1: {"keep", "pci"}, // "pci" pre-existing
		tagsR2: {},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tagsConfig([]string{tagsR1, tagsR2}, []string{"pci", "sox", "hipaa"}),
			},
			{
				PreConfig: func() { fake.setTags(tagsR1, "keep", "pci", "sox") }, // "hipaa" removed outside Terraform
				Config:    tagsConfig([]string{tagsR2}, []string{"pci", "sox", "hipaa"}),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: fake.checkTags(tagsR1, "keep", "pci", "sox"), // cleanup deferred
			},
			// Any other change triggers an update, which also cleans up r1.
			{
				Config: tagsConfig([]string{tagsR2}, []string{"pci", "sox", "hipaa", "gdpr"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.checkTags(tagsR1, "keep", "pci"),
					fake.checkTags(tagsR2, "pci", "sox", "hipaa", "gdpr"),
				),
			},
		},
	})
}

// Destroy also cleans up a drifted resource that is no longer in configuration.
func TestUnitResourceTagsResource_Ownership_DestroyCleansDriftedResource(t *testing.T) {
	fake := setupFakeEntitle(t, map[string][]string{
		tagsR1: {"keep", "pci"}, // "pci" pre-existing
		tagsR2: {},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tagsConfig([]string{tagsR1, tagsR2}, []string{"pci", "sox", "hipaa"}),
			},
			{
				PreConfig: func() { fake.setTags(tagsR1, "keep", "pci", "sox") },
				Config:    tagsConfig([]string{tagsR2}, []string{"pci", "sox", "hipaa"}),
			},
		},
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			fake.checkTags(tagsR1, "keep", "pci"),
			fake.checkTags(tagsR2),
		),
	})
}

// A pre-existing tag that disappears and is then re-added by Terraform is owned
// by Terraform from then on.
func TestUnitResourceTagsResource_Ownership_ReaddedPreexistingTagBecomesOwned(t *testing.T) {
	fake := setupFakeEntitle(t, map[string][]string{
		tagsR1: {"pci"}, // pre-existing
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tagsConfig([]string{tagsR1}, []string{"pci"}),
			},
			{
				PreConfig: func() { fake.setTags(tagsR1) }, // removed outside Terraform
				Config:    tagsConfig([]string{tagsR1}, []string{"pci"}),
				Check:     fake.checkTags(tagsR1, "pci"),
			},
		},
		CheckDestroy: fake.checkTags(tagsR1),
	})
}

// A failed apply must not make Terraform mistake tags it added part-way for
// pre-existing ones: after the job fails, the next apply finds nothing to do and
// destroy still removes those tags.
func TestUnitResourceTagsResource_Ownership_SurvivesFailedApply(t *testing.T) {
	fake := setupFakeEntitle(t, map[string][]string{
		tagsR1: {},
		tagsR2: {"keep"},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tagsConfig([]string{tagsR1}, []string{"pci"}),
			},
			// The job adding r2 applies its changes but reports failure.
			{
				PreConfig:   fake.failNext,
				Config:      tagsConfig([]string{tagsR1, tagsR2}, []string{"pci"}),
				ExpectError: regexp.MustCompile(`simulated\s+failure`),
			},
			{
				Config: tagsConfig([]string{tagsR1, tagsR2}, []string{"pci"}),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: fake.checkTags(tagsR2, "keep", "pci"),
			},
		},
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			fake.checkTags(tagsR1),
			fake.checkTags(tagsR2, "keep"),
		),
	})
}
