package events

import (
	"github.com/modernice/goes/event"
	"github.com/modernice/nice-cms/static/nav"
)

func NewRegistry() event.Registry {
	r := event.NewRegistry()
	Register(r)
	return r
}

func Register(r event.Registry) {
	nav.RegisterEvents(r)
}
