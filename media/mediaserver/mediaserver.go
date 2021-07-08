package mediaserver

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/modernice/cms/internal/api"
	"github.com/modernice/cms/media"
	"github.com/modernice/cms/media/document"
)

type server struct {
	chi.Router
}

type Option func(*server)

func WithDocuments(
	shelfs document.Repository,
	lookup *document.Lookup,
	storage media.Storage,
) Option {
	return func(s *server) {
		s.Mount("/shelfs", newDocumentServer(shelfs, lookup, storage))
	}
}

func New(opts ...Option) http.Handler {
	srv := server{Router: chi.NewRouter()}
	for _, opt := range opts {
		opt(&srv)
	}
	return &srv
}

type documentServer struct {
	chi.Router

	shelfs  document.Repository
	lookup  *document.Lookup
	storage media.Storage
}

func newDocumentServer(
	shelfs document.Repository,
	lookup *document.Lookup,
	storage media.Storage,
) *documentServer {
	srv := documentServer{
		Router:  chi.NewRouter(),
		shelfs:  shelfs,
		lookup:  lookup,
		storage: storage,
	}
	srv.init()
	return &srv
}

func (s *documentServer) init() {
	s.Get("/lookup/name/{Name}", s.lookupName)
	s.Post("/{ShelfID}/documents", s.uploadDocument)
	s.Put("/{ShelfID}/documents/{DocumentID}", s.updateDocument)
	s.Delete("/{ShelfID}/documents/{DocumentID}", s.deleteDocument)
	s.Post("/{ShelfID}/documents/{DocumentID}/tags", s.addTags)
	s.Delete("/{ShelfID}/documents/{DocumentID}/tags/{Tags}", s.removeTags)
}

func (s *documentServer) lookupName(w http.ResponseWriter, r *http.Request) {
	var resp struct {
		ShelfID uuid.UUID `json:"shelfId"`
	}

	name := chi.URLParam(r, "Name")
	id, ok := s.lookup.ShelfName(name)
	if !ok {
		api.Error(w, r, http.StatusNotFound, api.Friendly(nil, "Could not find Shelf named %q", name))
		return
	}
	resp.ShelfID = id

	api.JSON(w, r, http.StatusOK, resp)
}

func (s *documentServer) uploadDocument(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	uniqueName := r.FormValue("uniqueName")
	disk := r.FormValue("disk")
	path := r.FormValue("path")
	file, _, err := r.FormFile("document")
	if err != nil {
		api.Error(w, r, http.StatusUnprocessableEntity, api.Friendly(err, "Failed to parse file: %v", err))
		return
	}
	defer file.Close()

	shelfID, err := api.ExtractUUID(r, "ShelfID")
	if err != nil {
		api.Error(w, r, http.StatusBadRequest, err)
		return
	}

	shelf, err := s.fetchShelf(r.Context(), shelfID)
	if err != nil {
		api.Error(w, r, http.StatusNotFound, err)
		return
	}

	doc, err := shelf.Add(r.Context(), s.storage, file, uniqueName, name, disk, path)
	if err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to add document to shelf: %v", err))
		return
	}

	if err := s.shelfs.Save(r.Context(), shelf); err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to save shelf: %v", err))
		return
	}

	api.JSON(w, r, http.StatusCreated, doc)
}

func (s *documentServer) updateDocument(w http.ResponseWriter, r *http.Request) {
	shelfID, err := api.ExtractUUID(r, "ShelfID")
	if err != nil {
		api.Error(w, r, http.StatusBadRequest, err)
		return
	}

	documentID, err := api.ExtractUUID(r, "DocumentID")
	if err != nil {
		api.Error(w, r, http.StatusBadRequest, err)
		return
	}

	var req struct {
		Name       string `json:"name"`
		UniqueName string `json:"uniqueName"`
	}

	if err := api.Decode(r.Body, &req); err != nil {
		api.Error(w, r, http.StatusBadRequest, err)
		return
	}

	shelf, err := s.fetchShelf(r.Context(), shelfID)
	if err != nil {
		api.Error(w, r, http.StatusNotFound, err)
		return
	}

	if req.Name != "" {
		if _, err := shelf.RenameDocument(documentID, req.Name); err != nil {
			api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to rename document: %v", err))
			return
		}
	}

	doc, err := shelf.Document(documentID)
	if err != nil {
		api.Error(w, r, http.StatusNotFound, api.Friendly(err, "Document with UUID %q not found.", documentID))
		return
	}

	if req.UniqueName != "" {
		if _, err := shelf.MakeUnique(documentID, req.UniqueName); err != nil {
			api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to give document a unique name: %v", err))
			return
		}
	} else if doc.UniqueName != "" {
		if _, err := shelf.MakeNonUnique(documentID); err != nil {
			api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to remove unique name: %v", err))
			return
		}
	}

	if err := s.shelfs.Save(r.Context(), shelf); err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to save shelf: %v", err))
		return
	}

	doc, err = shelf.Document(documentID)
	if err != nil {
		api.Error(w, r, http.StatusNotFound, api.Friendly(err, "Document with UUID %q not found.", documentID))
		return
	}

	api.JSON(w, r, http.StatusOK, doc)
}

func (s *documentServer) deleteDocument(w http.ResponseWriter, r *http.Request) {
	shelfID, err := api.ExtractUUID(r, "ShelfID")
	if err != nil {
		api.Error(w, r, http.StatusBadRequest, err)
		return
	}

	documentID, err := api.ExtractUUID(r, "DocumentID")
	if err != nil {
		api.Error(w, r, http.StatusBadRequest, err)
		return
	}

	shelf, err := s.fetchShelf(r.Context(), shelfID)
	if err != nil {
		api.Error(w, r, http.StatusNotFound, err)
		return
	}

	if err := shelf.Remove(r.Context(), s.storage, documentID); err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to delete document: %v", err))
		return
	}

	if err := s.shelfs.Save(r.Context(), shelf); err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to save shelf: %v", err))
		return
	}

	api.NoContent(w, r)
}

func (s *documentServer) fetchShelf(ctx context.Context, id uuid.UUID) (*document.Shelf, error) {
	shelf, err := s.shelfs.Fetch(ctx, id)
	if err != nil {
		return nil, api.Friendly(err, "Shelf with UUID %q not found.", id)
	}
	return shelf, nil
}

func (s *documentServer) addTags(w http.ResponseWriter, r *http.Request) {
	shelfID, err := api.ExtractUUID(r, "ShelfID")
	if err != nil {
		api.Error(w, r, http.StatusBadRequest, err)
		return
	}

	documentID, err := api.ExtractUUID(r, "DocumentID")
	if err != nil {
		api.Error(w, r, http.StatusBadRequest, err)
		return
	}

	var req struct {
		Tags []string `json:"tags"`
	}

	if err := api.Decode(r.Body, &req); err != nil {
		api.Error(w, r, http.StatusBadGateway, err)
		return
	}

	shelf, err := s.fetchShelf(r.Context(), shelfID)
	if err != nil {
		api.Error(w, r, http.StatusNotFound, err)
		return
	}

	doc, err := shelf.Tag(documentID, req.Tags...)
	if err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to tag document: %v", err))
		return
	}

	if err := s.shelfs.Save(r.Context(), shelf); err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to save shelf: %v", err))
		return
	}

	api.JSON(w, r, http.StatusOK, doc)
}

func (s *documentServer) removeTags(w http.ResponseWriter, r *http.Request) {
	shelfID, err := api.ExtractUUID(r, "ShelfID")
	if err != nil {
		api.Error(w, r, http.StatusBadRequest, err)
		return
	}

	documentID, err := api.ExtractUUID(r, "DocumentID")
	if err != nil {
		api.Error(w, r, http.StatusBadRequest, err)
		return
	}

	taglist := chi.URLParam(r, "Tags")
	tags := strings.Split(taglist, ",")

	shelf, err := s.fetchShelf(r.Context(), shelfID)
	if err != nil {
		api.Error(w, r, http.StatusNotFound, err)
		return
	}

	doc, err := shelf.Untag(documentID, tags...)
	if err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to untag document: %v", err))
		return
	}

	if err := s.shelfs.Save(r.Context(), shelf); err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to save shelf: %v", err))
		return
	}

	api.JSON(w, r, http.StatusOK, doc)
}
