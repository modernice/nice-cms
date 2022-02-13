package commands

import (
	"github.com/modernice/goes/codec"
	"github.com/modernice/nice-cms/media/document"
	"github.com/modernice/nice-cms/media/image/gallery"
	"github.com/modernice/nice-cms/static/nav"
)

// NewRegistry returns a new command registry with all commands registered.
func NewRegistry() *codec.GobRegistry {
	r := codec.Gob(codec.New())
	Register(r)
	return r
}

// Register registers all commands into the registry.
func Register(r *codec.GobRegistry) {
	nav.RegisterCommands(r)
	document.RegisterCommands(r)
	gallery.RegisterCommands(r)
}
