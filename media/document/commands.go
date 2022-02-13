package document

import (
	"context"

	"github.com/google/uuid"
	"github.com/modernice/goes/codec"
	"github.com/modernice/goes/command"
	"github.com/modernice/goes/helper/streams"
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
func CreateShelf(id uuid.UUID, name string) command.Cmd[createShelfPayload] {
	return command.New(CreateShelfCommand, createShelfPayload{Name: name}, command.Aggregate[createShelfPayload](Aggregate, id))
}

type removePayload struct{ DocumentID uuid.UUID }

// Remove returns the command to remove a document from a shelf.
func Remove(shelfID, documentID uuid.UUID) command.Cmd[removePayload] {
	return command.New(RemoveCommand, removePayload{DocumentID: documentID}, command.Aggregate[removePayload](Aggregate, shelfID))
}

type renamePayload struct {
	DocumentID uuid.UUID
	Name       string
}

// Rename returns the command to rename a document in a shelf.
func Rename(shelfID, documentID uuid.UUID, name string) command.Cmd[renamePayload] {
	return command.New(RenameCommand, renamePayload{
		DocumentID: documentID,
		Name:       name,
	}, command.Aggregate[renamePayload](Aggregate, shelfID))
}

type makeUniquePayload struct {
	DocumentID uuid.UUID
	UniqueName string
}

// MakeUnique returns the command to make a document unique within a shelf.
func MakeUnique(shelfID, documentID uuid.UUID, uniqueName string) command.Cmd[makeUniquePayload] {
	return command.New(MakeUniqueCommand, makeUniquePayload{
		DocumentID: documentID,
		UniqueName: uniqueName,
	}, command.Aggregate[makeUniquePayload](Aggregate, shelfID))
}

type makeNonUniquePayload struct{ DocumentID uuid.UUID }

// MakeNonUnique returns the command to remove the uniqueness of a document
// within a shelf.
func MakeNonUnique(shelfID, documentID uuid.UUID) command.Cmd[makeNonUniquePayload] {
	return command.New(MakeNonUniqueCommand, makeNonUniquePayload{
		DocumentID: documentID,
	}, command.Aggregate[makeNonUniquePayload](Aggregate, shelfID))
}

type tagPayload struct {
	DocumentID uuid.UUID
	Tags       []string
}

// Tag returns the command to add tags to a document of a shelf.
func Tag(shelfID, documentID uuid.UUID, tags []string) command.Cmd[tagPayload] {
	return command.New(TagCommand, tagPayload{
		DocumentID: documentID,
		Tags:       tags,
	}, command.Aggregate[tagPayload](Aggregate, shelfID))
}

type untagPayload struct {
	DocumentID uuid.UUID
	Tags       []string
}

// Untag returns the command to remove tags from a document of a shelf.
func Untag(shelfID, documentID uuid.UUID, tags []string) command.Cmd[untagPayload] {
	return command.New(UntagCommand, untagPayload{
		DocumentID: documentID,
		Tags:       tags,
	}, command.Aggregate[untagPayload](Aggregate, shelfID))
}

// RegisterCommand registers document commands.
func RegisterCommands(r *codec.GobRegistry) {
	codec.GobRegister[createShelfPayload](r, CreateShelfCommand)
	codec.GobRegister[removePayload](r, RemoveCommand)
	codec.GobRegister[renamePayload](r, RenameCommand)
	codec.GobRegister[makeUniquePayload](r, MakeUniqueCommand)
	codec.GobRegister[makeNonUniquePayload](r, MakeNonUniqueCommand)
	codec.GobRegister[tagPayload](r, TagCommand)
	codec.GobRegister[untagPayload](r, UntagCommand)
}

// HandleCommand handles commands until ctx is canceled.
func HandleCommands(ctx context.Context, bus command.Bus, shelfs Repository, storage media.Storage) <-chan error {
	createErrors := command.MustHandle(ctx, bus, CreateShelfCommand, func(ctx command.ContextOf[createShelfPayload]) error {
		load := ctx.Payload()

		return shelfs.Use(ctx, ctx.AggregateID(), func(s *Shelf) error {
			return s.Create(load.Name)
		})
	})

	removeErrors := command.MustHandle(ctx, bus, RemoveCommand, func(ctx command.ContextOf[removePayload]) error {
		load := ctx.Payload()

		return shelfs.Use(ctx, ctx.AggregateID(), func(s *Shelf) error {
			return s.Remove(ctx, storage, load.DocumentID)
		})
	})

	renameErrors := command.MustHandle(ctx, bus, RenameCommand, func(ctx command.ContextOf[renamePayload]) error {
		load := ctx.Payload()

		return shelfs.Use(ctx, ctx.AggregateID(), func(s *Shelf) error {
			_, err := s.RenameDocument(load.DocumentID, load.Name)
			return err
		})
	})

	makeUniqueErrors := command.MustHandle(ctx, bus, MakeUniqueCommand, func(ctx command.ContextOf[makeUniquePayload]) error {
		load := ctx.Payload()

		return shelfs.Use(ctx, ctx.AggregateID(), func(s *Shelf) error {
			_, err := s.MakeUnique(load.DocumentID, load.UniqueName)
			return err
		})
	})

	makeNonUniqueErrors := command.MustHandle(ctx, bus, MakeNonUniqueCommand, func(ctx command.ContextOf[makeNonUniquePayload]) error {
		load := ctx.Payload()

		return shelfs.Use(ctx, ctx.AggregateID(), func(s *Shelf) error {
			_, err := s.MakeNonUnique(load.DocumentID)
			return err
		})
	})

	tagErrors := command.MustHandle(ctx, bus, TagCommand, func(ctx command.ContextOf[tagPayload]) error {
		load := ctx.Payload()

		return shelfs.Use(ctx, ctx.AggregateID(), func(s *Shelf) error {
			_, err := s.Tag(load.DocumentID, load.Tags...)
			return err
		})
	})

	untagErrors := command.MustHandle(ctx, bus, UntagCommand, func(ctx command.ContextOf[untagPayload]) error {
		load := ctx.Payload()

		return shelfs.Use(ctx, ctx.AggregateID(), func(s *Shelf) error {
			_, err := s.Untag(load.DocumentID, load.Tags...)
			return err
		})
	})

	return streams.FanInContext(
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
