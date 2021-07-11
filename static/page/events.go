package page

import (
	"github.com/modernice/cms/static/page/field"
	"github.com/modernice/goes/event"
)

const (
	// Created means a Page was created.
	Created = "cms.static.page.created"

	// FieldsAdded means Fields were added to a Page.
	FieldsAdded = "cms.static.page.fields_added"

	// FieldsRemoved means Fields were removed from a Page.
	FieldsRemoved = "cms.static.page.fields_removed"
)

// CreatedData is the event data for Created.
type CreatedData struct {
	Name string
}

// FieldsAddedData is the event data for FieldsAdded.
type FieldsAddedData struct {
	Fields []field.Field
}

// FieldsRemovedData is the event data for FieldsRemoved.
type FieldsRemovedData struct {
	Fields []string
}

// RegisterEvents register events into a registry.
func RegisterEvents(r event.Registry) {
	r.Register(Created, func() event.Data { return CreatedData{} })
	r.Register(FieldsAdded, func() event.Data { return FieldsAddedData{} })
	r.Register(FieldsRemoved, func() event.Data { return FieldsRemovedData{} })
}