package integer

import (
	"strconv"

	"github.com/modernice/nice-cms/static/page/field"
)

// Localize return an Option that localizes an Int field.
func Localize(val int, locales ...string) field.Option {
	return field.Localize(strconv.Itoa(val), locales...)
}
