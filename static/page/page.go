package page

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/modernice/cms/internal/unique"
	"github.com/modernice/cms/static/page/field"
	"github.com/modernice/goes/aggregate"
	"github.com/modernice/goes/event"
)

// Aggregate is the name of the Page aggregate.
const Aggregate = "cms.static.page"

var (
	// ErrEmptyName is returned when trying to create a Page with an empty name.
	ErrEmptyName = errors.New("empty name")

	// ErrNotCreated is returned when trying to use a Page that wasn't created yet.
	ErrNotCreated = errors.New("page not created")

	// ErrDuplicateField is returned when adding a Field to a Page that already
	// has a Field with the same name.
	ErrDuplicateField = errors.New("duplicate field")

	// ErrFieldNotFound is returned when trying to get a Field of a Page that
	// doesn't exist in the Page.
	ErrFieldNotFound = errors.New("field not found")

	// ErrGuarded is retured when trying to remove a guarded Field from a Page.
	ErrGuarded = errors.New("guarded field")
)

// Page is a web page.
type Page struct {
	*aggregate.Base

	Name   string
	Fields []field.Field
}

// Create creates the Page with the given name. Create(name, fields...) is a shortcut for
//	p := New(uuid.New())
//	err := p.Create("foo", fields...)
//
// Fields passed to Create are added to the Page as guarded Fields that cannot
// be removed. To add removable Fields to a Page p, use p.Add instead:
//	p, _ := page.Create("foo")
//	p.Add(field.NewText(...), field.NewToggle(...))
func Create(name string, fields ...field.Field) (*Page, error) {
	p := New(uuid.New())
	if err := p.Create(name); err != nil {
		return nil, err
	}
	if len(fields) > 0 {
		if err := p.Add(guarded(fields...)...); err != nil {
			return p, err
		}
	}
	return p, nil
}

// New returns a new Page. You probably want to use Create instead.
func New(id uuid.UUID) *Page {
	return &Page{
		Base:   aggregate.New(Aggregate, id),
		Fields: make([]field.Field, 0),
	}
}

func (p *Page) Field(name string) (field.Field, error) {
	for _, f := range p.Fields {
		if f.Name == name {
			return f, nil
		}
	}
	return field.Field{}, ErrFieldNotFound
}

// Create creates the page by giving it a name.
func (p *Page) Create(name string) error {
	if name = strings.TrimSpace(name); name == "" {
		return ErrEmptyName
	}

	aggregate.NextEvent(p, Created, CreatedData{Name: name})

	return nil
}

func (p *Page) create(evt event.Event) {
	data := evt.Data().(CreatedData)
	p.Name = data.Name
}

func (p *Page) Add(fields ...field.Field) error {
	if err := p.checkCreated(); err != nil {
		return err
	}

	names := make(map[string]bool)
	for _, f := range fields {
		if names[f.Name] {
			return fmt.Errorf("%q: %w", f.Name, ErrDuplicateField)
		}
		names[f.Name] = true

		if _, err := p.Field(f.Name); err == nil {
			return fmt.Errorf("%q: %w", f.Name, ErrDuplicateField)
		}
	}

	aggregate.NextEvent(p, FieldsAdded, FieldsAddedData{Fields: fields})

	return nil
}

func (p *Page) addFields(evt event.Event) {
	data := evt.Data().(FieldsAddedData)
	p.Fields = append(p.Fields, data.Fields...)
}

func (p *Page) Remove(fields ...string) error {
	fields = unique.Strings(fields...)

	for _, name := range fields {
		f, err := p.Field(name)
		if err != nil {
			return fmt.Errorf("%q: %w", name, err)
		}
		if f.Guarded {
			return fmt.Errorf("%q: %w", name, ErrGuarded)
		}
	}

	aggregate.NextEvent(p, FieldsRemoved, FieldsRemovedData{Fields: fields})

	return nil
}

func (p *Page) removeFields(evt event.Event) {
	data := evt.Data().(FieldsRemovedData)
	for _, name := range data.Fields {
		p.removeField(name)
	}
}

func (p *Page) removeField(name string) {
	for i, f := range p.Fields {
		if f.Name == name {
			p.Fields = append(p.Fields[:i], p.Fields[i+1:]...)
			return
		}
	}
}

func (p *Page) checkCreated() error {
	if p.Name == "" {
		return ErrNotCreated
	}
	return nil
}

// ApplyEvent applies aggregate events.
func (p *Page) ApplyEvent(evt event.Event) {
	switch evt.Name() {
	case Created:
		p.create(evt)
	case FieldsAdded:
		p.addFields(evt)
	case FieldsRemoved:
		p.removeFields(evt)
	}
}

func guarded(fields ...field.Field) []field.Field {
	out := make([]field.Field, len(fields))
	for i, f := range fields {
		f.Guarded = true
		out[i] = f
	}
	return out
}
