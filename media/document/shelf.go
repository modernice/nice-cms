package document

//go:generate mockgen -source=shelf.go -destination=./mock_document/shelf.go

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/modernice/goes/aggregate"
	"github.com/modernice/goes/event"
	"github.com/modernice/nice-cms/internal/unique"
	"github.com/modernice/nice-cms/media"
)

// Aggregate is the name of the Document aggregate.
const Aggregate = "cms.media.document.shelf"

var (
	// ErrShelfAlreadyCreated is returned when creating a Shelf that was already created.
	ErrShelfAlreadyCreated = errors.New("shelf already created")

	// ErrShelfNotCreated is returned when trying to use a Shelf that wasn't created yet.
	ErrShelfNotCreated = errors.New("shelf not created")

	// ErrEmptyName is returned when providing a name that is empty or contains only whitespace.
	ErrEmptyName = errors.New("empty name")

	// ErrDuplicateUniqueName is returned when adding a Document to a Shelf with
	// an already taken UniqueName.
	ErrDuplicateUniqueName = errors.New("duplicate unique name")

	// ErrShelfNotFound is returned when a Shelf cannot be found in a Repository.
	ErrShelfNotFound = errors.New("shelf not found")

	// ErrNotFound is returned when a Document cannot be found within a Shelf.
	ErrNotFound = errors.New("document not found")
)

// Repository stores and retrieves Documents.
type Repository interface {
	// Save stores a Shelf in the underlying store.
	Save(context.Context, *Shelf) error

	// Fetch returns the Shelf with the specified UUID or ErrShelfNotFound.
	Fetch(context.Context, uuid.UUID) (*Shelf, error)

	// Delete deletes the given Shelf from the Repository.
	Delete(context.Context, *Shelf) error

	// Use fetches the Shelf with the specified UUID, calls the provided
	// function with that Shelf and then saves the Shelf, but only if the
	// function returns nil.
	Use(context.Context, uuid.UUID, func(*Shelf) error) error
}

// Shelf is a named collection of Documents.
type Shelf struct {
	*aggregate.Base

	Name      string
	Documents []Document
}

// Document is a document in a Shelf.
type Document struct {
	media.Document

	ID uuid.UUID `json:"id"`

	// UniqueName is the unique name of the document. UniqueName may be
	// empty but if it is not, it should be unique.
	UniqueName string `json:"uniqueName"`
}

// NewShelf returns a new Shelf.
func NewShelf(id uuid.UUID) *Shelf {
	return &Shelf{
		Base:      aggregate.New(Aggregate, id),
		Documents: make([]Document, 0),
	}
}

// Document returns the Document with the given UUID or ErrDocumentNotFound.
func (s *Shelf) Document(id uuid.UUID) (Document, error) {
	for _, doc := range s.Documents {
		if doc.ID == id {
			return doc, nil
		}
	}
	return Document{}, ErrNotFound
}

// Find returns the Document with the provided UniqueName or ErrDocumentNotFound.
func (s *Shelf) Find(uniqueName string) (Document, error) {
	if uniqueName == "" {
		return Document{}, ErrEmptyName
	}
	for _, doc := range s.Documents {
		if doc.UniqueName == uniqueName {
			return doc, nil
		}
	}
	return Document{}, ErrNotFound
}

// ApplyEvent applies aggregate events.
func (s *Shelf) ApplyEvent(evt event.Event) {
	switch evt.Name() {
	case ShelfCreated:
		s.create(evt)
	case DocumentAdded:
		s.addDocument(evt)
	case DocumentReplaced:
		s.replaceDocument(evt)
	case DocumentRemoved:
		s.removeDocument(evt)
	case DocumentRenamed:
		s.renameDocument(evt)
	case DocumentMadeUnique:
		s.makeUnique(evt)
	case DocumentMadeNonUnique:
		s.makeNonUnique(evt)
	case DocumentTagged:
		s.tag(evt)
	case DocumentUntagged:
		s.untag(evt)
	}
}

// Create creates the Shelf by giving it a name. If name is empty, ErrEmptyName
// is returned. If the Shelf was already created, ErrShelfAlreadyCreated is
// returned.
func (s *Shelf) Create(name string) error {
	if s.Name != "" {
		return ErrShelfAlreadyCreated
	}
	if name = strings.TrimSpace(name); name == "" {
		return ErrEmptyName
	}
	aggregate.NextEvent(s, ShelfCreated, ShelfCreatedData{Name: name})
	return nil
}

func (s *Shelf) create(evt event.Event) {
	data := evt.Data().(ShelfCreatedData)
	s.Name = data.Name
}

// Add uploads the file in r to storage, adds it as a Document to the Shelf and
// returns the Document. If the Shelf wasn't created yet, ErrShelfNotCreated is
// returned.
//
// If uniqueName is a non-empty string, it must be unique across all existing
// Documents in the Shelf. Documents with a UniqueName can be accessed by their
// unique names. If uniqueName is already in use by another Document,
// ErrDuplicateUniqueName is returned.
func (s *Shelf) Add(ctx context.Context, storage media.Storage, r io.Reader, uniqueName, name, disk, path string) (Document, error) {
	if uniqueName != "" {
		if _, err := s.Find(uniqueName); err == nil {
			return Document{}, ErrDuplicateUniqueName
		}
	}

	doc, err := s.addWithID(ctx, storage, r, uniqueName, name, disk, path, uuid.New())
	if err != nil {
		return doc, err
	}

	aggregate.NextEvent(s, DocumentAdded, DocumentAddedData{Document: doc})

	return s.Document(doc.ID)
}

func (s *Shelf) addWithID(ctx context.Context, storage media.Storage, r io.Reader, uniqueName, name, disk, path string, id uuid.UUID) (Document, error) {
	if err := s.checkCreated(); err != nil {
		return Document{}, err
	}

	doc := media.NewDocument(name, disk, path, 0)
	doc, err := doc.Upload(ctx, r, storage)
	if err != nil {
		return Document{}, fmt.Errorf("upload to storage: %w", err)
	}

	sdoc := Document{
		Document:   doc,
		ID:         id,
		UniqueName: uniqueName,
	}

	return sdoc, nil
}

func (s *Shelf) addDocument(evt event.Event) {
	data := evt.Data().(DocumentAddedData)
	s.Documents = append(s.Documents, data.Document)
}

// Remove deletes the Document with the given UUID from storage and removes it
// from the Shelf. If the Shelf wasn't created yet, ErrShelfNotCreated is
// returned.
//
// No error is returned if the Storage fails to delete the file. Instead, the
// new `DocumentRemoved` aggregate event of the Shelf will contain the deletion
// error.
func (s *Shelf) Remove(ctx context.Context, storage media.Storage, id uuid.UUID) error {
	if err := s.checkCreated(); err != nil {
		return err
	}

	doc, err := s.Document(id)
	if err != nil {
		return err
	}

	deleteError := doc.Delete(ctx, storage)

	data := DocumentRemovedData{Document: doc}

	if deleteError != nil {
		data.DeleteError = deleteError.Error()
	}

	aggregate.NextEvent(s, DocumentRemoved, data)

	return nil
}

func (s *Shelf) removeDocument(evt event.Event) {
	data := evt.Data().(DocumentRemovedData)
	s.remove(data.Document.ID)
}

func (s *Shelf) remove(id uuid.UUID) {
	for i, doc := range s.Documents {
		if doc.ID == id {
			s.Documents = append(s.Documents[:i], s.Documents[i+1:]...)
			return
		}
	}
}

// Replace replaces the document with the given UUID with the document in r.
func (s *Shelf) Replace(ctx context.Context, storage media.Storage, r io.Reader, id uuid.UUID) (Document, error) {
	doc, err := s.Document(id)
	if err != nil {
		return doc, err
	}

	replaced, err := s.addWithID(ctx, storage, r, doc.UniqueName, doc.Name, doc.Disk, doc.Path, doc.ID)
	if err != nil {
		return doc, fmt.Errorf("upload document: %w", err)
	}

	aggregate.NextEvent(s, DocumentReplaced, DocumentReplacedData{Document: replaced})

	return s.Document(replaced.ID)
}

func (s *Shelf) replaceDocument(evt event.Event) {
	data := evt.Data().(DocumentReplacedData)
	s.replace(data.Document.ID, data.Document)
}

// RenameDocument renames the Document with the given UUID. It does not rename
// the UniqueName of a Document; use s.MakeUnique to do that. If the Document
// cannot be found in the Shelf, ErrDocumentNotFound is returned.
func (s *Shelf) RenameDocument(id uuid.UUID, name string) (Document, error) {
	doc, err := s.Document(id)
	if err != nil {
		return doc, err
	}

	aggregate.NextEvent(s, DocumentRenamed, DocumentRenamedData{
		DocumentID: doc.ID,
		OldName:    doc.Name,
		Name:       name,
	})

	return s.Document(doc.ID)
}

func (s *Shelf) renameDocument(evt event.Event) {
	data := evt.Data().(DocumentRenamedData)
	doc, err := s.Document(data.DocumentID)
	if err != nil {
		return
	}
	doc.Name = data.Name
	s.replace(doc.ID, doc)
}

func (s *Shelf) replace(id uuid.UUID, doc Document) {
	doc.ID = id
	for i, sdoc := range s.Documents {
		if sdoc.ID == doc.ID {
			s.Documents[i] = doc
			return
		}
	}
}

// MakeUnique gives the Document with the given UUID the UniqueName uniqueName.
// If the Document cannot be found in the Shelf, ErrDocumentNotFound is returned.
// If uniqueName is an empty string, ErrEmptyName is returned. If uniqueName is
// already taken by another Document, ErrDuplicateUniqueName is returned.
func (s *Shelf) MakeUnique(id uuid.UUID, uniqueName string) (Document, error) {
	doc, err := s.Document(id)
	if err != nil {
		return doc, err
	}

	if uniqueName == "" {
		return doc, ErrEmptyName
	}

	if _, err := s.Find(uniqueName); !errors.Is(err, ErrNotFound) {
		return doc, ErrDuplicateUniqueName
	}

	aggregate.NextEvent(s, DocumentMadeUnique, DocumentMadeUniqueData{
		DocumentID: doc.ID,
		UniqueName: uniqueName,
	})

	return s.Document(id)
}

func (s *Shelf) makeUnique(evt event.Event) {
	data := evt.Data().(DocumentMadeUniqueData)
	doc, err := s.Document(data.DocumentID)
	if err != nil {
		return
	}
	doc.UniqueName = data.UniqueName
	s.replace(doc.ID, doc)
}

// MakeNonUnique removes the UniqueName of the Document with the given UUID. If
// the Document cannot be found in the Shelf, ErrDocumentNotFound is returned.
func (s *Shelf) MakeNonUnique(id uuid.UUID) (Document, error) {
	doc, err := s.Document(id)
	if err != nil {
		return doc, err
	}

	if doc.UniqueName == "" {
		return doc, nil
	}

	aggregate.NextEvent(s, DocumentMadeNonUnique, DocumentMadeNonUniqueData{
		DocumentID: doc.ID,
		UniqueName: doc.UniqueName,
	})

	return s.Document(id)
}

func (s *Shelf) makeNonUnique(evt event.Event) {
	data := evt.Data().(DocumentMadeNonUniqueData)
	doc, err := s.Document(data.DocumentID)
	if err != nil {
		return
	}
	doc.UniqueName = ""
	s.replace(doc.ID, doc)
}

// Tag tags the Document with the given UUID with tags. If the Document cannot
// be found in the Shelf, ErrDocumentNotFound is returned.
func (s *Shelf) Tag(id uuid.UUID, tags ...string) (Document, error) {
	doc, err := s.Document(id)
	if err != nil {
		return doc, err
	}

	tags = unique.Strings(tags...)

	if len(tags) == 0 {
		return doc, nil
	}

	if doc.HasTag(tags...) {
		return doc, nil
	}

	aggregate.NextEvent(s, DocumentTagged, DocumentTaggedData{
		DocumentID: doc.ID,
		Tags:       tags,
	})

	return s.Document(doc.ID)
}

func (s *Shelf) tag(evt event.Event) {
	data := evt.Data().(DocumentTaggedData)
	doc, err := s.Document(data.DocumentID)
	if err != nil {
		return
	}
	doc.Document = doc.WithTag(data.Tags...)
	s.replace(doc.ID, doc)
}

// Untag removes tags from the Document with the given UUID. If the Document
// cannot be found in the Shelf, ErrDocumentNotFound is returned.
func (s *Shelf) Untag(id uuid.UUID, tags ...string) (Document, error) {
	doc, err := s.Document(id)
	if err != nil {
		return doc, err
	}

	tags = unique.Strings(tags...)

	if len(tags) == 0 {
		return doc, nil
	}

	var needsUntag bool
	for _, tag := range tags {
		if doc.HasTag(tag) {
			needsUntag = true
			break
		}
	}

	if !needsUntag {
		return doc, nil
	}

	aggregate.NextEvent(s, DocumentUntagged, DocumentUntaggedData{
		DocumentID: doc.ID,
		Tags:       tags,
	})

	return s.Document(doc.ID)
}

func (s *Shelf) untag(evt event.Event) {
	data := evt.Data().(DocumentUntaggedData)
	doc, err := s.Document(data.DocumentID)
	if err != nil {
		return
	}
	doc.Document = doc.WithoutTag(data.Tags...)
	s.replace(doc.ID, doc)
}

// A SearchOption is used to filter Documents in a search.
type SearchOption func(*searchConfig)

type searchConfig struct {
	names []string
	exprs []*regexp.Regexp
	tags  []string
}

func (cfg searchConfig) allows(doc Document) bool {
	if len(cfg.names) > 0 {
		var found bool
		for _, name := range cfg.names {
			if name == doc.Name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(cfg.exprs) > 0 {
		var found bool
		for _, expr := range cfg.exprs {
			if expr.MatchString(doc.Name) || expr.MatchString(doc.UniqueName) ||
				expr.MatchString(doc.Disk) || expr.MatchString(doc.Path) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(cfg.tags) > 0 {
		var found bool
		for _, tag := range cfg.tags {
			if doc.HasTag(tag) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// ForName returns a SearchOption that filters Documents by their names. A
// Document is included in the result if it has one of the provided names.
func ForName(names ...string) SearchOption {
	return func(cfg *searchConfig) {
		cfg.names = unique.Strings(append(cfg.names, names...)...)
	}
}

// ForRegexp returns a SearchOption that filters Documents by a regular
// expression. A Document is included in the result if expr matches on either
// the Name, UniqueName, Disk or Path of the Document.
func ForRegexp(expr *regexp.Regexp) SearchOption {
	return func(cfg *searchConfig) {
		cfg.exprs = append(cfg.exprs, expr)
	}
}

// ForTag returns a SearchOption that filters Documents by their tags. A
// Document is included in the result if it has at least one of the provided
// tags.
func ForTag(tags ...string) SearchOption {
	return func(cfg *searchConfig) {
		cfg.tags = unique.Strings(append(cfg.tags, tags...)...)
	}
}

// Search returns the Documents in s that are allowed by the provided
// SearchOptions.
func (s *Shelf) Search(opts ...SearchOption) []Document {
	var cfg searchConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	var out []Document
	for _, doc := range s.Documents {
		if cfg.allows(doc) {
			out = append(out, doc)
		}
	}

	return out
}

func (s *Shelf) checkCreated() error {
	if s.Name == "" {
		return ErrShelfNotCreated
	}
	return nil
}

type goesRepository struct {
	repo aggregate.Repository
}

// GoesRepository returns a Repository that uses an aggregate.Repository under
// the hood.
func GoesRepository(repo aggregate.Repository) Repository {
	return &goesRepository{repo: repo}
}

func (r *goesRepository) Save(ctx context.Context, shelf *Shelf) error {
	return r.repo.Save(ctx, shelf)
}

func (r *goesRepository) Fetch(ctx context.Context, id uuid.UUID) (*Shelf, error) {
	shelf := NewShelf(id)
	if err := r.repo.Fetch(ctx, shelf); err != nil {
		return nil, fmt.Errorf("fetch Shelf %q: %w", id, err)
	}
	return shelf, nil
}

func (r *goesRepository) Delete(ctx context.Context, shelf *Shelf) error {
	return r.repo.Delete(ctx, shelf)
}

func (r *goesRepository) Use(ctx context.Context, id uuid.UUID, fn func(*Shelf) error) error {
	shelf, err := r.Fetch(ctx, id)
	if err != nil {
		return err
	}
	if err := fn(shelf); err != nil {
		return err
	}
	if err := r.Save(ctx, shelf); err != nil {
		return fmt.Errorf("save shelf: %w", err)
	}
	return nil
}
