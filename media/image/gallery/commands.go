package gallery

import (
	"context"

	"github.com/google/uuid"
	"github.com/modernice/goes/codec"
	"github.com/modernice/goes/command"
	"github.com/modernice/goes/helper/fanin"
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
func Create(id uuid.UUID, name string) command.Command {
	return command.New(CreateCommand, createPayload{Name: name}, command.Aggregate(Aggregate, id))
}

type deleteStackPayload struct {
	StackID uuid.UUID
}

// DeleteStack returns the command to delete an image from a gallery.
func DeleteStack(galleryID, stackID uuid.UUID) command.Command {
	return command.New(DeleteStackCommand, deleteStackPayload{StackID: stackID}, command.Aggregate(Aggregate, galleryID))
}

type tagStackPayload struct {
	StackID uuid.UUID
	Tags    []string
}

// TagStack returns the command to tag an image of a gallery.
func TagStack(galleryID, stackID uuid.UUID, tags []string) command.Command {
	return command.New(TagStackCommand, tagStackPayload{
		StackID: stackID,
		Tags:    tags,
	}, command.Aggregate(Aggregate, galleryID))
}

type untagStackPayload struct {
	StackID uuid.UUID
	Tags    []string
}

// UntagStack returns the command to remove tags from an image of a gallery.
func UntagStack(galleryID, stackID uuid.UUID, tags []string) command.Command {
	return command.New(UntagStackCommand, untagStackPayload{
		StackID: stackID,
		Tags:    tags,
	}, command.Aggregate(Aggregate, galleryID))
}

type renameStackPayload struct {
	StackID uuid.UUID
	Name    string
}

// RenameStack returns the command to rename an image of a gallery.
func RenameStack(galleryID, stackID uuid.UUID, name string) command.Command {
	return command.New(RenameStackCommand, renameStackPayload{
		StackID: stackID,
		Name:    name,
	}, command.Aggregate(Aggregate, galleryID))
}

type updateStackPayload struct {
	Stack Stack
}

// UpdateStack returns the command to update a stack of a gallery.
func UpdateStack(galleryID uuid.UUID, stack Stack) command.Command {
	return command.New(UpdateStackCommand, updateStackPayload{Stack: stack}, command.Aggregate(Aggregate, galleryID))
}

type sortPayload struct {
	Sorting []uuid.UUID
}

// Sort returns the command to sort a gallery.
func Sort(galleryID uuid.UUID, sorting []uuid.UUID) command.Command {
	return command.New(SortCommand, sortPayload{Sorting: sorting}, command.Aggregate(Aggregate, galleryID))
}

// RegisterCommands register the gallery commands into a command registry.
func RegisterCommands(r *codec.GobRegistry) {
	r.GobRegister(CreateCommand, func() interface{} { return createPayload{} })
	r.GobRegister(DeleteStackCommand, func() interface{} { return deleteStackPayload{} })
	r.GobRegister(TagStackCommand, func() interface{} { return tagStackPayload{} })
	r.GobRegister(UntagStackCommand, func() interface{} { return untagStackPayload{} })
	r.GobRegister(RenameStackCommand, func() interface{} { return renameStackPayload{} })
	r.GobRegister(UpdateStackCommand, func() interface{} { return updateStackPayload{} })
	r.GobRegister(SortCommand, func() interface{} { return sortPayload{} })
}

// HandleCommands handles commands until ctx is canceled.
func HandleCommands(ctx context.Context, bus command.Bus, galleries Repository, storage media.Storage) <-chan error {
	h := command.NewHandler(bus)

	createErrors := h.MustHandle(ctx, CreateCommand, func(ctx command.Context) error {
		cmd := ctx.Command()
		load := cmd.Payload().(createPayload)

		return galleries.Use(ctx, cmd.AggregateID(), func(g *Gallery) error {
			return g.Create(load.Name)
		})
	})

	deleteStackErrors := h.MustHandle(ctx, DeleteStackCommand, func(ctx command.Context) error {
		cmd := ctx.Command()
		load := cmd.Payload().(deleteStackPayload)

		return galleries.Use(ctx, cmd.AggregateID(), func(g *Gallery) error {
			s, err := g.Stack(load.StackID)
			if err != nil {
				return err
			}
			return g.Delete(ctx, storage, s)
		})
	})

	tagStackErrors := h.MustHandle(ctx, TagStackCommand, func(ctx command.Context) error {
		cmd := ctx.Command()
		load := cmd.Payload().(tagStackPayload)

		return galleries.Use(ctx, cmd.AggregateID(), func(g *Gallery) error {
			s, err := g.Stack(load.StackID)
			if err != nil {
				return err
			}
			_, err = g.Tag(ctx, s, load.Tags...)
			return err
		})
	})

	untagStackErrors := h.MustHandle(ctx, UntagStackCommand, func(ctx command.Context) error {
		cmd := ctx.Command()
		load := cmd.Payload().(untagStackPayload)

		return galleries.Use(ctx, cmd.AggregateID(), func(g *Gallery) error {
			s, err := g.Stack(load.StackID)
			if err != nil {
				return err
			}
			_, err = g.Untag(ctx, s, load.Tags...)
			return err
		})
	})

	renameStackErrors := h.MustHandle(ctx, RenameStackCommand, func(ctx command.Context) error {
		cmd := ctx.Command()
		load := cmd.Payload().(renameStackPayload)

		return galleries.Use(ctx, cmd.AggregateID(), func(g *Gallery) error {
			_, err := g.RenameStack(ctx, load.StackID, load.Name)
			return err
		})
	})

	updateStackErrors := h.MustHandle(ctx, UpdateStackCommand, func(ctx command.Context) error {
		cmd := ctx.Command()
		load := cmd.Payload().(updateStackPayload)

		return galleries.Use(ctx, cmd.AggregateID(), func(g *Gallery) error {
			return g.Update(load.Stack.ID, func(Stack) Stack {
				return load.Stack
			})
		})
	})

	sortErrors := h.MustHandle(ctx, SortCommand, func(ctx command.Context) error {
		cmd := ctx.Command()
		load := cmd.Payload().(sortPayload)

		return galleries.Use(ctx, cmd.AggregateID(), func(g *Gallery) error {
			g.Sort(load.Sorting)
			return nil
		})
	})

	return fanin.ErrorsContext(
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
