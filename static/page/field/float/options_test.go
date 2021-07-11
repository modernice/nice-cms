package float_test

import (
	"fmt"
	"testing"

	"github.com/modernice/cms/static/page/field"
	"github.com/modernice/cms/static/page/field/float"
)

func TestLocalize(t *testing.T) {
	tog := field.NewFloat("foo", 3.3, float.Localize(8.5, "de", "ch"))

	tests := map[string]string{
		"":    fmt.Sprintf("%f", 3.3),
		" ":   fmt.Sprintf("%f", 3.3),
		"en":  fmt.Sprintf("%f", 3.3),
		"abc": fmt.Sprintf("%f", 3.3),
		"de":  fmt.Sprintf("%f", 8.5),
		"ch":  fmt.Sprintf("%f", 8.5),
	}

	for locale, want := range tests {
		got := tog.Value(locale)
		if got != want {
			t.Fatalf("Value(%q) should return %v; got %v", locale, want, got)
		}
	}
}
