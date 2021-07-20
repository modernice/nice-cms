package mediaserver

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/modernice/nice-cms/internal/api"
	"github.com/modernice/nice-cms/media"
	"github.com/modernice/nice-cms/media/document"
	"github.com/modernice/nice-cms/media/image/gallery"
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

func WithGalleries(
	galleries gallery.Repository,
	lookup *gallery.Lookup,
	storage media.Storage,
) Option {
	return func(s *server) {
		s.Mount("/galleries", newGalleryServer(galleries, lookup, storage))
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
	s.Put("/{ShelfID}/documents/{DocumentID}", s.replaceDocument)
	s.Patch("/{ShelfID}/documents/{DocumentID}", s.updateDocument)
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

func (s *documentServer) replaceDocument(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("document")
	if err != nil {
		api.Error(w, r, http.StatusBadRequest, api.Friendly(err, "Invalid file: %v", err))
		return
	}
	defer file.Close()

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

	replaced, err := shelf.Replace(r.Context(), s.storage, file, documentID)
	if err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to replace document: %v", err))
		return
	}

	if err := s.shelfs.Save(r.Context(), shelf); err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to save shelf: %v", err))
		return
	}

	api.JSON(w, r, http.StatusOK, replaced)
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
		Name       string  `json:"name"`
		UniqueName *string `json:"uniqueName"`
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

	if req.UniqueName != nil {
		if *req.UniqueName != "" {
			if _, err := shelf.MakeUnique(documentID, *req.UniqueName); err != nil {
				api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to give document a unique name: %v", err))
				return
			}
		} else {
			if _, err := shelf.MakeNonUnique(documentID); err != nil {
				api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to remove unique name: %v", err))
				return
			}
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

	tags := strings.Split(chi.URLParam(r, "Tags"), ",")

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

type galleryServer struct {
	chi.Router

	galleries gallery.Repository
	lookup    *gallery.Lookup
	storage   media.Storage
}

func newGalleryServer(
	galleries gallery.Repository,
	lookup *gallery.Lookup,
	storage media.Storage,
) *galleryServer {
	srv := galleryServer{
		Router:    chi.NewRouter(),
		galleries: galleries,
		lookup:    lookup,
		storage:   storage,
	}
	srv.init()
	return &srv
}

func (s *galleryServer) init() {
	s.Get("/lookup/name/{Name}", s.lookupName)
	s.Post("/{GalleryID}/stacks", s.uploadImage)
	s.Put("/{GalleryID}/stacks/{StackID}", s.replaceImage)
	s.Patch("/{GalleryID}/stacks/{StackID}", s.updateStack)
	s.Delete("/{GalleryID}/stacks/{StackID}", s.deleteStack)
	s.Post("/{GalleryID}/stacks/{StackID}/tags", s.tagStack)
	s.Delete("/{GalleryID}/stacks/{StackID}/tags/{Tags}", s.untagStack)
}

func (s *galleryServer) lookupName(w http.ResponseWriter, r *http.Request) {
	var resp struct {
		GalleryID uuid.UUID `json:"galleryId"`
	}

	name := chi.URLParam(r, "Name")
	id, ok := s.lookup.GalleryName(name)
	if !ok {
		api.Error(w, r, http.StatusNotFound, api.Friendly(nil, "Could not find Gallery named %q", name))
		return
	}
	resp.GalleryID = id

	api.JSON(w, r, http.StatusOK, resp)
}

func (s *galleryServer) uploadImage(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	disk := r.FormValue("disk")
	path := r.FormValue("path")
	file, _, err := r.FormFile("image")
	if err != nil {
		api.Error(w, r, http.StatusBadRequest, api.Friendly(err, "Invalid file: %v", err))
		return
	}
	defer file.Close()

	galleryID, err := api.ExtractUUID(r, "GalleryID")
	if err != nil {
		api.Error(w, r, http.StatusBadRequest, err)
		return
	}

	g, err := s.galleries.Fetch(r.Context(), galleryID)
	if err != nil {
		api.Error(w, r, http.StatusNotFound, api.Friendly(err, "Gallery with UUID %q not found.", galleryID))
		return
	}

	stack, err := g.Upload(r.Context(), s.storage, file, name, disk, path)
	if err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to upload image: %v", err))
		return
	}

	if err := s.galleries.Save(r.Context(), g); err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to save gallery: %v", err))
		return
	}

	api.JSON(w, r, http.StatusCreated, stack)
}

func (s *galleryServer) deleteStack(w http.ResponseWriter, r *http.Request) {
	galleryID, err := api.ExtractUUID(r, "GalleryID")
	if err != nil {
		api.Error(w, r, http.StatusBadRequest, err)
		return
	}

	stackID, err := api.ExtractUUID(r, "StackID")
	if err != nil {
		api.Error(w, r, http.StatusBadRequest, err)
		return
	}

	g, err := s.galleries.Fetch(r.Context(), galleryID)
	if err != nil {
		api.Error(w, r, http.StatusNotFound, api.Friendly(err, "Gallery with UUID %q not found.", galleryID))
		return
	}

	stack, err := g.Stack(stackID)
	if err != nil {
		api.Error(w, r, http.StatusNotFound, api.Friendly(err, "Stack with UUID %q not found.", stackID))
		return
	}

	if err := g.Delete(r.Context(), s.storage, stack); err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to delete stack: %v", err))
		return
	}

	if err := s.galleries.Save(r.Context(), g); err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to save gallery: %v", err))
		return
	}

	api.NoContent(w, r)
}

func (s *galleryServer) tagStack(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Tags []string `json:"tags"`
	}

	if err := api.Decode(r.Body, &req); err != nil {
		api.Error(w, r, http.StatusBadRequest, err)
		return
	}

	galleryID, err := api.ExtractUUID(r, "GalleryID")
	if err != nil {
		api.Error(w, r, http.StatusBadRequest, err)
		return
	}

	stackID, err := api.ExtractUUID(r, "StackID")
	if err != nil {
		api.Error(w, r, http.StatusBadRequest, err)
		return
	}

	g, err := s.galleries.Fetch(r.Context(), galleryID)
	if err != nil {
		api.Error(w, r, http.StatusNotFound, api.Friendly(err, "Gallery with UUID %q not found.", galleryID))
		return
	}

	stack, err := g.Stack(stackID)
	if err != nil {
		api.Error(w, r, http.StatusNotFound, api.Friendly(err, "Stack with UUID %q not found.", stackID))
		return
	}

	if stack, err = g.Tag(r.Context(), stack, req.Tags...); err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to tag image: %v", err))
		return
	}

	if err := s.galleries.Save(r.Context(), g); err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to save gallery: %v", err))
		return
	}

	api.JSON(w, r, http.StatusOK, stack)
}

func (s *galleryServer) untagStack(w http.ResponseWriter, r *http.Request) {
	galleryID, err := api.ExtractUUID(r, "GalleryID")
	if err != nil {
		api.Error(w, r, http.StatusBadRequest, err)
		return
	}

	stackID, err := api.ExtractUUID(r, "StackID")
	if err != nil {
		api.Error(w, r, http.StatusBadRequest, err)
		return
	}

	tags := strings.Split(chi.URLParam(r, "Tags"), ",")

	g, err := s.galleries.Fetch(r.Context(), galleryID)
	if err != nil {
		api.Error(w, r, http.StatusNotFound, api.Friendly(err, "Gallery with UUID %q not found.", galleryID))
		return
	}

	stack, err := g.Stack(stackID)
	if err != nil {
		api.Error(w, r, http.StatusNotFound, api.Friendly(err, "Stack with UUID %q not found.", stackID))
		return
	}

	if stack, err = g.Untag(r.Context(), stack, tags...); err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to untag image: %v", err))
		return
	}

	if err := s.galleries.Save(r.Context(), g); err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to save gallery: %v", err))
		return
	}

	api.JSON(w, r, http.StatusOK, stack)
}

func (s *galleryServer) replaceImage(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("image")
	if err != nil {
		api.Error(w, r, http.StatusBadRequest, api.Friendly(err, "Invalid file: %v", err))
		return
	}
	defer file.Close()

	galleryID, err := api.ExtractUUID(r, "GalleryID")
	if err != nil {
		api.Error(w, r, http.StatusBadRequest, err)
		return
	}

	stackID, err := api.ExtractUUID(r, "StackID")
	if err != nil {
		api.Error(w, r, http.StatusBadRequest, err)
		return
	}

	g, err := s.galleries.Fetch(r.Context(), galleryID)
	if err != nil {
		api.Error(w, r, http.StatusNotFound, api.Friendly(err, "Gallery with UUID %q not found.", galleryID))
		return
	}

	replaced, err := g.Replace(r.Context(), s.storage, file, stackID)
	if err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to replace image: %v", err))
		return
	}

	if err := s.galleries.Save(r.Context(), g); err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to save gallery: %v", err))
		return
	}

	api.JSON(w, r, http.StatusOK, replaced)
}

func (s *galleryServer) updateStack(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := api.Decode(r.Body, &req); err != nil {
		api.Error(w, r, http.StatusBadGateway, err)
		return
	}

	galleryID, err := api.ExtractUUID(r, "GalleryID")
	if err != nil {
		api.Error(w, r, http.StatusBadRequest, err)
		return
	}

	stackID, err := api.ExtractUUID(r, "StackID")
	if err != nil {
		api.Error(w, r, http.StatusBadRequest, err)
		return
	}

	g, err := s.galleries.Fetch(r.Context(), galleryID)
	if err != nil {
		api.Error(w, r, http.StatusNotFound, api.Friendly(err, "Gallery with UUID %q not found.", galleryID))
		return
	}

	if req.Name != "" {
		if _, err := g.RenameStack(r.Context(), stackID, req.Name); err != nil {
			api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to rename stack: %v", err))
			return
		}
	}

	if err := s.galleries.Save(r.Context(), g); err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to save gallery: %v", err))
		return
	}

	stack, err := g.Stack(stackID)
	if err != nil {
		api.Error(w, r, http.StatusNotFound, api.Friendly(err, "Stack with UUID %q not found.", stackID))
		return
	}

	api.JSON(w, r, http.StatusOK, stack)
}
