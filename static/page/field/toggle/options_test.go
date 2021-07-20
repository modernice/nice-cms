package toggle_test

import (
	"testing"

	"github.com/modernice/nice-cms/static/page/field"
	"github.com/modernice/nice-cms/static/page/field/toggle"
)

func TestLocalize(t *testing.T) {
	tog := field.NewToggle("foo", true, toggle.Localize(false, "de", "ch"))

	tests := map[string]string{
		"":    "1",
		" ":   "1",
		"en":  "1",
		"abc": "1",
		"de":  "0",
		"ch":  "0",
	}

	for locale, want := range tests {
		got := tog.Value(locale)
		if got != want {
			t.Fatalf("Value(%q) should return %v; got %v", locale, want, got)
		}
	}
}
