package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// All is a wildcard for all routes.
var All = route("*", "*")

// Gallery routes
var (
	LookupGalleryByName      = route("GET", "/galleries/lookup/name/{Name}")
	LookupGalleryStackByName = route("GET", "/galleries/{GalleryID}/lookup/stack-name/{Name}")
	ShowGallery              = route("GET", "/galleries/{GalleryID}")
	UploadImage              = route("POST", "/galleries/{GalleryID}/stacks")
	ReplaceImage             = route("PUT", "/galleries/{GalleryID}/stacks/{StackID}")
	UpdateStack              = route("PATCH", "/galleries/{GalleryID}/stacks/{StackID}")
	DeleteStack              = route("DELETE", "/galleries/{GalleryID}/stacks/{StackID}")
	TagStack                 = route("POST", "/galleries/{GalleryID}/stacks/{StackID}/tags")
	UntagStack               = route("DELETE", "/galleries/{GalleryID}/stacks/{StackID}/tags/{Tags}")
	SortGallery              = route("PATCH", "/galleries/{GalleryID}/sorting")

	GalleryReadRoutes = [...]Route{
		LookupGalleryByName,
		LookupGalleryStackByName,
		ShowGallery,
	}

	GalleryWriteRoutes = [...]Route{
		UploadImage,
		ReplaceImage,
		UpdateStack,
		DeleteStack,
		TagStack,
		UntagStack,
		SortGallery,
	}

	GalleryRoutes = [...]Route{
		LookupGalleryByName,
		LookupGalleryStackByName,
		ShowGallery,
		UploadImage,
		ReplaceImage,
		UpdateStack,
		DeleteStack,
		TagStack,
		UntagStack,
	}
)

// Document routes
var (
	LookupShelfByName = route("GET", "/shelfs/lookup/name/{Name}")
	ShowShelf         = route("GET", "/shelfs/{ShelfID}")
	UploadDocument    = route("POST", "/shelfs/{ShelfID}/documents")
	ReplaceDocument   = route("PUT", "/shelfs/{ShelfID}/documents/{DocumentID}")
	UpdateDocument    = route("PATCH", "/shelfs/{ShelfID}/documents/{DocumentID}")
	DeleteDocument    = route("DELETE", "/shelfs/{ShelfID}/documents/{DocumentID}")
	TagDocument       = route("POST", "/shelfs/{ShelfID}/documents/{DocumentID}/tags")
	UntagDocument     = route("DELETE", "/shelfs/{ShelfID}/documents/{DocumentID}/tags/{Tags}")

	DocumentReadRoutes = [...]Route{
		LookupShelfByName,
		ShowShelf,
	}

	DocumentWriteRoutes = [...]Route{
		UploadDocument,
		ReplaceDocument,
		UpdateDocument,
		DeleteDocument,
		TagDocument,
		UntagDocument,
	}

	DocumentRoutes = [...]Route{
		LookupShelfByName,
		ShowShelf,
		UploadDocument,
		ReplaceDocument,
		UpdateDocument,
		DeleteDocument,
		TagDocument,
		UntagDocument,
	}
)

// Route is a route with a method and path.
type Route struct {
	Method string
	Path   string
}

// Routes configures the routes for one of the media components.
type Routes struct {
	disabled   []Route
	middleware map[Route][]func(http.Handler) http.Handler
}

// Option is a Routes option.
type Option func(*Routes)

// Disable disables the provided routes.
func Disable(routes ...Route) Option {
	return func(r *Routes) {
		r.disabled = append(r.disabled, routes...)
	}
}

// Middleware adds middleware to the given routes. If routes is empty, the
// middleware is added to all routes.
func Middleware(middleware func(http.Handler) http.Handler, routes ...Route) Option {
	if len(routes) == 0 {
		routes = []Route{All}
	}
	return func(r *Routes) {
		for _, route := range routes {
			r.middleware[route] = append(r.middleware[route], middleware)
		}
	}
}

// Middlewares adds multiple middlewares to the given routes. If routes is
// empty, the middleware is added to all routes.
func Middlewares(middlewares []func(http.Handler) http.Handler, routes ...Route) Option {
	if len(routes) == 0 {
		routes = []Route{All}
	}
	return func(r *Routes) {
		for _, route := range routes {
			r.middleware[route] = append(r.middleware[route], middlewares...)
		}
	}
}

// New returns a route configuration.
func New(opts ...Option) Routes {
	r := Routes{middleware: make(map[Route][]func(http.Handler) http.Handler)}
	for _, opt := range opts {
		opt(&r)
	}
	return r
}

// Disabled returns whether the given Route is disabled.
func (r Routes) Disabled(route Route) bool {
	for _, d := range r.disabled {
		if route == d || d == All {
			return true
		}
	}
	return false
}

// Middleware returns the middleare for the given Route.
func (r Routes) Middleware(route Route) []func(http.Handler) http.Handler {
	return append(r.middleware[All], r.middleware[route]...)
}

// Install installs the routes in the given Router, using the provided Handler,
// but only if the Route wasn't disabled.
func (r Routes) Install(router chi.Router, route Route, h http.Handler) {
	if !r.Disabled(route) {
		router.With(r.Middleware(route)...).Method(route.Method, route.Path, h)
	}
}

func route(method, path string) Route {
	return Route{Method: method, Path: path}
}
