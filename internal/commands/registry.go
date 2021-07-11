package commands

import (
	"github.com/modernice/cms/static/nav"
	"github.com/modernice/goes/command"
)

func NewRegistry() command.Registry {
	r := command.NewRegistry()
	Register(r)
	return r
}

func Register(r command.Registry) {
	nav.RegisterCommands(r)
}
