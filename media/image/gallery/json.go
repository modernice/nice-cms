package gallery

import "github.com/google/uuid"

// JSONGallery is the JSON representation of a Gallery.
type JSONGallery struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Stacks Stacks    `json:"stacks"`
}

// JSON returns the JSONGallery for g.
func (g *Implementation) JSON() JSONGallery {
	id, _, _ := g.gallery.Aggregate()
	return JSONGallery{
		ID:     id,
		Name:   g.Name,
		Stacks: g.Stacks,
	}
}

// Stack returns the Stack with the given UUID or ErrStackNotFound.
func (g JSONGallery) Stack(id uuid.UUID) (Stack, error) {
	for _, stack := range g.Stacks {
		if stack.ID == id {
			return stack.copy(), nil
		}
	}
	return Stack{}, ErrStackNotFound
}
