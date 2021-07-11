package meta_test

import (
	"encoding/json"
	"testing"

	"github.com/modernice/cms/static/page/field"
	"github.com/modernice/cms/static/page/field/meta"
)

func TestLocalize(t *testing.T) {
	def := meta.Data{
		Title:       "Foo",
		Description: "Foo says hello.",
	}
	otherVal := meta.Data{
		Title:       "Bar",
		Description: "Bar says hello.",
	}
	tog := field.NewMeta("foo", def, meta.Localize(otherVal, "de", "ch"))

	b, _ := json.Marshal(def)
	wantDef := string(b)
	b, _ = json.Marshal(otherVal)
	wantOther := string(b)

	tests := map[string]string{
		"":    wantDef,
		" ":   wantDef,
		"en":  wantDef,
		"abc": wantDef,
		"de":  wantOther,
		"ch":  wantOther,
	}

	for locale, want := range tests {
		got := tog.Value(locale)
		if got != want {
			t.Fatalf("Value(%q) should return %v; got %v", locale, want, got)
		}
	}
}
