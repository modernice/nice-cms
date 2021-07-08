package gallery

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/modernice/goes/event"
	"github.com/modernice/goes/projection"
	"github.com/modernice/goes/projection/schedule"
)

// Lookup provides lookup of Gallery UUIDs. It is thread-safe.
type Lookup struct {
	galleriesMux sync.RWMutex
	galleries    map[uuid.UUID]*galleryLookup

	galleryNamesMux sync.RWMutex
	galleryNameToID map[string]uuid.UUID
}

// NewLookup returns a new Lookup.
func NewLookup() *Lookup {
	return &Lookup{
		galleries:       make(map[uuid.UUID]*galleryLookup),
		galleryNameToID: make(map[string]uuid.UUID),
	}
}

// GalleryName returns the UUID of the Gallery with the give name, or false if
// Lookup has no UUID for name.
func (l *Lookup) GalleryName(name string) (uuid.UUID, bool) {
	return l.galleryName(name)
}

// StackName returns the UUID of the Stack with the given StackName in the
// Gallery with the given UUID, or false if Lookup has no UUID for name.
func (l *Lookup) StackName(galleryID uuid.UUID, name string) (uuid.UUID, bool) {
	return l.gallery(galleryID).name(name)
}

// Project projects the Lookup in a new goroutine and returns a channel of
// asynchronous errors.
func (l *Lookup) Project(ctx context.Context, bus event.Bus, store event.Store, opts ...schedule.ContinuousOption) (<-chan error, error) {
	schedule := schedule.Continuously(bus, store, []string{
		Created,
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

func (l *Lookup) applyJob(job projection.Job) error {
	return job.Apply(job, l)
}

// ApplyEvent applies aggregate events.
func (l *Lookup) ApplyEvent(evt event.Event) {
	switch evt.Name() {
	case Created:
		l.galleryCreated(evt)
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

func (l *Lookup) galleryCreated(evt event.Event) {
	data := evt.Data().(CreatedData)
	l.setGalleryName(evt.AggregateID(), data.Name)
}

func (l *Lookup) imageUploaded(evt event.Event) {
	data := evt.Data().(ImageUploadedData)
	l.setStackName(evt.AggregateID(), data.Stack.ID, data.Stack.Original().Name)
}

func (l *Lookup) stackDeleted(evt event.Event) {
	data := evt.Data().(StackDeletedData)
	l.removeStackName(evt.AggregateID(), data.Stack.ID, data.Stack.Original().Name)
}

func (l *Lookup) stackRenamed(evt event.Event) {
	data := evt.Data().(StackRenamedData)
	l.removeStackName(evt.AggregateID(), data.StackID, data.OldName)
	l.setStackName(evt.AggregateID(), data.StackID, data.Name)
}

func (l *Lookup) stackUpdated(evt event.Event) {
	data := evt.Data().(StackUpdatedData)
	l.setStackName(evt.AggregateID(), data.Stack.ID, data.Stack.Original().Name)
}

func (l *Lookup) setGalleryName(galleryID uuid.UUID, name string) {
	l.galleryNamesMux.Lock()
	defer l.galleryNamesMux.Unlock()
	l.galleryNameToID[name] = galleryID
}

func (l *Lookup) galleryName(name string) (uuid.UUID, bool) {
	l.galleryNamesMux.RLock()
	defer l.galleryNamesMux.RUnlock()
	id, ok := l.galleryNameToID[name]
	return id, ok
}

func (l *Lookup) setStackName(galleryID, stackID uuid.UUID, name string) {
	s := l.gallery(galleryID)
	s.setStackName(stackID, name)
}

func (l *Lookup) removeStackName(galleryID, stackID uuid.UUID, name string) {
	s := l.gallery(galleryID)
	s.removeStackName(stackID, name)
}

func (l *Lookup) gallery(id uuid.UUID) *galleryLookup {
	l.galleriesMux.RLock()
	s, ok := l.galleries[id]
	l.galleriesMux.RUnlock()
	if ok {
		return s
	}

	s = newGalleryLookup()
	l.galleriesMux.Lock()
	defer l.galleriesMux.Unlock()
	l.galleries[id] = s

	return s
}

type galleryLookup struct {
	mux           sync.RWMutex
	stackNameToID map[string]uuid.UUID
}

func newGalleryLookup() *galleryLookup {
	return &galleryLookup{stackNameToID: make(map[string]uuid.UUID)}
}

func (l *galleryLookup) setStackName(id uuid.UUID, name string) {
	l.mux.Lock()
	defer l.mux.Unlock()
	l.stackNameToID[name] = id
}

func (l *galleryLookup) removeStackName(id uuid.UUID, name string) {
	l.mux.Lock()
	defer l.mux.Unlock()
	if l.stackNameToID[name] == id {
		delete(l.stackNameToID, name)
	}
}

func (l *galleryLookup) name(name string) (uuid.UUID, bool) {
	l.mux.RLock()
	defer l.mux.RUnlock()
	id, ok := l.stackNameToID[name]
	return id, ok
}
