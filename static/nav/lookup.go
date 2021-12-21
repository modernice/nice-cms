package nav

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/modernice/goes/event"
	"github.com/modernice/goes/projection"
	"github.com/modernice/goes/projection/schedule"
)

// Lookup provides UUID lookup for Navs.
//
// Use NewLookup to create a Lookup.
type Lookup struct {
	nameToIDMux sync.RWMutex
	nameToID    map[string]uuid.UUID
}

// NewLookup returns a new Lookup.
func NewLookup() *Lookup {
	return &Lookup{
		nameToID: make(map[string]uuid.UUID),
	}
}

// Name returns the UUID of the Nav with the given name, or false.
func (l *Lookup) Name(name string) (uuid.UUID, bool) {
	l.nameToIDMux.RLock()
	defer l.nameToIDMux.RUnlock()
	id, ok := l.nameToID[name]
	return id, ok
}

// Project projects the Lookup in a new goroutine and returns a channel of
// asynchronous errors.
func (l *Lookup) Project(ctx context.Context, bus event.Bus, store event.Store, opts ...schedule.ContinuousOption) (<-chan error, error) {
	schedule := schedule.Continuously(bus, store, Events[:], opts...)

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

// ApplyEvent applies events.
func (l *Lookup) ApplyEvent(evt event.Event) {
	switch evt.Name() {
	case Created:
		l.created(evt)
	}
}

func (l *Lookup) created(evt event.Event) {
	data := evt.Data().(CreatedData)
	id, _, _ := evt.Aggregate()
	l.setID(data.Name, id)
}

func (l *Lookup) setID(name string, id uuid.UUID) {
	l.nameToIDMux.Lock()
	defer l.nameToIDMux.Unlock()
	l.nameToID[name] = id
}
