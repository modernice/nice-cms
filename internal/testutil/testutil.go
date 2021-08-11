package testutil

import (
	"github.com/modernice/goes/aggregate"
	"github.com/modernice/goes/aggregate/repository"
	"github.com/modernice/goes/command"
	"github.com/modernice/goes/command/cmdbus"
	"github.com/modernice/goes/event"
	"github.com/modernice/goes/event/eventbus/chanbus"
	"github.com/modernice/goes/event/eventstore"
	"github.com/modernice/goes/event/eventstore/memstore"
	"github.com/modernice/nice-cms/internal/events"
)

// Goes returns helper functions that return goes components for testing.
func Goes() (
	func() (event.Bus, event.Store, event.Registry),
	func() (command.Bus, command.Registry),
	func() aggregate.Repository,
) {
	ereg := events.NewRegistry()
	ebus := chanbus.New()
	estore := eventstore.WithBus(memstore.New(), ebus)

	creg := command.NewRegistry()
	cbus := cmdbus.New(creg, ereg, ebus)

	repo := repository.New(estore)

	return func() (event.Bus, event.Store, event.Registry) {
			return ebus, estore, ereg
		}, func() (command.Bus, command.Registry) {
			return cbus, creg
		}, func() aggregate.Repository {
			return repo
		}
}
