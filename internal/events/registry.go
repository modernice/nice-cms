package events

import (
	"github.com/modernice/goes/event"
	"github.com/modernice/nice-cms/media/document"
	"github.com/modernice/nice-cms/media/image/gallery"
	"github.com/modernice/nice-cms/static/nav"
)

// NewRegistry returns a new event registry with all events registered.
func NewRegistry() event.Registry {
	r := event.NewRegistry()
	Register(r)
	return r
}

// Register registers all events into the event registry.
func Register(r event.Registry) {
	nav.RegisterEvents(r)
	document.RegisterEvents(r)
	gallery.RegisterEvents(r)
}
