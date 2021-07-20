package commands

import (
	"github.com/modernice/goes/command"
	"github.com/modernice/nice-cms/static/nav"
)

func NewRegistry() command.Registry {
	r := command.NewRegistry()
	Register(r)
	return r
}

func Register(r command.Registry) {
	nav.RegisterCommands(r)
}
