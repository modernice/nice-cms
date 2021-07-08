package document

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/modernice/goes/event"
	"github.com/modernice/goes/projection"
	"github.com/modernice/goes/projection/schedule"
)

// Lookup provides lookup of Shelf UUIDs. It is thread-safe.
type Lookup struct {
	shelfsMux sync.RWMutex
	shelfs    map[uuid.UUID]*shelfLookup

	shelfNamesMux sync.RWMutex
	shelfNameToID map[string]uuid.UUID
}

// NewLookup returns a new Lookup.
func NewLookup() *Lookup {
	return &Lookup{
		shelfs:        make(map[uuid.UUID]*shelfLookup),
		shelfNameToID: make(map[string]uuid.UUID),
	}
}

// ShelfName returns the UUID of the Shelf with the given name, or false if the
// Lookup has no UUID for name.
func (l *Lookup) ShelfName(name string) (uuid.UUID, bool) {
	return l.shelfName(name)
}

// UniqueName returns the UUID of the Document with the given UniqueName in the
// Shelf with the given UUID, or false if Lookup has no UUID for uniqueName.
func (l *Lookup) UniqueName(shelfID uuid.UUID, uniqueName string) (uuid.UUID, bool) {
	return l.shelf(shelfID).uniqueName(uniqueName)
}

// Project projects the Lookup in a new goroutine and returns a channel of
// asynchronous errors.
func (l *Lookup) Project(ctx context.Context, bus event.Bus, store event.Store, opts ...schedule.ContinuousOption) (<-chan error, error) {
	schedule := schedule.Continuously(bus, store, []string{
		ShelfCreated,
		DocumentAdded,
		DocumentRemoved,
		DocumentMadeUnique,
		DocumentMadeNonUnique,
	}, opts...)

	errs, err := schedule.Subscribe(ctx, l.applyJob)
	if err != nil {
		return nil, fmt.Errorf("subscribe to projection schedule: %w", err)
	}

	go schedule.Trigger(ctx)

	return errs, nil
}

func (l *Lookup) applyJob(job projection.Job) error {
	return job.Apply(job, l)
}

// ApplyEvent applies aggregate events.
func (l *Lookup) ApplyEvent(evt event.Event) {
	switch evt.Name() {
	case ShelfCreated:
		l.shelfCreated(evt)
	case DocumentAdded:
		l.documentAdded(evt)
	case DocumentRemoved:
		l.documentRemoved(evt)
	case DocumentMadeUnique:
		l.documentMadeUnique(evt)
	case DocumentMadeNonUnique:
		l.documentMadeNonUnique(evt)
	}
}

func (l *Lookup) shelfCreated(evt event.Event) {
	data := evt.Data().(ShelfCreatedData)
	l.setShelfName(evt.AggregateID(), data.Name)
}

func (l *Lookup) documentAdded(evt event.Event) {
	data := evt.Data().(DocumentAddedData)
	if data.Document.UniqueName != "" {
		l.setUniqueName(evt.AggregateID(), data.Document.ID, data.Document.UniqueName)
	}
}

func (l *Lookup) documentRemoved(evt event.Event) {
	data := evt.Data().(DocumentRemovedData)
	if data.Document.UniqueName != "" {
		l.removeUniqueName(evt.AggregateID(), data.Document.ID, data.Document.UniqueName)
	}
}

func (l *Lookup) documentMadeUnique(evt event.Event) {
	data := evt.Data().(DocumentMadeUniqueData)
	l.setUniqueName(evt.AggregateID(), data.DocumentID, data.UniqueName)
}

func (l *Lookup) documentMadeNonUnique(evt event.Event) {
	data := evt.Data().(DocumentMadeNonUniqueData)
	l.removeUniqueName(evt.AggregateID(), data.DocumentID, data.UniqueName)
}

func (l *Lookup) setUniqueName(shelfID, documentID uuid.UUID, name string) {
	s := l.shelf(shelfID)
	s.setUniqueName(documentID, name)
}

func (l *Lookup) removeUniqueName(shelfID, documentID uuid.UUID, name string) {
	s := l.shelf(shelfID)
	s.removeUniqueName(documentID, name)
}

func (l *Lookup) shelf(id uuid.UUID) *shelfLookup {
	l.shelfsMux.RLock()
	s, ok := l.shelfs[id]
	l.shelfsMux.RUnlock()
	if ok {
		return s
	}

	s = newShelfLookup()
	l.shelfsMux.Lock()
	defer l.shelfsMux.Unlock()
	l.shelfs[id] = s

	return s
}

func (l *Lookup) setShelfName(id uuid.UUID, name string) {
	l.shelfNamesMux.Lock()
	defer l.shelfNamesMux.Unlock()
	l.shelfNameToID[name] = id
}

func (l *Lookup) shelfName(name string) (uuid.UUID, bool) {
	l.shelfNamesMux.RLock()
	defer l.shelfNamesMux.RUnlock()
	id, ok := l.shelfNameToID[name]
	return id, ok
}

type shelfLookup struct {
	mux            sync.RWMutex
	uniqueNameToID map[string]uuid.UUID
}

func newShelfLookup() *shelfLookup {
	return &shelfLookup{uniqueNameToID: make(map[string]uuid.UUID)}
}

func (l *shelfLookup) setUniqueName(id uuid.UUID, name string) {
	l.mux.Lock()
	defer l.mux.Unlock()
	l.uniqueNameToID[name] = id
}

func (l *shelfLookup) removeUniqueName(id uuid.UUID, name string) {
	l.mux.Lock()
	defer l.mux.Unlock()
	if l.uniqueNameToID[name] == id {
		delete(l.uniqueNameToID, name)
	}
}

func (l *shelfLookup) uniqueName(name string) (uuid.UUID, bool) {
	l.mux.RLock()
	defer l.mux.RUnlock()
	id, ok := l.uniqueNameToID[name]
	return id, ok
}
