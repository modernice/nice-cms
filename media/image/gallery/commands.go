package gallery

import (
	"context"

	"github.com/google/uuid"
	"github.com/modernice/goes/command"
	"github.com/modernice/goes/helper/fanin"
	"github.com/modernice/nice-cms/media"
)

// Gallery commands
const (
	CreateCommand      = "cms.media.image.gallery.create"
	DeleteStackCommand = "cms.media.image.gallery.stack_deleted"
	TagStackCommand    = "cms.media.image.gallery.stack_tagged"
	UntagStackCommand  = "cms.media.image.gallery.stack_untagged"
	RenameStackCommand = "cms.media.image.gallery.stack_renamed"
	UpdateStackCommand = "cms.media.image.gallery.stack_updated"
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

func RegisterCommands(r command.Registry) {
	r.Register(CreateCommand, func() command.Payload { return createPayload{} })
	r.Register(DeleteStackCommand, func() command.Payload { return deleteStackPayload{} })
	r.Register(TagStackCommand, func() command.Payload { return tagStackPayload{} })
	r.Register(UntagStackCommand, func() command.Payload { return untagStackPayload{} })
	r.Register(RenameStackCommand, func() command.Payload { return renameStackPayload{} })
	r.Register(UpdateStackCommand, func() command.Payload { return updateStackPayload{} })
}

// HandleCommands handles commands until ctx is canceled.
func HandleCommands(ctx context.Context, h *command.Handler, galleries Repository, storage media.Storage) <-chan error {
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

	return fanin.ErrorsContext(
		ctx,
		createErrors,
		deleteStackErrors,
		tagStackErrors,
		untagStackErrors,
		renameStackErrors,
		updateStackErrors,
	)
}
