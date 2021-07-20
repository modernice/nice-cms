package nav_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/modernice/goes/aggregate/repository"
	"github.com/modernice/goes/event"
	"github.com/modernice/goes/event/eventbus/chanbus"
	"github.com/modernice/goes/event/eventstore"
	"github.com/modernice/goes/event/eventstore/memstore"
	"github.com/modernice/nice-cms/static/nav"
)

func TestReadCache(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bus := chanbus.New()
	store := eventstore.WithBus(memstore.New(), bus)
	aggregates := repository.New(store)
	repo := newDelayedRepository(nav.GoesRepository(aggregates), 100*time.Millisecond)

	n, _ := nav.Create("foo", nav.NewLabel("foo", "Foo"))
	if err := repo.Save(ctx, n); err != nil {
		t.Fatalf("save Nav: %v", err)
	}

	cache := nav.NewReadCache(repo)

	errs, err := cache.Run(ctx, bus, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Run failed with %q", err)
	}

	go func() {
		for err := range errs {
			panic(err)
		}
	}()

	start := time.Now()
	fetched, err := cache.Fetch(ctx, n.ID)
	if err != nil {
		t.Fatalf("Fetch failed with %q", err)
	}
	dur := time.Since(start)

	if dur < 100*time.Millisecond {
		t.Fatalf("first fetch should take at least 100ms; took %v", dur)
	}

	if !cmp.Equal(fetched, n) {
		t.Fatalf("fetched Nav differs from original:\n\n%s", cmp.Diff(n, fetched))
	}

	start = time.Now()
	fetched, err = cache.Fetch(ctx, n.ID)
	if err != nil {
		t.Fatalf("Fetch failed with %q", err)
	}
	dur = time.Since(start)

	if dur >= 100*time.Millisecond {
		t.Fatalf("second fetch should take less than 100ms; took %v", dur)
	}

	// must do this because gob decodes empty slices as nil...
	fetched.Base.Changes = make([]event.Event, 0)

	if !cmp.Equal(fetched, n) {
		t.Fatalf("fetched Nav differs from original:\n\n%s", cmp.Diff(n, fetched))
	}
}

type delayedRepository struct {
	nav.Repository

	delay time.Duration
}

func newDelayedRepository(repo nav.Repository, delay time.Duration) *delayedRepository {
	return &delayedRepository{
		Repository: repo,
		delay:      delay,
	}
}

func (r *delayedRepository) Fetch(ctx context.Context, id uuid.UUID) (*nav.Nav, error) {
	<-time.After(r.delay)
	return r.Repository.Fetch(ctx, id)
}
