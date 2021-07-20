package text

import "github.com/modernice/nice-cms/static/page/field"

// Localize returns an Option that localizes a Text field.
func Localize(val string, locales ...string) field.Option {
	return func(f *field.Field) {
		for _, locale := range locales {
			f.Values[locale] = val
		}
	}
}
