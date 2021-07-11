package page

import "github.com/modernice/goes/event"

const (
	// Created means a Page was created.
	Created = "cms.static.page.created"
)

// CreatedData is the event data for Created.
type CreatedData struct {
	Name string
}

// RegisterEvents register events into a registry.
func RegisterEvents(r event.Registry) {
	r.Register(Created, func() event.Data { return CreatedData{} })
}
