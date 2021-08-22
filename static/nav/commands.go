package nav

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/modernice/goes/command"
)

const (
	// CreateCommand is the command for creating a Nav.
	CreateCommand = "cms.static.nav.create"
)

type createPayload struct {
	Name  string
	Items []Item
}

// CreateCmd returns the command for creating a Nav.
func CreateCmd(name string, items ...Item) command.Command {
	return command.New(CreateCommand, createPayload{
		Name:  name,
		Items: items,
	}, command.Aggregate(Aggregate, uuid.New()))
}

// RegisterCommands register commands into a registry.
func RegisterCommands(r command.Registry) {
	r.Register(CreateCommand, func() command.Payload { return createPayload{} })
}

// HandleCommands handles navigation commands until ctx is canceled. The
// returned error channel is also closed when ctx is canceled. Calls
// RegisterCommands(reg) before subscribing to commands.
func HandleCommands(ctx context.Context, reg command.Registry, bus command.Bus, repo Repository, lookup *Lookup) (<-chan error, error) {
	if reg != nil {
		RegisterCommands(reg)
	}

	h := command.NewHandler(bus)

	errs, err := h.Handle(ctx, CreateCommand, func(ctx command.Context) error {
		cmd := ctx.Command()
		load := cmd.Payload().(createPayload)

		if _, ok := lookup.Name(load.Name); ok {
			return errors.New("name already in use")
		}

		nav, err := CreateWithID(cmd.AggregateID(), load.Name, load.Items...)
		if err != nil {
			return err
		}

		if err := repo.Save(ctx, nav); err != nil {
			return fmt.Errorf("save navigation: %w", err)
		}

		return nil
	})

	return errs, err
}
