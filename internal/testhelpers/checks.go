package testhelpers

// Shared Terraform state checks. No build tag, so both unit and acceptance tests can use them.

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestCheckSetElemAbsent fails if value is an element of the set or list attribute attr
// of the resource or data source name. It is the inverse of resource.TestCheckTypeSetElemAttr.
func TestCheckSetElemAbsent(name, attr, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("%s not found in state", name)
		}

		prefix := attr + "."
		for k, v := range rs.Primary.Attributes {
			if strings.HasPrefix(k, prefix) && k != prefix+"#" && k != prefix+"%" && v == value {
				return fmt.Errorf("%s: %s still contains %q", name, attr, value)
			}
		}

		return nil
	}
}
