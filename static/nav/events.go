package nav

import "github.com/modernice/goes/event"

const (
	// Created means a Nav was created.
	Created = "cms.static.nav.created"

	// ItemsAdded means Items were added to a Nav.
	ItemsAdded = "cms.static.nav.items_added"

	// ItemsRemoved means Items were removed from a Nav.
	ItemsRemoved = "cms.static.nav.items_removed"

	// Sorted means a Nav was sorted.
	Sorted = "cms.static.nav.sorted"
)

// Events are all navigation events.
var Events = [...]string{
	Created,
	ItemsAdded,
	ItemsRemoved,
	Sorted,
}

// CreatedData is the event data for Created.
type CreatedData struct {
	Name string
}

// ItemsAddedData is the event data for ItemsAdded.
type ItemsAddedData struct {
	Items []Item
	Index int
	Path  string
}

// ItemsRemovedData is the event data for ItemsRemoved.
type ItemsRemovedData struct {
	Items []string
}

// SortedData is the event data for Sorted.
type SortedData struct {
	Sorting []string
	Path    string
}

// RegisterEvents registers events into an event registry.
func RegisterEvents(r event.Registry) {
	r.Register(Created, func() event.Data { return CreatedData{} })
	r.Register(ItemsAdded, func() event.Data { return ItemsAddedData{} })
	r.Register(ItemsRemoved, func() event.Data { return ItemsRemovedData{} })
	r.Register(Sorted, func() event.Data { return SortedData{} })
}
