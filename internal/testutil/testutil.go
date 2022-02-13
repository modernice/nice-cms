package testutil

import (
	"github.com/modernice/goes/aggregate"
	"github.com/modernice/goes/aggregate/repository"
	"github.com/modernice/goes/codec"
	"github.com/modernice/goes/command"
	"github.com/modernice/goes/command/cmdbus"
	"github.com/modernice/goes/event"
	"github.com/modernice/goes/event/eventbus"
	"github.com/modernice/goes/event/eventstore"
	"github.com/modernice/nice-cms/internal/events"
)

// Goes returns helper functions that return goes components for testing.
func Goes() (
	func() (event.Bus, event.Store, *codec.GobRegistry),
	func() (command.Bus, *codec.Registry),
	func() aggregate.Repository,
) {
	ereg := events.NewRegistry()
	ebus := eventbus.New()
	estore := eventstore.WithBus(eventstore.New(), ebus)

	creg := codec.New()
	cbus := cmdbus.New(ereg.Registry, ebus)

	repo := repository.New(estore)

	return func() (event.Bus, event.Store, *codec.GobRegistry) {
			return ebus, estore, ereg
		}, func() (command.Bus, *codec.Registry) {
			return cbus, creg
		}, func() aggregate.Repository {
			return repo
		}
}
