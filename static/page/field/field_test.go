package field_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/bojanz/currency"
	"github.com/modernice/nice-cms/static/page/field"
	"github.com/modernice/nice-cms/static/page/metadata"
)

func TestNew(t *testing.T) {
	name := "foo"
	typ := field.Text
	def := "Foo"
	f := field.New(name, typ, def)

	if f.Name != name {
		t.Fatalf("Name should be %q; is %q", name, f.Name)
	}

	if f.Type != typ {
		t.Fatalf("Type should be %q; is %q", typ, f.Type)
	}

	if val := f.Value(""); val != def {
		t.Fatalf("Value(%q) should return %q; got %q", "", def, val)
	}

	if f.Guarded {
		t.Fatalf("Guarded should be %v; is %v", false, f.Guarded)
	}
}

func TestField_Value(t *testing.T) {
	name := "foo"
	typ := field.Type("foo-type")
	def := "Foo"
	f := field.New(name, typ, def)

	if f.Name != name {
		t.Fatalf("Name should be %q; is %q", name, f.Name)
	}

	tests := map[string]string{
		"":   def,
		"  ": def,
		"de": def,
		"en": def,
	}

	for locale, want := range tests {
		val := f.Value(locale)
		if val != want {
			t.Fatalf("Value(%q) should return %q; got %q", locale, want, val)
		}
	}
}

func TestNewText(t *testing.T) {
	name := "foo"
	def := "Foo"
	f := field.NewText(name, def)

	if f.Name != name {
		t.Fatalf("Name should be %q; is %q", name, f.Name)
	}

	if f.Type != field.Text {
		t.Fatalf("Type should be %q; is %q", field.Text, f.Type)
	}

	tests := map[string]string{
		"":   def,
		"  ": def,
		"de": def,
		"en": def,
	}

	for locale, want := range tests {
		val := f.Value(locale)
		if val != want {
			t.Fatalf("Value(%q) should return %q; got %q", locale, want, val)
		}
	}
}

func TestNewToggle(t *testing.T) {
	name := "foo"
	def := true
	f := field.NewToggle(name, def)

	if f.Name != name {
		t.Fatalf("Name should be %q; is %q", name, f.Name)
	}

	if f.Type != field.Toggle {
		t.Fatalf("Type should be %q; is %q", field.Toggle, f.Type)
	}

	tests := map[string]string{
		"":   "1",
		" ":  "1",
		"en": "1",
		"de": "1",
	}

	for locale, want := range tests {
		got := f.Value(locale)
		if got != want {
			t.Fatalf("Value(%q) should return %v; got %v", locale, want, got)
		}
	}
}

func TestNewInt(t *testing.T) {
	name := "foo"
	def := 3
	f := field.NewInt(name, def)

	if f.Name != name {
		t.Fatalf("Name should be %q; is %q", name, f.Name)
	}

	if f.Type != field.Int {
		t.Fatalf("Type should be %q; is %q", field.Int, f.Type)
	}

	tests := map[string]string{
		"":   "3",
		" ":  "3",
		"en": "3",
		"de": "3",
	}

	for locale, want := range tests {
		got := f.Value(locale)
		if got != want {
			t.Fatalf("Value(%q) should return %v; got %v", locale, want, got)
		}
	}
}

func TestNewFloat(t *testing.T) {
	name := "foo"
	def := 3.5
	num := field.NewFloat(name, def)

	if num.Name != name {
		t.Fatalf("Name should be %q; is %q", name, num.Name)
	}

	if num.Type != field.Float {
		t.Fatalf("Type should be %q; is %q", field.Float, num.Type)
	}

	tests := map[string]string{
		"":   fmt.Sprintf("%f", 3.5),
		" ":  fmt.Sprintf("%f", 3.5),
		"en": fmt.Sprintf("%f", 3.5),
		"de": fmt.Sprintf("%f", 3.5),
	}

	for locale, want := range tests {
		got := num.Value(locale)
		if got != want {
			t.Fatalf("Value(%q) should return %v; got %v", locale, want, got)
		}
	}
}

func TestNewMoney(t *testing.T) {
	name := "foo"
	def, _ := currency.NewAmount("195.95", "USD")
	f := field.NewMoney(name, def)

	if f.Name != name {
		t.Fatalf("Name should be %q; is %q", name, f.Name)
	}

	if f.Type != field.Money {
		t.Fatalf("Type should be %q; is %q", field.Money, f.Type)
	}

	formatter := currency.NewFormatter(currency.NewLocale("en"))

	tests := map[string]string{
		"":   formatter.Format(def),
		" ":  formatter.Format(def),
		"en": formatter.Format(def),
		"de": formatter.Format(def),
	}

	for locale, want := range tests {
		got := f.Value(locale)
		if got != want {
			t.Fatalf("Value(%q) should return %v; got %v", locale, want, got)
		}
	}
}

func TestNewMeta(t *testing.T) {
	name := "foo"
	def := metadata.Data{
		Title:       "Foo",
		Description: "Foo says hello.",
	}
	f := field.NewMeta(name, def)

	if f.Name != name {
		t.Fatalf("Name should be %q; is %q", name, f.Name)
	}

	if f.Type != field.Meta {
		t.Fatalf("Type should be %q; is %q", field.Meta, f.Type)
	}

	b, err := json.Marshal(def)
	if err != nil {
		t.Fatalf("json encode metadata: %v", err)
	}
	want := string(b)

	tests := map[string]string{
		"":   want,
		" ":  want,
		"en": want,
		"de": want,
	}

	for locale, want := range tests {
		got := f.Value(locale)
		if got != want {
			t.Fatalf("Value(%q) should return %v; got %v", locale, want, got)
		}
	}
}

func TestGuarded(t *testing.T) {
	f := field.New("foo", field.Text, "Foo", field.Guarded())
	if !f.Guarded {
		t.Fatalf("Guarded should be %v; is %v", true, f.Guarded)
	}
}
