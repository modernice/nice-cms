package nav

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/modernice/goes/event"
	"github.com/modernice/goes/helper/streams"
)

// ReadCache is a read-cache for Navs.
type ReadCache struct {
	repo Repository

	mux   sync.RWMutex
	cache map[uuid.UUID][]byte
}

// NewReadCache returns a new ReadCache.
func NewReadCache(repo Repository) *ReadCache {
	return &ReadCache{
		repo:  repo,
		cache: make(map[uuid.UUID][]byte),
	}
}

// Fetch fetches the Nav with the given UUID. Fetch first tries to find the Nav
// in the cache. If it's not cached, Fetch actually fetches from the underlying
// Repository.
func (c *ReadCache) Fetch(ctx context.Context, id uuid.UUID) (*Nav, error) {
	if nav, ok := c.cached(id); ok {
		return nav, nil
	}

	nav, err := c.repo.Fetch(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetch Nav from Repository: %w", err)
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(nav); err != nil {
		return nil, fmt.Errorf("gob encode: %w", err)
	}

	c.mux.Lock()
	defer c.mux.Unlock()
	c.cache[nav.ID] = buf.Bytes()

	return nav, nil
}

func (c *ReadCache) cached(id uuid.UUID) (*Nav, bool) {
	c.mux.RLock()
	b, ok := c.cache[id]
	c.mux.RUnlock()
	if !ok {
		return nil, false
	}

	var nav Nav
	if err := gob.NewDecoder(bytes.NewReader(b)).Decode(&nav); err != nil {
		c.mux.Lock()
		defer c.mux.Unlock()
		delete(c.cache, id)
		return nil, false
	}

	return &nav, true
}

// Run starts the ReadCache in a new goroutine and returns a channel of
// asynchronous errors that is closed when ctx is canceled.
//
// Whenever a Nav event is published, ReadCache flushes the Cache for that Nav.
func (c *ReadCache) Run(ctx context.Context, bus event.Bus, debounce time.Duration) (<-chan error, error) {
	events, errs, err := bus.Subscribe(ctx, Events[:]...)
	if err != nil {
		return nil, fmt.Errorf("subscribe to %v events: %w", Events, err)
	}

	out := make(chan error)
	go c.handle(ctx, debounce, events, errs, out)

	return out, nil
}

func (c *ReadCache) handle(
	ctx context.Context,
	debounce time.Duration,
	events <-chan event.Event,
	errs <-chan error,
	out chan<- error,
) {
	defer close(out)

	fail := func(err error) {
		select {
		case <-ctx.Done():
		case out <- err:
		}
	}

	var mux sync.Mutex
	debouncers := make(map[uuid.UUID]*time.Timer)

	defer func() {
		mux.Lock()
		defer mux.Unlock()
		for _, timer := range debouncers {
			timer.Stop()
		}
		debouncers = nil
	}()

	streams.ForEach(ctx, func(evt event.Event) {
		mux.Lock()
		defer mux.Unlock()

		id, _, _ := evt.Aggregate()

		if debouncer, ok := debouncers[id]; ok {
			debouncer.Stop()
			delete(debouncers, id)
			debouncer = nil
		}

		debouncers[id] = time.AfterFunc(debounce, func() {
			c.mux.Lock()
			defer c.mux.Unlock()
			delete(c.cache, id)

			mux.Lock()
			defer mux.Unlock()
			delete(debouncers, id)
		})
	}, fail, events, errs)
}
