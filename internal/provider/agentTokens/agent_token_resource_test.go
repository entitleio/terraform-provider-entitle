//go:build acceptance

package agentTokens_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/entitleio/terraform-provider-entitle/internal/testhelpers"
)

func TestAgentTokenResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testhelpers.ProviderConfig + `

resource "entitle_agent_token" "my_agent_token" {
	name = "My Agent Token"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("entitle_agent_token.my_agent_token", "name", "My Agent Token"),

					// Verify dynamic values have any value set in the state.
					resource.TestCheckResourceAttrSet("entitle_agent_token.my_agent_token", "id"),
					resource.TestCheckResourceAttrSet("entitle_agent_token.my_agent_token", "token"),
				),
			},
			// Update testing
			{
				Config: testhelpers.ProviderConfig + `

resource "entitle_agent_token" "my_agent_token" {
	name = "My Agent Token UPDATED"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify
					resource.TestCheckResourceAttr("entitle_agent_token.my_agent_token", "name", "My Agent Token UPDATED"),
				),
			},
		},
	})
}

// TestAgentTokenResourceRotation covers in-place rotation of an agent token:
// any change to the rotation attribute rotates the secret while keeping the
// resource id, no change rotates nothing, and the pending rotation is visible
// in the plan before it is applied.
func TestAgentTokenResourceRotation(t *testing.T) {
	const addr = "entitle_agent_token.rotating_agent"

	var idBefore, tokenBefore string

	capture := func(id, token *string) resource.TestCheckFunc {
		return resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttrWith(addr, "id", func(value string) error {
				*id = value
				return nil
			}),
			resource.TestCheckResourceAttrWith(addr, "token", func(value string) error {
				*token = value
				return nil
			}),
		)
	}

	// assertRotated fails unless the token differs from the last one seen, then
	// records the new one so the next step can be checked against it.
	assertRotated := func(previous *string) resource.TestCheckFunc {
		return resource.TestCheckResourceAttrWith(addr, "token", func(value string) error {
			if value == "" {
				return fmt.Errorf("token is empty after rotation")
			}

			if value == *previous {
				return fmt.Errorf("token was not rotated, it still matches the previous value")
			}

			*previous = value

			return nil
		})
	}

	config := func(rotation string) string {
		if rotation == "" {
			return testhelpers.ProviderConfig + `

resource "entitle_agent_token" "rotating_agent" {
	name = "Rotating Agent Token"
}
`
		}

		return testhelpers.ProviderConfig + fmt.Sprintf(`

resource "entitle_agent_token" "rotating_agent" {
	name     = "Rotating Agent Token"
	rotation = %q
}
`, rotation)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testhelpers.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create without a rotation trigger.
			{
				Config: config(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(addr, "id"),
					resource.TestCheckResourceAttrSet(addr, "token"),
					resource.TestCheckNoResourceAttr(addr, "rotation"),
					capture(&idBefore, &tokenBefore),
				),
			},
			// Setting the attribute for the first time on a resource that
			// already exists is a change like any other, so it rotates. The
			// plan must say so before the apply: an update with a new token.
			{
				Config: config("1"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
						plancheck.ExpectUnknownValue(addr, tfjsonpath.New("token")),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "rotation", "1"),
					resource.TestCheckResourceAttrPtr(addr, "id", &idBefore),
					assertRotated(&tokenBefore),
				),
			},
			// No config change: nothing pending, and in particular no rotation.
			{
				Config:   config("1"),
				PlanOnly: true,
			},
			// Changing the value rotates again, and the id still survives.
			{
				Config: config("2"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
						plancheck.ExpectUnknownValue(addr, tfjsonpath.New("token")),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "rotation", "2"),
					resource.TestCheckResourceAttrPtr(addr, "id", &idBefore),
					assertRotated(&tokenBefore),
				),
			},
			// Removing the attribute must not rotate: the secret is unchanged.
			{
				Config: config(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(addr, "rotation"),
					resource.TestCheckResourceAttrPtr(addr, "id", &idBefore),
					resource.TestCheckResourceAttrPtr(addr, "token", &tokenBefore),
				),
			},
			// A rotated token must not produce a diff on refresh, even though
			// the API never returns the secret on read.
			{
				Config:   config(""),
				PlanOnly: true,
			},
		},
	})
}
