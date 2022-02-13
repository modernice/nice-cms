package gallery

import (
	"context"

	"github.com/google/uuid"
	"github.com/modernice/goes/codec"
	"github.com/modernice/goes/command"
	"github.com/modernice/goes/helper/streams"
	"github.com/modernice/nice-cms/media"
)

// Gallery commands
const (
	CreateCommand      = "cms.media.image.gallery.create"
	DeleteStackCommand = "cms.media.image.gallery.delete_stack"
	TagStackCommand    = "cms.media.image.gallery.tag_stack"
	UntagStackCommand  = "cms.media.image.gallery.untag_stack"
	RenameStackCommand = "cms.media.image.gallery.rename_stack"
	UpdateStackCommand = "cms.media.image.gallery.update_stack"
	SortCommand        = "cms.media.image.gallery.sort"
)

type createPayload struct {
	Name string
}

// Create returns the command to create a gallery.
func Create(id uuid.UUID, name string) command.Cmd[createPayload] {
	return command.New(CreateCommand, createPayload{Name: name}, command.Aggregate[createPayload](Aggregate, id))
}

type deleteStackPayload struct {
	StackID uuid.UUID
}

// DeleteStack returns the command to delete an image from a gallery.
func DeleteStack(galleryID, stackID uuid.UUID) command.Cmd[deleteStackPayload] {
	return command.New(DeleteStackCommand, deleteStackPayload{StackID: stackID}, command.Aggregate[deleteStackPayload](Aggregate, galleryID))
}

type tagStackPayload struct {
	StackID uuid.UUID
	Tags    []string
}

// TagStack returns the command to tag an image of a gallery.
func TagStack(galleryID, stackID uuid.UUID, tags []string) command.Cmd[tagStackPayload] {
	return command.New(TagStackCommand, tagStackPayload{
		StackID: stackID,
		Tags:    tags,
	}, command.Aggregate[tagStackPayload](Aggregate, galleryID))
}

type untagStackPayload struct {
	StackID uuid.UUID
	Tags    []string
}

// UntagStack returns the command to remove tags from an image of a gallery.
func UntagStack(galleryID, stackID uuid.UUID, tags []string) command.Cmd[untagStackPayload] {
	return command.New(UntagStackCommand, untagStackPayload{
		StackID: stackID,
		Tags:    tags,
	}, command.Aggregate[untagStackPayload](Aggregate, galleryID))
}

type renameStackPayload struct {
	StackID uuid.UUID
	Name    string
}

// RenameStack returns the command to rename an image of a gallery.
func RenameStack(galleryID, stackID uuid.UUID, name string) command.Cmd[renameStackPayload] {
	return command.New(RenameStackCommand, renameStackPayload{
		StackID: stackID,
		Name:    name,
	}, command.Aggregate[renameStackPayload](Aggregate, galleryID))
}

type updateStackPayload struct {
	Stack Stack
}

// UpdateStack returns the command to update a stack of a gallery.
func UpdateStack(galleryID uuid.UUID, stack Stack) command.Cmd[updateStackPayload] {
	return command.New(UpdateStackCommand, updateStackPayload{Stack: stack}, command.Aggregate[updateStackPayload](Aggregate, galleryID))
}

type sortPayload struct {
	Sorting []uuid.UUID
}

// Sort returns the command to sort a gallery.
func Sort(galleryID uuid.UUID, sorting []uuid.UUID) command.Cmd[sortPayload] {
	return command.New(SortCommand, sortPayload{Sorting: sorting}, command.Aggregate[sortPayload](Aggregate, galleryID))
}

// RegisterCommands register the gallery commands into a command registry.
func RegisterCommands(r *codec.GobRegistry) {
	codec.GobRegister[createPayload](r, CreateCommand)
	codec.GobRegister[deleteStackPayload](r, DeleteStackCommand)
	codec.GobRegister[tagStackPayload](r, TagStackCommand)
	codec.GobRegister[untagStackPayload](r, UntagStackCommand)
	codec.GobRegister[renameStackPayload](r, RenameStackCommand)
	codec.GobRegister[updateStackPayload](r, UpdateStackCommand)
	codec.GobRegister[sortPayload](r, SortCommand)
}

// HandleCommands handles commands until ctx is canceled.
func HandleCommands(ctx context.Context, bus command.Bus, galleries Repository, storage media.Storage) <-chan error {
	createErrors := command.MustHandle(ctx, bus, CreateCommand, func(ctx command.Context) error {
		load := ctx.Payload().(createPayload)

		return galleries.Use(ctx, ctx.AggregateID(), func(g *Gallery) error {
			return g.Create(load.Name)
		})
	})

	deleteStackErrors := command.MustHandle(ctx, bus, DeleteStackCommand, func(ctx command.Context) error {
		load := ctx.Payload().(deleteStackPayload)

		return galleries.Use(ctx, ctx.AggregateID(), func(g *Gallery) error {
			s, err := g.Stack(load.StackID)
			if err != nil {
				return err
			}
			return g.Delete(ctx, storage, s)
		})
	})

	tagStackErrors := command.MustHandle(ctx, bus, TagStackCommand, func(ctx command.Context) error {
		load := ctx.Payload().(tagStackPayload)

		return galleries.Use(ctx, ctx.AggregateID(), func(g *Gallery) error {
			s, err := g.Stack(load.StackID)
			if err != nil {
				return err
			}
			_, err = g.Tag(ctx, s, load.Tags...)
			return err
		})
	})

	untagStackErrors := command.MustHandle(ctx, bus, UntagStackCommand, func(ctx command.Context) error {
		load := ctx.Payload().(untagStackPayload)

		return galleries.Use(ctx, ctx.AggregateID(), func(g *Gallery) error {
			s, err := g.Stack(load.StackID)
			if err != nil {
				return err
			}
			_, err = g.Untag(ctx, s, load.Tags...)
			return err
		})
	})

	renameStackErrors := command.MustHandle(ctx, bus, RenameStackCommand, func(ctx command.Context) error {
		load := ctx.Payload().(renameStackPayload)

		return galleries.Use(ctx, ctx.AggregateID(), func(g *Gallery) error {
			_, err := g.RenameStack(ctx, load.StackID, load.Name)
			return err
		})
	})

	updateStackErrors := command.MustHandle(ctx, bus, UpdateStackCommand, func(ctx command.Context) error {
		load := ctx.Payload().(updateStackPayload)

		return galleries.Use(ctx, ctx.AggregateID(), func(g *Gallery) error {
			return g.Update(load.Stack.ID, func(Stack) Stack {
				return load.Stack
			})
		})
	})

	sortErrors := command.MustHandle(ctx, bus, SortCommand, func(ctx command.Context) error {
		load := ctx.Payload().(sortPayload)

		return galleries.Use(ctx, ctx.AggregateID(), func(g *Gallery) error {
			g.Sort(load.Sorting)
			return nil
		})
	})

	return streams.FanInContext(
		ctx,
		createErrors,
		deleteStackErrors,
		tagStackErrors,
		untagStackErrors,
		renameStackErrors,
		updateStackErrors,
		sortErrors,
	)
}
