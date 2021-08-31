package money

import (
	"github.com/modernice/nice-cms/static/page/field"
	"github.com/radical-app/money"
	"github.com/radical-app/money/moneyfmt"
)

// Localize returns an Option that localizes a Money field.
func Localize(val money.Money, locales ...string) field.Option {
	return func(f *field.Field) {
		for _, locale := range locales {
			localized, _ := moneyfmt.Display(val, locale)
			field.Localize(localized, locale)(f)
		}
	}
}
