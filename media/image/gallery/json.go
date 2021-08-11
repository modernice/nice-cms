package gallery

import "github.com/google/uuid"

// JSONGallery is the JSON representation of a Gallery.
type JSONGallery struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Stacks []Stack   `json:"stacks"`
}

// JSON returns the JSONGallery for g.
func (g *Gallery) JSON() JSONGallery {
	return JSONGallery{
		ID:     g.ID,
		Name:   g.Name,
		Stacks: g.Stacks,
	}
}
