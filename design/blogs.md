# Blogs

Blog posts may differ slightly in structure between different blogs because of
different requirements, but the basic structure of a blog post is mostly the
same. A post must have a title, the actual content (as HTML) and publishing
info (publishAt, publisher etc.). Additional fields should be available to
developers.

## Requirements

- posts must be able to be referenced by navigations
- 

## Design

```go
package example

func NewDefaultBlog(posts blog.PostRepository) *blog.Blog {
	return blog.New("blog-name", posts) // blog with sensible defaults
}

func NewCustomBlog(posts blog.PostRepository) *blog.Blog {
	return blog.New(
		"blog-name",
		posts,

		blog.WithAuthors(), // add author information to posts

		blog.Localized("de", "en", "fr"), // add localization

		blog.WithField( // add custom fields to posts
			field.NewToggle("featured", func(f *field.Toggle) bool {
				return false
			}),
		),
	)
}
```

```go
package example

func NewBlogServer(blogs ...*blog.Blog) http.Handler {
	return blog.NewHTTPServer(blogs...)
}
```
