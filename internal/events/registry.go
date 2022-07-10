package events

import (
	"github.com/modernice/goes/codec"
	"github.com/modernice/goes/command/cmdbus"
	"github.com/modernice/nice-cms/media/document"
	"github.com/modernice/nice-cms/media/image/gallery"
	"github.com/modernice/nice-cms/static/nav"
)

// NewRegistry returns a new event registry with all events registered.
func NewRegistry() codec.Registerer {
	r := codec.New()
	Register(r)
	return r
}

// Register registers all events into the event registry.
func Register(r codec.Registerer) {
	nav.RegisterEvents(r)
	document.RegisterEvents(r)
	gallery.RegisterEvents(r)
	cmdbus.RegisterEvents(r)
}
