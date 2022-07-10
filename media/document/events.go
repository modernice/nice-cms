package document

import (
	"github.com/google/uuid"
	"github.com/modernice/goes/codec"
)

// Shelf events
const (
	ShelfCreated          = "cms.media.document.shelf.created"
	DocumentAdded         = "cms.media.document.shelf.document_added"
	DocumentRemoved       = "cms.media.document.shelf.document_removed"
	DocumentReplaced      = "cms.media.document.shelf.document_replaced"
	DocumentRenamed       = "cms.media.document.shelf.document_renamed"
	DocumentMadeUnique    = "cms.media.document.shelf.document_made_unique"
	DocumentMadeNonUnique = "cms.media.document.shelf.document_made_non_unique"
	DocumentTagged        = "cms.media.document.shelf.document_tagged"
	DocumentUntagged      = "cms.media.document.shelf.document_untagged"
)

// ShelfCreatedData is the event data for the ShelfCreated event.
type ShelfCreatedData struct {
	Name string
}

// DocumentAddedData is the event data for the DocumentAdded event.
type DocumentAddedData struct {
	Document Document
}

// DocumentReplacedData is the event data for the DocumentReplaced event.
type DocumentReplacedData struct {
	Document Document
}

// DocumentRemovedData is the event data for the DocumentRemoved event.
type DocumentRemovedData struct {
	Document    Document
	DeleteError string
}

// DocumentRenamedData is the event data for the DocumentRenamed event.
type DocumentRenamedData struct {
	DocumentID uuid.UUID
	OldName    string
	Name       string
}

// DocumentMadeUniqueData is the event data for the DocumentMadeUnique event.
type DocumentMadeUniqueData struct {
	DocumentID uuid.UUID
	UniqueName string
}

// DocumentMadeNonUniqueData is the event data for the DocumentMadeNonUnique event.
type DocumentMadeNonUniqueData struct {
	DocumentID uuid.UUID
	UniqueName string
}

// DocumentTaggedData is the event data for the DocumentTagged event.
type DocumentTaggedData struct {
	DocumentID uuid.UUID
	Tags       []string
}

// DocumentUntaggedData is the event data for the DocumentUntagged event.
type DocumentUntaggedData struct {
	DocumentID uuid.UUID
	Tags       []string
}

// RegisterEvents registers Shelf events into an event registry.
func RegisterEvents(r codec.Registerer) {
	codec.Register[ShelfCreatedData](r, ShelfCreated)
	codec.Register[DocumentAddedData](r, DocumentAdded)
	codec.Register[DocumentReplacedData](r, DocumentReplaced)
	codec.Register[DocumentRemovedData](r, DocumentRemoved)
	codec.Register[DocumentRenamedData](r, DocumentRenamed)
	codec.Register[DocumentMadeUniqueData](r, DocumentMadeUnique)
	codec.Register[DocumentMadeNonUniqueData](r, DocumentMadeNonUnique)
	codec.Register[DocumentTaggedData](r, DocumentTagged)
	codec.Register[DocumentUntaggedData](r, DocumentUntagged)
}
