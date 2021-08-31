package page_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/modernice/goes/test"
	"github.com/modernice/nice-cms/static/page"
	"github.com/modernice/nice-cms/static/page/field"
)

func TestPage_Create_emptyName(t *testing.T) {
	p := page.New(uuid.New())
	name := " "
	if err := p.Create(name); !errors.Is(err, page.ErrEmptyName) {
		t.Fatalf("Create should fail with %q; got %q", page.ErrEmptyName, err)
	}
}

func TestPage_Create(t *testing.T) {
	p := page.New(uuid.New())
	name := "foo"
	if err := p.Create(name); err != nil {
		t.Fatalf("Create failed with %q", err)
	}

	if p.Name != name {
		t.Fatalf("Name should be %q; is %q", name, p.Name)
	}

	test.Change(t, p, page.Created, test.EventData(page.CreatedData{Name: name}))
}

func TestPage_Create_withFields(t *testing.T) {
	p := page.New(uuid.New())

	fields := []field.Field{
		field.NewText("foo", "Foo"),
		field.NewInt("bar", 1),
		field.NewToggle("baz", false),
	}

	if err := p.Create("foo", fields...); err != nil {
		t.Fatalf("Create failed with %q", err)
	}

	for _, f := range fields {
		pf, err := p.Field(f.Name)
		if err != nil {
			t.Fatalf("Field(%q) failed with %q", f.Name, err)
		}

		if !pf.Guarded {
			t.Fatalf("Fields passed to Create should be made guarded")
		}
	}

	test.Change(t, p, page.FieldsAdded, test.EventData(page.FieldsAddedData{
		Fields: guarded(fields...),
	}))
}

func TestPage_Add_notCreated(t *testing.T) {
	p := page.New(uuid.New())

	if err := p.Add(field.NewInt("foo", 3)); !errors.Is(err, page.ErrNotCreated) {
		t.Fatalf("Add should fail with %q; got %q", page.ErrNotCreated, err)
	}

	test.NoChange(t, p, page.FieldsAdded)
}

func TestPage_Add(t *testing.T) {
	p := page.New(uuid.New())
	p.Create("foo")

	f := field.NewInt("foo", 3)
	f2 := field.NewInt("bar", 5)
	if err := p.Add(f, f2); err != nil {
		t.Fatalf("Add failed with %q", err)
	}

	added, err := p.Field("foo")
	if err != nil {
		t.Fatalf("Field(%q) failed with %q", "foo", err)
	}

	if !cmp.Equal(added, f) {
		t.Fatalf("added field should be %v; is %v", f, added)
	}

	added, err = p.Field("bar")
	if err != nil {
		t.Fatalf("Field(%q) failed with %q", "bar", err)
	}

	if !cmp.Equal(added, f2) {
		t.Fatalf("added field should be %v; is %v", f2, added)
	}

	test.Change(t, p, page.FieldsAdded, test.EventData(page.FieldsAddedData{Fields: []field.Field{f, f2}}))
}

func TestPage_Add_duplicate(t *testing.T) {
	p := page.New(uuid.New())
	p.Create("foo")

	f := field.NewInt("foo", 3)
	if err := p.Add(f); err != nil {
		t.Fatalf("Add failed with %q", err)
	}

	if err := p.Add(f); !errors.Is(err, page.ErrDuplicateField) {
		t.Fatalf("Add should fail with %q; got %q", page.ErrDuplicateField, err)
	}

	test.Change(t, p, page.FieldsAdded, test.EventData(page.FieldsAddedData{Fields: []field.Field{f}}), test.Exactly(1))
}

func TestPage_Add_duplicates(t *testing.T) {
	p := page.New(uuid.New())
	p.Create("foo")

	fields := []field.Field{
		field.NewText("foo", "Foo"),
		field.NewText("bar", "Bar"),
		field.NewText("foo", "Foo"),
	}

	if err := p.Add(fields...); !errors.Is(err, page.ErrDuplicateField) {
		t.Fatalf("Add should fail with %q; got %q", page.ErrDuplicateField, err)
	}

	test.NoChange(t, p, page.FieldsAdded)
}

func TestPage_Remove_notFound(t *testing.T) {
	p := page.New(uuid.New())

	if err := p.Remove("foo"); !errors.Is(err, page.ErrFieldNotFound) {
		t.Fatalf("Remove should fail with %q; got %q", page.ErrFieldNotFound, err)
	}

	test.NoChange(t, p, page.FieldsRemoved)
}

func TestPage_Remove(t *testing.T) {
	p := page.New(uuid.New())
	p.Create("foo")

	f := field.NewInt("foo", 1)
	f2 := field.NewInt("bar", 1)
	f3 := field.NewInt("baz", 1)
	if err := p.Add(f, f2, f3); err != nil {
		t.Fatalf("Add failed with %q", err)
	}

	if err := p.Remove("foo", "baz"); err != nil {
		t.Fatalf("Remove failed with %q", err)
	}

	for _, name := range []string{"foo", "baz"} {
		if _, err := p.Field(name); !errors.Is(err, page.ErrFieldNotFound) {
			t.Fatalf("Field(%q) should fail with %q; got %q", name, page.ErrFieldNotFound, err)
		}
	}

	if _, err := p.Field("bar"); err != nil {
		t.Fatalf("Field(%q) failed with %q", "bar", err)
	}

	test.Change(t, p, page.FieldsRemoved, test.EventData(page.FieldsRemovedData{Fields: []string{"foo", "baz"}}))
}

func TestPage_Remove_guarded(t *testing.T) {
	p := page.New(uuid.New())
	p.Create("foo", field.NewText("foo", "Foo"))

	if err := p.Remove("foo"); !errors.Is(err, page.ErrGuarded) {
		t.Fatalf("Remove should fail with %q; got %q", page.ErrGuarded, err)
	}
}

func TestPage_Field_notFound(t *testing.T) {
	p := page.New(uuid.New())

	if _, err := p.Field("foo"); !errors.Is(err, page.ErrFieldNotFound) {
		t.Fatalf("Field(%q) should fail with %q; got %q", "foo", page.ErrFieldNotFound, err)
	}
}

func TestPage_Field(t *testing.T) {
	p := page.New(uuid.New())
	p.Create("foo")

	f := field.NewInt("foo", 1)
	if err := p.Add(f); err != nil {
		t.Fatalf("Add failed with %q", err)
	}

	if _, err := p.Field("foo"); err != nil {
		t.Fatalf("Field(%q) failed with %q", "foo", err)
	}
}

func guarded(fields ...field.Field) []field.Field {
	out := make([]field.Field, len(fields))
	for i, f := range fields {
		f.Guarded = true
		out[i] = f
	}
	return out
}
