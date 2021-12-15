package nav_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/modernice/goes/aggregate/repository"
	"github.com/modernice/goes/event/eventbus"
	"github.com/modernice/goes/event/eventstore"
	"github.com/modernice/nice-cms/static/nav"
)

func TestLookup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ebus := eventbus.New()
	estore := eventstore.WithBus(eventstore.New(), ebus)
	repo := nav.GoesRepository(repository.New(estore))

	lookup := nav.NewLookup()

	errs, err := lookup.Project(ctx, ebus, estore)
	if err != nil {
		t.Fatalf("run lookup: %v", err)
	}
	go func() {
		for err := range errs {
			panic(err)
		}
	}()

	if _, ok := lookup.Name("foo"); ok {
		t.Fatalf("Name(%q) should return %v", "foo", false)
	}

	n, err := nav.Create("foo")
	if err != nil {
		t.Fatalf("create Nav: %v", err)
	}
	if err := repo.Save(ctx, n); err != nil {
		t.Fatalf("save Nav: %v", err)
	}

	<-time.After(50 * time.Millisecond)

	id, ok := lookup.Name("foo")
	if !ok {
		t.Fatalf("Name(%q) returned %v", "foo", false)
	}

	if id == uuid.Nil {
		t.Fatalf("ID should be non-nil")
	}
}
