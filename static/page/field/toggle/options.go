package toggle

import "github.com/modernice/cms/static/page/field"

// Localize returns an Option that localizes a Toggle field.
func Localize(val bool, locales ...string) field.Option {
	return field.Localize(toString(val), locales...)
}

func toString(v bool) string {
	if v {
		return "1"
	}
	return "0"
}
