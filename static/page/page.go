package page

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/modernice/goes/aggregate"
	"github.com/modernice/goes/event"
)

// Aggregate is the name of the Page aggregate.
const Aggregate = "cms.static.page"

var (
	// ErrEmptyName is returned when trying to create a Page with an empty name.
	ErrEmptyName = errors.New("empty name")
)

// Page is a web page.
type Page struct {
	*aggregate.Base

	Name string
}

// Create creates the Page with the given name. Create(name) is a shortcut for
//	p := New(uuid.New())
//	err := p.Create(name)
func Create(name string) (*Page, error) {
	p := New(uuid.New())
	if err := p.Create(name); err != nil {
		return nil, err
	}
	return p, nil
}

// New returns a new Page. You probably want to use Create instead.
func New(id uuid.UUID) *Page {
	return &Page{
		Base: aggregate.New(Aggregate, id),
	}
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

// ApplyEvent applies aggregate events.
func (p *Page) ApplyEvent(evt event.Event) {
	switch evt.Name() {
	case Created:
		p.create(evt)
	}
}
