//go:build acceptance
// +build acceptance

package testhelpers

import (
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/entitleio/terraform-provider-entitle/internal/provider"
)

var (
	ProviderConfig = fmt.Sprintf(`
provider "entitle" {
  endpoint = "https://api.entitle.io"
  api_key  = "%s"
}
`, os.Getenv("ENTITLE_API_KEY"))
	TestAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"entitle": providerserver.NewProtocol6WithError(provider.New("test")()),
	}
)
