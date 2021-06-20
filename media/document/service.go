package document

//go:generate mockgen -source=service.go -destination=./mock_document/service.go

import (
	"context"
	"errors"
	"sync"

	"github.com/modernice/cms/media"
)

var (
	ErrUnknownDocument = errors.New("unknown document")
)

// Repository stores and retrieves Documents.
type Repository interface {
	// Save stores a Document in the underlying store.
	Save(context.Context, media.Document) error

	// Get returns the Document located at the specified disk and path or
	// ErrUnknownDocument.
	Get(_ context.Context, disk, path string) (media.Document, error)

	// Lookup returns the Document with the specified DocumentName or
	// ErrUnknownDocument.
	Lookup(context.Context, string) (media.Document, error)

	// Delete deletes the given Document fromt the repository.
	Delete(context.Context, media.Document) error
}

type documentRepository struct {
	mux    sync.RWMutex
	docs   map[string]map[string]media.Document // map[DISK]map[PATH]media.Document
	lookup map[string]media.Document            // map[NAME]media.Document
}

// InMemoryRepository return an in-memory Repository.
func InMemoryRepository() Repository {
	return &documentRepository{
		docs:   make(map[string]map[string]media.Document),
		lookup: make(map[string]media.Document),
	}
}

func (r *documentRepository) Save(_ context.Context, doc media.Document) error {
	r.mux.Lock()
	defer r.mux.Unlock()

	disk, ok := r.docs[doc.Disk]
	if !ok {
		disk = make(map[string]media.Document)
		r.docs[doc.Disk] = disk
	}
	disk[doc.Path] = doc

	if doc.DocumentName != "" {
		r.lookup[doc.DocumentName] = doc
	}

	return nil
}

func (r *documentRepository) Get(_ context.Context, diskName, path string) (media.Document, error) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	disk, ok := r.docs[diskName]
	if !ok {
		return media.Document{}, ErrUnknownDocument
	}
	doc, ok := disk[path]
	if !ok {
		return media.Document{}, ErrUnknownDocument
	}
	return doc, nil
}

func (r *documentRepository) Lookup(_ context.Context, name string) (media.Document, error) {
	if name == "" {
		return media.Document{}, ErrUnknownDocument
	}

	r.mux.RLock()
	defer r.mux.RUnlock()

	if doc, ok := r.lookup[name]; ok {
		return doc, nil
	}

	return media.Document{}, ErrUnknownDocument
}

func (r *documentRepository) Delete(ctx context.Context, doc media.Document) error {
	r.mux.Lock()
	defer r.mux.Unlock()
	delete(r.docs, doc.DocumentName)
	return nil
}

type Service struct{}

func NewService() *Service {
	return nil
}
