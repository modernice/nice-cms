# Blog

## Posts

Blog posts may differ slightly in structure between different blogs because of
different requirements, but the basic structure of a blog post is mostly the
same. A post must have a title, the actual content (as HTML) and publishing
info (publishAt, publisher etc.). Additional fields should be available.

```go
package example

func NewDefaultBlog() *blog.Blog {
	return blog.New() // blog with sensible defaults
}

func NewCustomBlog() *blog.Blog {
	return blog.New(
		blog.WithAuthors(), // add author information to posts

		blog.WithField(
			field.Toggle("featured", func(f *field.Toggle) bool {
				return false
			}),
		),
	)
}
```
