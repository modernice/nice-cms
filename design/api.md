# API

The API service must provide API access to the different modules (static pages,
blog etc.).

## Requirements

- access to static page & navigation management
- access to media management
- access to review management
-

The API **must not** handle authentication. That should be done by developers.

## Design

```go
package example

func NewAPI(
	static http.Handler,
	blogs http.Handler,
	media http.Handler,
	reviews http.Handler,
) http.Handler {
	return api.New(
		api.WithStatic(static),
		api.WithBlogs(blogs),
		api.WithMedia(media),
		api.WithReviews(reviews),
	)
}
```
