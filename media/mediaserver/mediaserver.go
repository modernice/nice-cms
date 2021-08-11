package mediaserver

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/modernice/goes/command"
	"github.com/modernice/goes/command/cmdbus/dispatch"
	"github.com/modernice/nice-cms/internal/api"
	"github.com/modernice/nice-cms/media/document"
	"github.com/modernice/nice-cms/media/image/gallery"
)

type Client interface {
	LookupShelfByName(context.Context, string) (uuid.UUID, bool, error)
	UploadDocument(_ context.Context, shelfID uuid.UUID, _ io.Reader, uniqueName, name, disk, path string) (document.Document, error)
	ReplaceDocument(_ context.Context, shelfID, documentID uuid.UUID, _ io.Reader) (document.Document, error)
	FetchShelf(context.Context, uuid.UUID) (document.JSONShelf, error)

	LookupGalleryByName(context.Context, string) (uuid.UUID, bool, error)
	LookupGalleryStackByName(_ context.Context, galleryID uuid.UUID, name string) (uuid.UUID, bool, error)
	UploadImage(_ context.Context, galleryID uuid.UUID, _ io.Reader, name, disk, path string) (gallery.Stack, error)
	ReplaceImage(_ context.Context, galleryID, stackID uuid.UUID, _ io.Reader) (gallery.Stack, error)
	FetchGallery(context.Context, uuid.UUID) (gallery.JSONGallery, error)
}

func New(client Client, commands command.Bus) http.Handler {
	r := chi.NewRouter()
	r.Mount("/documents", newDocumentServer(client, commands))
	r.Mount("/galleries", newGalleryServer(client, commands))
	return r
}

type documentServer struct {
	chi.Router

	client   Client
	commands command.Bus
}

func newDocumentServer(client Client, commands command.Bus) *documentServer {
	s := documentServer{
		Router:   chi.NewRouter(),
		client:   client,
		commands: commands,
	}
	s.init()
	return &s
}

func (s *documentServer) init() {
	s.Get("/lookup/name/{Name}", s.lookupName)
	s.Get("/{ShelfID}", s.showShelf)
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

	id, ok, err := s.client.LookupShelfByName(r.Context(), name)
	if err != nil {
		api.Error(w, r, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		api.Error(w, r, http.StatusNotFound, api.Friendly(nil, "No shelf named %q found.", name))
	}
	resp.ShelfID = id

	api.JSON(w, r, http.StatusOK, resp)
}

func (s *documentServer) showShelf(w http.ResponseWriter, r *http.Request) {
	id, err := api.ExtractUUID(r, "ShelfID")
	if err != nil {
		api.Error(w, r, http.StatusBadRequest, err)
		return
	}

	shelf, err := s.client.FetchShelf(r.Context(), id)
	if err != nil {
		api.Error(w, r, http.StatusNotFound, api.Friendly(err, "Shelf %q not found: %v.", id, err))
		return
	}

	api.JSON(w, r, http.StatusOK, shelf)
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

	doc, err := s.client.UploadDocument(r.Context(), shelfID, file, uniqueName, name, disk, path)
	if err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to upload document to shelf: %v", err))
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

	replaced, err := s.client.ReplaceDocument(r.Context(), shelfID, documentID, file)
	if err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to replace document: %v", err))
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

	cmd := document.Rename(shelfID, documentID, req.Name)
	if err := s.commands.Dispatch(r.Context(), cmd, dispatch.Synchronous()); err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to dispatch %q command: %v", cmd.Name(), err))
		return
	}

	if req.UniqueName != nil {
		if *req.UniqueName != "" {
			cmd = document.MakeUnique(shelfID, documentID, *req.UniqueName)
		} else {
			cmd = document.MakeNonUnique(shelfID, documentID)
		}

		if err := s.commands.Dispatch(r.Context(), cmd, dispatch.Synchronous()); err != nil {
			api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to dispatch %q command: %v", cmd.Name(), err))
			return
		}
	}

	shelf, err := s.client.FetchShelf(r.Context(), shelfID)
	if err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Shelf %q not found.", shelfID))
		return
	}

	doc, err := shelf.Document(documentID)
	if err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Document %q not found.", documentID))
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

	cmd := document.Remove(shelfID, documentID)
	if err := s.commands.Dispatch(r.Context(), cmd, dispatch.Synchronous()); err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to remove document: %v", err))
		return
	}

	api.NoContent(w, r)
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

	cmd := document.Tag(shelfID, documentID, req.Tags)
	if err := s.commands.Dispatch(r.Context(), cmd, dispatch.Synchronous()); err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to dispatch %q command: %v", cmd.Name(), err))
		return
	}

	shelf, err := s.client.FetchShelf(r.Context(), shelfID)
	if err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Shelf %q not found.", shelfID))
	}

	doc, err := shelf.Document(documentID)
	if err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Document %q not found.", documentID))
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

	cmd := document.Untag(shelfID, documentID, tags)
	if err := s.commands.Dispatch(r.Context(), cmd, dispatch.Synchronous()); err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to dispatch %q command: %v", cmd.Name(), err))
		return
	}

	shelf, err := s.client.FetchShelf(r.Context(), shelfID)
	if err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Shelf %q not found.", shelfID))
	}

	doc, err := shelf.Document(documentID)
	if err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Document %q not found.", documentID))
		return
	}

	api.JSON(w, r, http.StatusOK, doc)
}

type galleryServer struct {
	chi.Router

	client   Client
	commands command.Bus
}

func newGalleryServer(client Client, commands command.Bus) *galleryServer {
	srv := galleryServer{
		Router:   chi.NewRouter(),
		client:   client,
		commands: commands,
	}
	srv.init()
	return &srv
}

func (s *galleryServer) init() {
	s.Get("/lookup/name/{Name}", s.lookupName)
	s.Get("/{GalleryID}", s.showGallery)
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

	id, ok, err := s.client.LookupGalleryByName(r.Context(), name)
	if err != nil {
		api.Error(w, r, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		api.Error(w, r, http.StatusNotFound, api.Friendly(nil, "Lookup failed for gallery %q.", name))
		return
	}
	resp.GalleryID = id

	api.JSON(w, r, http.StatusOK, resp)
}

func (s *galleryServer) showGallery(w http.ResponseWriter, r *http.Request) {
	id, err := api.ExtractUUID(r, "GalleryID")
	if err != nil {
		api.Error(w, r, http.StatusBadRequest, err)
		return
	}

	g, err := s.client.FetchGallery(r.Context(), id)
	if err != nil {
		api.Error(w, r, http.StatusNotFound, api.Friendly(err, "Gallery %q not found: %v.", id, err))
	}

	api.JSON(w, r, http.StatusOK, g)
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

	stack, err := s.client.UploadImage(r.Context(), galleryID, file, name, disk, path)
	if err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to upload image: %v", err))
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

	cmd := gallery.DeleteStack(galleryID, stackID)
	if err := s.commands.Dispatch(r.Context(), cmd, dispatch.Synchronous()); err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to dispatch %q command: %v", cmd.Name(), err))
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

	cmd := gallery.TagStack(galleryID, stackID, req.Tags)
	if err := s.commands.Dispatch(r.Context(), cmd, dispatch.Synchronous()); err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to dispatch %q command: %v", cmd.Name(), err))
		return
	}

	g, err := s.client.FetchGallery(r.Context(), galleryID)
	if err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Gallery %q not found: %v", galleryID, err))
		return
	}

	stack, err := g.Stack(stackID)
	if err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Stack %q not found.", stackID))
	}

	api.JSON(w, r, http.StatusCreated, stack)
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

	cmd := gallery.UntagStack(galleryID, stackID, tags)
	if err := s.commands.Dispatch(r.Context(), cmd, dispatch.Synchronous()); err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to dispatch %q command: %v", cmd.Name(), err))
		return
	}

	g, err := s.client.FetchGallery(r.Context(), galleryID)
	if err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Gallery %q not found: %v", galleryID, err))
		return
	}

	stack, err := g.Stack(stackID)
	if err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Stack %q not found.", stackID))
	}

	api.JSON(w, r, http.StatusCreated, stack)
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

	replaced, err := s.client.ReplaceImage(r.Context(), galleryID, stackID, file)
	if err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to replace image: %v", err))
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

	if req.Name != "" {
		cmd := gallery.RenameStack(galleryID, stackID, req.Name)
		if err := s.commands.Dispatch(r.Context(), cmd, dispatch.Synchronous()); err != nil {
			api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Failed to dispatch %q command: %v", cmd.Name(), err))
			return
		}
	}

	g, err := s.client.FetchGallery(r.Context(), galleryID)
	if err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Gallery %q not found: %v", galleryID, err))
		return
	}

	stack, err := g.Stack(stackID)
	if err != nil {
		api.Error(w, r, http.StatusInternalServerError, api.Friendly(err, "Stack %q not found.", stackID))
	}

	api.JSON(w, r, http.StatusOK, stack)
}
