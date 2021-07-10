package nav_test

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/modernice/cms/static/nav"
)

func TestItem_Type(t *testing.T) {
	label := nav.NewLabel("contact", "Contact")
	staticLink := nav.NewStaticLink("contact", "/contact", "Contact")

	tests := map[*nav.Item]nav.ItemType{
		&label:      nav.Label,
		&staticLink: nav.StaticLink,
	}

	for item, want := range tests {
		t.Run(string(want), func(t *testing.T) {
			if item.Type != want {
				t.Fatalf("Type should be %q; is %q", want, item.Type)
			}
		})
	}
}

func TestItem_Path(t *testing.T) {
	item := nav.NewStaticLink("contact", "/contact", "Contact")

	tests := map[string]string{
		"de": "/contact",
		"en": "/contact",
		"it": "/contact",
		"":   "/contact",
	}

	for locale, want := range tests {
		got := item.Path(locale)
		if got != want {
			t.Fatalf("Path(%q) should return %q; got %q", locale, want, got)
		}
	}

	item = nav.NewStaticLink(
		"contact", "/contact", "Contact",
		nav.LocalePath("de", "/kontakt"),
		nav.LocalePath("it", "/contatta"),
	)

	tests = map[string]string{
		"de": "/kontakt",
		"en": "/contact",
		"it": "/contatta",
		"":   "/contact",
	}

	for locale, want := range tests {
		got := item.Path(locale)
		if got != want {
			t.Fatalf("Path(%q) should return %q; got %q", locale, want, got)
		}
	}
}

func TestItem_Label(t *testing.T) {
	item := nav.NewStaticLink("contact", "/contact", "Contact")

	tests := map[string]string{
		"de":  "Contact",
		"it":  "Contact",
		"":    "Contact",
		"en":  "Contact",
		"abc": "Contact",
	}

	for locale, want := range tests {
		got := item.Label(locale)
		if got != want {
			t.Fatalf("Label(%q) should return %q; got %q", locale, want, got)
		}
	}

	item = nav.NewStaticLink(
		"contact", "/contact", "Contact",
		nav.LocaleLabel("de", "Kontakt"),
		nav.LocaleLabel("it", "Contatta"),
	)

	tests = map[string]string{
		"de":  "Kontakt",
		"it":  "Contatta",
		"":    "Contact",
		"en":  "Contact",
		"abc": "Contact",
	}

	for locale, want := range tests {
		got := item.Label(locale)
		if got != want {
			t.Fatalf("Label(%q) should return %q; got %q", locale, want, got)
		}
	}
}

func TestNewItem_Label_LocalePath(t *testing.T) {
	item := nav.NewItem("foo", nav.Label, nav.LocalePath("de", "/foo"))

	if len(item.Paths) != 0 {
		t.Fatalf("LocalePath option should have no effect with ItemType %q", nav.Label)
	}
}

func TestNewItem_SubTree(t *testing.T) {
	items := []nav.Item{
		nav.NewLabel("foo", "Foo"),
		nav.NewLabel("bar", "Bar"),
		nav.NewLabel("baz", "Baz"),
		nav.NewLabel("qux", "Qux"),
	}

	item := nav.NewItem("foo", nav.Label, nav.SubTree(items...))

	want := nav.NewTree(items...)
	if item.Tree == nil {
		t.Fatalf("item.Tree should be non-nil")
	}

	if !reflect.DeepEqual(item.Tree, want) {
		t.Fatalf("wrong item.Tree\nDiff:\n%s", cmp.Diff(want, item.Tree))
	}
}

func TestNewItem_Initial(t *testing.T) {
	item := nav.NewItem("foo", nav.Label, nav.Initial())

	if item.Initial != true {
		t.Fatalf("Initial should be %v; is %v", true, false)
	}
}
