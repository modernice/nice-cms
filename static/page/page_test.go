package page_test

import (
	"errors"
	"testing"

	"github.com/modernice/cms/static/page"
	"github.com/modernice/goes/test"
)

func TestCreate_emptyName(t *testing.T) {
	name := " "
	if _, err := page.Create(name); !errors.Is(err, page.ErrEmptyName) {
		t.Fatalf("Create should fail with %q; got %q", page.ErrEmptyName, err)
	}
}

func TestCreate(t *testing.T) {
	name := "foo"
	p, err := page.Create(name)
	if err != nil {
		t.Fatalf("Create failed with %q", err)
	}

	if p.Name != name {
		t.Fatalf("Name should be %q; is %q", name, p.Name)
	}

	test.Change(t, p, page.Created, test.WithEventData(page.CreatedData{Name: name}))
}
