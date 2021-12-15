package document

import (
	"context"

	"github.com/google/uuid"
	"github.com/modernice/goes/codec"
	"github.com/modernice/goes/command"
	"github.com/modernice/goes/helper/fanin"
	"github.com/modernice/nice-cms/media"
)

// Shelf commands.
const (
	CreateShelfCommand   = "cms.media.document.shelf.create"
	RemoveCommand        = "cms.media.document.shelf.remove_document"
	RenameCommand        = "cms.media.document.shelf.rename_document"
	MakeUniqueCommand    = "cms.media.document.shelf.make_document_unique"
	MakeNonUniqueCommand = "cms.media.document.shelf.make_document_non_unique"
	TagCommand           = "cms.media.document.shelf.tag_document"
	UntagCommand         = "cms.media.document.shelf.untag_document"
)

type createShelfPayload struct{ Name string }

// CreateShelf returns the command to create a shelf.
func CreateShelf(id uuid.UUID, name string) command.Command {
	return command.New(CreateShelfCommand, createShelfPayload{Name: name}, command.Aggregate(Aggregate, id))
}

type removePayload struct{ DocumentID uuid.UUID }

// Remove returns the command to remove a document from a shelf.
func Remove(shelfID, documentID uuid.UUID) command.Command {
	return command.New(RemoveCommand, removePayload{DocumentID: documentID}, command.Aggregate(Aggregate, shelfID))
}

type renamePayload struct {
	DocumentID uuid.UUID
	Name       string
}

// Rename returns the command to rename a document in a shelf.
func Rename(shelfID, documentID uuid.UUID, name string) command.Command {
	return command.New(RenameCommand, renamePayload{
		DocumentID: documentID,
		Name:       name,
	}, command.Aggregate(Aggregate, shelfID))
}

type makeUniquePayload struct {
	DocumentID uuid.UUID
	UniqueName string
}

// MakeUnique returns the command to make a document unique within a shelf.
func MakeUnique(shelfID, documentID uuid.UUID, uniqueName string) command.Command {
	return command.New(MakeUniqueCommand, makeUniquePayload{
		DocumentID: documentID,
		UniqueName: uniqueName,
	}, command.Aggregate(Aggregate, shelfID))
}

type makeNonUniquePayload struct{ DocumentID uuid.UUID }

// MakeNonUnique returns the command to remove the uniqueness of a document
// within a shelf.
func MakeNonUnique(shelfID, documentID uuid.UUID) command.Command {
	return command.New(MakeNonUniqueCommand, makeNonUniquePayload{
		DocumentID: documentID,
	}, command.Aggregate(Aggregate, shelfID))
}

type tagPayload struct {
	DocumentID uuid.UUID
	Tags       []string
}

// Tag returns the command to add tags to a document of a shelf.
func Tag(shelfID, documentID uuid.UUID, tags []string) command.Command {
	return command.New(TagCommand, tagPayload{
		DocumentID: documentID,
		Tags:       tags,
	}, command.Aggregate(Aggregate, shelfID))
}

type untagPayload struct {
	DocumentID uuid.UUID
	Tags       []string
}

// Untag returns the command to remove tags from a document of a shelf.
func Untag(shelfID, documentID uuid.UUID, tags []string) command.Command {
	return command.New(UntagCommand, untagPayload{
		DocumentID: documentID,
		Tags:       tags,
	}, command.Aggregate(Aggregate, shelfID))
}

// RegisterCommand registers document commands.
func RegisterCommands(r *codec.GobRegistry) {
	r.GobRegister(CreateShelfCommand, func() interface{} { return createShelfPayload{} })
	r.GobRegister(RemoveCommand, func() interface{} { return removePayload{} })
	r.GobRegister(RenameCommand, func() interface{} { return renamePayload{} })
	r.GobRegister(MakeUniqueCommand, func() interface{} { return makeUniquePayload{} })
	r.GobRegister(MakeNonUniqueCommand, func() interface{} { return makeNonUniquePayload{} })
	r.GobRegister(TagCommand, func() interface{} { return tagPayload{} })
	r.GobRegister(UntagCommand, func() interface{} { return untagPayload{} })
}

// HandleCommand handles commands until ctx is canceled.
func HandleCommands(ctx context.Context, bus command.Bus, shelfs Repository, storage media.Storage) <-chan error {
	h := command.NewHandler(bus)

	createErrors := h.MustHandle(ctx, CreateShelfCommand, func(ctx command.Context) error {
		cmd := ctx.Command()
		load := cmd.Payload().(createShelfPayload)

		return shelfs.Use(ctx, cmd.AggregateID(), func(s *Shelf) error {
			return s.Create(load.Name)
		})
	})

	removeErrors := h.MustHandle(ctx, RemoveCommand, func(ctx command.Context) error {
		cmd := ctx.Command()
		load := cmd.Payload().(removePayload)

		return shelfs.Use(ctx, cmd.AggregateID(), func(s *Shelf) error {
			return s.Remove(ctx, storage, load.DocumentID)
		})
	})

	renameErrors := h.MustHandle(ctx, RenameCommand, func(ctx command.Context) error {
		cmd := ctx.Command()
		load := cmd.Payload().(renamePayload)

		return shelfs.Use(ctx, cmd.AggregateID(), func(s *Shelf) error {
			_, err := s.RenameDocument(load.DocumentID, load.Name)
			return err
		})
	})

	makeUniqueErrors := h.MustHandle(ctx, MakeUniqueCommand, func(ctx command.Context) error {
		cmd := ctx.Command()
		load := cmd.Payload().(makeUniquePayload)

		return shelfs.Use(ctx, cmd.AggregateID(), func(s *Shelf) error {
			_, err := s.MakeUnique(load.DocumentID, load.UniqueName)
			return err
		})
	})

	makeNonUniqueErrors := h.MustHandle(ctx, MakeNonUniqueCommand, func(ctx command.Context) error {
		cmd := ctx.Command()
		load := cmd.Payload().(makeNonUniquePayload)

		return shelfs.Use(ctx, cmd.AggregateID(), func(s *Shelf) error {
			_, err := s.MakeNonUnique(load.DocumentID)
			return err
		})
	})

	tagErrors := h.MustHandle(ctx, TagCommand, func(ctx command.Context) error {
		cmd := ctx.Command()
		load := cmd.Payload().(tagPayload)

		return shelfs.Use(ctx, cmd.AggregateID(), func(s *Shelf) error {
			_, err := s.Tag(load.DocumentID, load.Tags...)
			return err
		})
	})

	untagErrors := h.MustHandle(ctx, UntagCommand, func(ctx command.Context) error {
		cmd := ctx.Command()
		load := cmd.Payload().(untagPayload)

		return shelfs.Use(ctx, cmd.AggregateID(), func(s *Shelf) error {
			_, err := s.Untag(load.DocumentID, load.Tags...)
			return err
		})
	})

	return fanin.ErrorsContext(
		ctx,
		createErrors,
		removeErrors,
		renameErrors,
		makeUniqueErrors,
		makeNonUniqueErrors,
		tagErrors,
		untagErrors,
	)
}
