package document

import "github.com/google/uuid"

type JSONShelf struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Documents []Document `json:"documents"`
}

func (s *Shelf) JSON() JSONShelf {
	return JSONShelf{
		ID:        s.ID,
		Name:      s.Name,
		Documents: s.Documents,
	}
}

// Document returns the Document with the given UUID or ErrDocumentNotFound.
func (s JSONShelf) Document(id uuid.UUID) (Document, error) {
	for _, doc := range s.Documents {
		if doc.ID == id {
			return doc, nil
		}
	}
	return Document{}, ErrNotFound
}
