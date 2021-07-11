package events

import (
	"github.com/modernice/cms/static/nav"
	"github.com/modernice/goes/event"
)

func NewRegistry() event.Registry {
	r := event.NewRegistry()
	Register(r)
	return r
}

func Register(r event.Registry) {
	nav.RegisterEvents(r)
}
