package text_test

import (
	"testing"

	"github.com/modernice/nice-cms/static/page/field"
	"github.com/modernice/nice-cms/static/page/field/text"
)

func TestLocalize(t *testing.T) {
	name := "foo"
	def := "Foo"
	txt := field.NewText(name, def, text.Localize("Bar", "de", "ch"))

	if txt.Name != name {
		t.Fatalf("Name should be %q; is %q", name, txt.Name)
	}

	tests := map[string]string{
		"":    "Foo",
		"en":  "Foo",
		"de":  "Bar",
		"ch":  "Bar",
		"abc": "Foo",
	}

	for locale, want := range tests {
		got := txt.Value(locale)
		if got != want {
			t.Fatalf("Value(%q) should return %q; got %q", locale, want, got)
		}
	}
}
