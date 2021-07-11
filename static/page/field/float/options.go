package float

import (
	"fmt"

	"github.com/modernice/cms/static/page/field"
)

// Localize returns an Option that localizes a Float field.
func Localize(val float64, locales ...string) field.Option {
	return field.Localize(fmt.Sprintf("%f", val), locales...)
}
