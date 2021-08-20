package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

var All = route("*", "*")

// Gallery routes
var (
	LookupGalleryByName      = route("GET", "/lookup/name/{Name}")
	LookupGalleryStackByName = route("GET", "/{GalleryID}/lookup/stack-name/{Name}")
	ShowGallery              = route("GET", "/{GalleryID}")
	UploadImage              = route("POST", "/{GalleryID}/stacks")
	ReplaceImage             = route("PUT", "/{GalleryID}/stacks/{StackID}")
	UpdateStack              = route("PATCH", "/{GalleryID}/stacks/{StackID}")
	DeleteStack              = route("DELETE", "/{GalleryID}/stacks/{StackID}")
	TagStack                 = route("POST", "/{GalleryID}/stacks/{StackID}/tags")
	UntagStack               = route("DELETE", "/{GalleryID}/stacks/{StackID}/tags/{Tags}")
)

// Document routes
var (
	LookupShelfByName = route("GET", "/lookup/name/{Name}")
	ShowShelf         = route("GET", "/{ShelfID}")
	UploadDocument    = route("POST", "/{ShelfID}/documents")
	ReplaceDocument   = route("PUT", "/{ShelfID}/documents/{DocumentID}")
	UpdateDocument    = route("PATCH", "/{ShelfID}/documents/{DocumentID}")
	DeleteDocument    = route("DELETE", "/{ShelfID}/documents/{DocumentID}")
	TagDocument       = route("POST", "/{ShelfID}/documents/{DocumentID}/tags")
	UntagDocument     = route("DELETE", "/{ShelfID}/documents/{DocumentID}/tags/{Tags}")
)

type Route struct {
	Method string
	Path   string
}

type Routes struct {
	disabled   []Route
	middleware map[Route][]func(http.Handler) http.Handler
}

type Option func(*Routes)

func Disable(routes ...Route) Option {
	return func(r *Routes) {
		r.disabled = append(r.disabled, routes...)
	}
}

func Middleware(route Route, middleware ...func(http.Handler) http.Handler) Option {
	return func(r *Routes) {
		r.middleware[route] = append(r.middleware[route], middleware...)
	}
}

func New(opts ...Option) Routes {
	var r Routes
	for _, opt := range opts {
		opt(&r)
	}
	return r
}

func (r Routes) Disabled(route Route) bool {
	for _, d := range r.disabled {
		if route == d || d == All {
			return true
		}
	}
	return false
}

func (r Routes) Middleware(route Route) []func(http.Handler) http.Handler {
	return append(r.middleware[All], r.middleware[route]...)
}

func (r Routes) Install(router chi.Router, route Route, h http.Handler) {
	if !r.Disabled(route) {
		router.With(r.Middleware(route)...).Method(route.Method, route.Path, h)
	}
}

func route(method, path string) Route {
	return Route{Method: method, Path: path}
}
