package commands

import (
	"github.com/modernice/goes/command"
	"github.com/modernice/nice-cms/media/document"
	"github.com/modernice/nice-cms/static/nav"
	"github.com/modernice/nice-cms/static/page"
)

// NewRegistry returns a new command registry with all commands registered.
func NewRegistry() command.Registry {
	r := command.NewRegistry()
	Register(r)
	return r
}

// Register registers all commands into the registry.
func Register(r command.Registry) {
	page.RegisterCommands(r)
	nav.RegisterCommands(r)
	document.RegisterCommands(r)
}
