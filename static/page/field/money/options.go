package money

import (
	"github.com/bojanz/currency"
	"github.com/modernice/cms/internal/money"
	"github.com/modernice/cms/static/page/field"
)

// Localize returns an Option that localizes a Money field.
func Localize(val currency.Amount, locales ...string) field.Option {
	return field.Localize(money.Format(val), locales...)
}
