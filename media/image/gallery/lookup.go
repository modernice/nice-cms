package gallery

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/modernice/goes/event"
	"github.com/modernice/goes/project"
)

// Lookup provides lookup of Gallery UUIDs. It is thread-safe.
type Lookup struct {
	mux       sync.RWMutex
	galleries map[uuid.UUID]*galleryLookup
}

// NewLookup returns a new Lookup.
func NewLookup() *Lookup {
	return &Lookup{galleries: make(map[uuid.UUID]*galleryLookup)}
}

// Name returns the UUID of the Stack with the given Name in the Gallery with
// the given UUID, or false if Lookup has no UUID for name.
func (l *Lookup) Name(galleryID uuid.UUID, name string) (uuid.UUID, bool) {
	return l.gallery(galleryID).name(name)
}

// Project projects the Lookup in a new goroutine and returns a channel of
// asynchronous errors.
func (l *Lookup) Project(ctx context.Context, bus event.Bus, store event.Store, opts ...project.ContinuousOption) (<-chan error, error) {
	schedule := project.Continuously(bus, store, []string{
		ImageUploaded,
		StackDeleted,
		StackRenamed,
		StackUpdated, // TODO: remove event type; it's too broad
	}, opts...)

	errs, err := schedule.Subscribe(ctx, l.applyJob)
	if err != nil {
		return nil, fmt.Errorf("subscribe to projection schedule: %w", err)
	}

	go schedule.Trigger(ctx)

	return errs, nil
}

func (l *Lookup) applyJob(job project.Job) error {
	return job.Apply(job.Context(), l)
}

// ApplyEvent applies aggregate events.
func (l *Lookup) ApplyEvent(evt event.Event) {
	switch evt.Name() {
	case ImageUploaded:
		l.imageUploaded(evt)
	case StackDeleted:
		l.stackDeleted(evt)
	case StackRenamed:
		l.stackRenamed(evt)
	case StackUpdated:
		l.stackUpdated(evt)
	}
}

func (l *Lookup) imageUploaded(evt event.Event) {
	data := evt.Data().(ImageUploadedData)
	l.setName(evt.AggregateID(), data.Stack.ID, data.Stack.Original().Name)
}

func (l *Lookup) stackDeleted(evt event.Event) {
	data := evt.Data().(StackDeletedData)
	l.removeName(evt.AggregateID(), data.Stack.ID, data.Stack.Original().Name)
}

func (l *Lookup) stackRenamed(evt event.Event) {
	data := evt.Data().(StackRenamedData)
	l.removeName(evt.AggregateID(), data.StackID, data.OldName)
	l.setName(evt.AggregateID(), data.StackID, data.Name)
}

func (l *Lookup) stackUpdated(evt event.Event) {
	data := evt.Data().(StackUpdatedData)
	l.setName(evt.AggregateID(), data.Stack.ID, data.Stack.Original().Name)
}

func (l *Lookup) setName(galleryID, stackID uuid.UUID, name string) {
	s := l.gallery(galleryID)
	s.setName(stackID, name)
}

func (l *Lookup) removeName(galleryID, stackID uuid.UUID, name string) {
	s := l.gallery(galleryID)
	s.removeName(stackID, name)
}

func (l *Lookup) gallery(id uuid.UUID) *galleryLookup {
	l.mux.RLock()
	s, ok := l.galleries[id]
	l.mux.RUnlock()
	if ok {
		return s
	}

	s = newGalleryLookup()
	l.mux.Lock()
	defer l.mux.Unlock()
	l.galleries[id] = s

	return s
}

type galleryLookup struct {
	mux      sync.RWMutex
	nameToID map[string]uuid.UUID
}

func newGalleryLookup() *galleryLookup {
	return &galleryLookup{nameToID: make(map[string]uuid.UUID)}
}

func (l *galleryLookup) setName(id uuid.UUID, name string) {
	l.mux.Lock()
	defer l.mux.Unlock()
	l.nameToID[name] = id
}

func (l *galleryLookup) removeName(id uuid.UUID, name string) {
	l.mux.Lock()
	defer l.mux.Unlock()
	if l.nameToID[name] == id {
		delete(l.nameToID, name)
	}
}

func (l *galleryLookup) name(name string) (uuid.UUID, bool) {
	l.mux.RLock()
	defer l.mux.RUnlock()
	id, ok := l.nameToID[name]
	return id, ok
}
