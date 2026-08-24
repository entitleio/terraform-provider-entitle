package validators

import (
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework-validators/numbervalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// DurationValidator returns a validator that accepts supported duration values in seconds.
func DurationValidator() validator.Number {
	return numbervalidator.OneOf(
		big.NewFloat(-1),
		big.NewFloat(1800),
		big.NewFloat(3600),
		big.NewFloat(10800),
		big.NewFloat(21600),
		big.NewFloat(43200),
		big.NewFloat(57600),
		big.NewFloat(86400),
		big.NewFloat(259200),
		big.NewFloat(604800),
		big.NewFloat(2628000),
		big.NewFloat(7884000),
		big.NewFloat(15768000),
		big.NewFloat(31536000),
		big.NewFloat(63072000),
	)
}

// DurationSetValidator  returns a validator that accepts set of supported duration values in seconds.
func DurationSetValidator() validator.Set {
	return setvalidator.ValueNumbersAre(DurationValidator())
}
