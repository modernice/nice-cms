package integer_test

import (
	"testing"

	"github.com/modernice/cms/static/page/field"
	"github.com/modernice/cms/static/page/field/integer"
)

func TestLocalize(t *testing.T) {
	tog := field.NewInt("foo", 3, integer.Localize(8, "de", "ch"))

	tests := map[string]string{
		"":    "3",
		" ":   "3",
		"en":  "3",
		"abc": "3",
		"de":  "8",
		"ch":  "8",
	}

	for locale, want := range tests {
		got := tog.Value(locale)
		if got != want {
			t.Fatalf("Value(%q) should return %v; got %v", locale, want, got)
		}
	}
}
