package page

import (
	"context"

	"github.com/google/uuid"
	"github.com/modernice/goes/command"
	"github.com/modernice/goes/helper/fanin"
	"github.com/modernice/nice-cms/static/page/field"
)

// Page commands
const (
	CreateCommand       = "cms.static.page.create"
	AddFieldsCommand    = "cms.static.page.add_fields"
	RemoveFieldsCommand = "cms.static.page.remove_fields"
)

type createPayload struct {
	Name   string
	Fields []field.Field
}

// Create returns the command to create the Page with the given UUID.
func Create(id uuid.UUID, name string, fields ...field.Field) command.Command {
	return command.New(CreateCommand, createPayload{
		Name:   name,
		Fields: fields,
	}, command.Aggregate(Aggregate, id))
}

type addFieldsPayload struct {
	Fields []field.Field
}

// AddFields returns the command to add fields to the Page with the given UUID.
func AddFields(id uuid.UUID, fields ...field.Field) command.Command {
	return command.New(AddFieldsCommand, addFieldsPayload{Fields: fields}, command.Aggregate(Aggregate, id))
}

type removeFieldsPayload struct {
	Fields []string
}

// RemoveFields returns the command to remove fields from the Page witht the
// given UUID.
func RemoveFields(id uuid.UUID, fields ...string) command.Command {
	return command.New(RemoveFieldsCommand, removeFieldsPayload{Fields: fields}, command.Aggregate(Aggregate, id))
}

// RegisterCommand registers commands into a registry.
func RegisterCommands(r command.Registry) {
	r.Register(CreateCommand, func() command.Payload { return createPayload{} })
	r.Register(AddFieldsCommand, func() command.Payload { return addFieldsPayload{} })
	r.Register(RemoveFieldsCommand, func() command.Payload { return removeFieldsPayload{} })
}

// HandleCommands handles commands until ctx is canceled.
func HandleCommands(ctx context.Context, bus command.Bus, pages Repository) <-chan error {
	h := command.NewHandler(bus)

	createErrors := h.MustHandle(ctx, CreateCommand, func(ctx command.Context) error {
		cmd := ctx.Command()
		load := cmd.Payload().(createPayload)

		return pages.Use(ctx, cmd.AggregateID(), func(p *Page) error {
			return p.Create(load.Name, load.Fields...)
		})
	})

	addFieldsErrors := h.MustHandle(ctx, AddFieldsCommand, func(ctx command.Context) error {
		cmd := ctx.Command()
		load := cmd.Payload().(addFieldsPayload)

		return pages.Use(ctx, cmd.AggregateID(), func(p *Page) error {
			return p.Add(load.Fields...)
		})
	})

	removeFieldsErrors := h.MustHandle(ctx, RemoveFieldsCommand, func(ctx command.Context) error {
		cmd := ctx.Command()
		load := cmd.Payload().(removeFieldsPayload)

		return pages.Use(ctx, cmd.AggregateID(), func(p *Page) error {
			return p.Remove(load.Fields...)
		})
	})

	return fanin.ErrorsContext(ctx, createErrors, addFieldsErrors, removeFieldsErrors)
}
