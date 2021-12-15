package nav_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/modernice/goes/aggregate/repository"
	"github.com/modernice/goes/codec"
	"github.com/modernice/goes/command"
	"github.com/modernice/goes/command/cmdbus"
	"github.com/modernice/goes/command/cmdbus/dispatch"
	"github.com/modernice/goes/event"
	"github.com/modernice/goes/event/eventbus"
	"github.com/modernice/goes/event/eventstore"
	"github.com/modernice/nice-cms/internal/commands"
	"github.com/modernice/nice-cms/internal/discard"
	"github.com/modernice/nice-cms/internal/events"
	"github.com/modernice/nice-cms/static/nav"
)

func TestCreateCmd(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ereg := events.NewRegistry()
	ebus := eventbus.New()
	estore := eventstore.WithBus(eventstore.New(), ebus)
	creg := commands.NewRegistry()
	cbus := cmdbus.New(creg, ereg.Registry, ebus)

	repo := nav.GoesRepository(repository.New(estore))
	lookup, errs := newLookup(t, ctx, ebus, estore)
	panicOn(errs)

	errs = handleCommands(t, ctx, creg.Registry, cbus, repo, lookup)
	panicOn(errs)

	items := []nav.Item{
		nav.NewLabel("foo", "Foo"),
		nav.NewLabel("bar", "Bar"),
		nav.NewLabel("baz", "Baz"),
	}
	cmd := nav.CreateCmd("foo", items...)

	if err := cbus.Dispatch(ctx, cmd, dispatch.Sync()); err != nil {
		t.Fatalf("dispatch command: %v", err)
	}

	<-time.After(20 * time.Millisecond)

	id, ok := lookup.Name("foo")
	if !ok {
		t.Fatalf("Lookup.Name(%q) returned %v", "foo", false)
	}

	if id == uuid.Nil {
		t.Fatalf("Lookup should return a non-nil UUID")
	}
}

func TestCreateCmd_duplicateName(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ereg := events.NewRegistry()
	ebus := eventbus.New()
	estore := eventstore.WithBus(eventstore.New(), ebus)
	creg := commands.NewRegistry()
	cbus := cmdbus.New(creg, ereg.Registry, ebus)

	repo := nav.GoesRepository(repository.New(estore))
	lookup, errs := newLookup(t, ctx, ebus, estore)
	panicOn(errs)

	errs = handleCommands(t, ctx, creg.Registry, cbus, repo, lookup)
	go discard.Errors(errs)

	items := []nav.Item{
		nav.NewLabel("foo", "Foo"),
		nav.NewLabel("bar", "Bar"),
		nav.NewLabel("baz", "Baz"),
	}
	cmd := nav.CreateCmd("foo", items...)

	if err := cbus.Dispatch(ctx, cmd, dispatch.Sync()); err != nil {
		t.Fatalf("dispatch command: %v", err)
	}

	cmd = nav.CreateCmd("foo", items...)

	<-time.After(20 * time.Millisecond)

	if err := cbus.Dispatch(ctx, cmd, dispatch.Sync()); err == nil || !strings.Contains(err.Error(), "name already in use") {
		t.Fatalf("Dispatch should fail with an error containing %q; got %q", "name already in use", err)
	}
}

func newLookup(t *testing.T, ctx context.Context, bus event.Bus, store event.Store) (*nav.Lookup, <-chan error) {
	l := nav.NewLookup()
	errs, err := l.Project(ctx, bus, store)
	if err != nil {
		t.Fatalf("project Lookup: %v", err)
	}
	return l, errs
}

func handleCommands(t *testing.T, ctx context.Context, creg *codec.Registry, cbus command.Bus, repo nav.Repository, lookup *nav.Lookup) <-chan error {
	return nav.HandleCommands(ctx, cbus, repo, lookup)
}

func panicOn(errs <-chan error) {
	go func() {
		for err := range errs {
			panic(err)
		}
	}()
}
