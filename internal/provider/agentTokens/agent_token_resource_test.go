//go:build acceptance

package agentTokens_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

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
