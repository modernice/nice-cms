package document

import (
	"github.com/google/uuid"
	"github.com/modernice/goes/event"
)

// Shelf events
const (
	ShelfCreated          = "cms.media.document.shelf.created"
	DocumentAdded         = "cms.media.document.shelf.document_added"
	DocumentRemoved       = "cms.media.document.shelf.document_removed"
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
func RegisterEvents(r event.Registry) {
	r.Register(ShelfCreated, func() event.Data { return ShelfCreatedData{} })
	r.Register(DocumentAdded, func() event.Data { return DocumentAddedData{} })
	r.Register(DocumentRemoved, func() event.Data { return DocumentRemovedData{} })
	r.Register(DocumentRenamed, func() event.Data { return DocumentRenamedData{} })
	r.Register(DocumentMadeUnique, func() event.Data { return DocumentMadeUniqueData{} })
	r.Register(DocumentMadeNonUnique, func() event.Data { return DocumentMadeNonUniqueData{} })
	r.Register(DocumentTagged, func() event.Data { return DocumentTaggedData{} })
	r.Register(DocumentUntagged, func() event.Data { return DocumentUntaggedData{} })
}
