package meta

import (
	"github.com/modernice/nice-cms/static/page/field"
	"github.com/modernice/nice-cms/static/page/metadata"
)

// Localize returns an Option that localizes a Meta field.
func Localize(data metadata.Data, locales ...string) field.Option {
	str, err := data.JSON()
	if err != nil {
		panic(err)
	}
	return field.Localize(str, locales...)
}
