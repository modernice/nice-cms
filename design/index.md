# cms

`cms` is to become a developer-focused, headless content management toolkit that
can be used to build content management systems of any kind.

## Background

Most web projects, regardless of size or complexity, need some kind of content
management tool that allows website owners to maintain the content of their
sites. This could involve publishing blog posts or event just quick typo fixes.
The problem with these kinds of content changes are that they can be implemented
technically in very different ways, and static pages are rarely considered to be
editable by the website owner, although this is a need that arises with most
clients and is certainly something a website owner wishes to be able to do.

**It is unreasonable to expect website owners to pay a developer for every little
change they want to apply to their website's content.**

## Goals

The CMS should primarily be focused on developers and providing a development
experience that makes it trivial to build content management backends. At the
same time, it shouldn't constrain developers when building larger, more complex
websites or apps. This means that the CMS should be easily integratable
into an existing codebase and not force a specific coding style, code structure
or deployment method.

### Static content management

- manage navigation trees
  - create/delete/update trees
  - guarded trees (not deletable by website owner)
  - import/export of trees
- manage pages
  - enable/disable pages
  - update content
  - metadata
    - title
    - description
    - meta tags (OpenGraph etc.)
  - multilingual
  - import/export

### Blog management

- create/delete/update/publish/unpublish posts
- create/delete/update categories
- create/delete/update authors

### Media management

- upload/delete/tag/update images/videos/documents
- image processing (resizing)
- optional CDN integration

### Frontend tooling

- static site management
- blog management
- media management
- query tools
- testing tools
- content editor

### Non-goals

- authorization (should be implemented by the developers themselves)
- visual content builder
- pre-built web UI
- schema building (developers define schemas themselves through provided
  modules)

## Detailed Design

### Navigation trees

Developers create navigation trees and query them from the frontend to display
and integrate them on the website. A tree is a recursive list, where each list
item may itself be its own list.

A navigation item may be of the following types:

- link to static page
  - without subtree
  - with subtree
- link to blog post (without subtree)
- label only (with subtree)

A navigation tree is identified by a unique name (string) and can be guarded
from being deleted.

On the frontend, developers can query navigation trees by their name to

- build the view for website users.
- build the management for website owners.

Developers do not have to code data structures to implement navigation trees,
because navigation tree logic doesn't need to be customizable.

### Static pages

Static pages can have different requirements, depending on the website. What
kind of content a page needs must be decided by the developer who implements the
static page data structure.

A page may require various different fields of different types. Developers
create the data structures for static pages by embedding content modules in
their structures:

```go
package example

type ContactPage struct {
  *static.Page

  // Fields provided by *static.Page
  ID uuid.UUID
  Slugs *field.LocalizedText
  Titles *field.LocalizedText
  Descriptions *field.LocalizedText
  Meta *field.LocalizedMeta
  Enabled bool

  ContactMail *field.Text
  ContactPhone *field.Text
}

func NewContactPage() *ContactPage {
  contactMail := field.NewText()
  contactPhone := field.NewText()

  return &ContactPage{
    Page: static.NewPage(contactMail, contactPhone),
    ContactMail: contactMail,
    ContactPhone: contactPhone,
  }
}
```

### Blog posts

A blog post differs from static pages only by the fields in the data structure
that the developer builds. Pre-built structures should be provided for this:

```go
package example

type BlogPost struct {
  *blog.Post

  // Fields provided by *blog.Post
  ID uuid.UUID
  Categories []uuid.UUID
  Slugs *field.LocalizedText
  Titles *field.LocalizedText
  Descriptions *field.LocalizedText
  Meta *field.LocalizedMeta
  PublishedAt time.Time
  PublishedBy blog.Publisher

  Content *field.LocalizedHTML
}

func NewBlogPost() *BlogPost {
  content := field.NewLocalizedHTML()

  return &BlogPost{
    Post: blog.NewPost(content),
    Content: content,
  }
}
```

Blog categories are fully implemented by the toolkit and therefore do not need
custom data structures.

### Blog authors

A blog author is a person who publishes blog posts. Information about the author
can be displayed beside blog posts.

Developers create custom data structures for authors:

```go
package example

type BlogAuthor struct {
  *blog.Author

  // Fields provided by *blog.Author
  Name *field.LocalizedText
  Description *field.LocalizedText // optional description about the author
  Image blog.AuthorImage

  Email *field.Text
}

func NewBlogAuthor() *BlogAuthor {
  email := field.NewText()

  return &BlogAuthor{
    Author: blog.NewAuthor(email),
    Email: email,
  }
}
```

### Media management

Media management is provided pre-built. Developers can use the pre-built media
management to provide their clients with feature-complete media management.

```go
package example

func NewMediaServer() http.Handler {
  images := media.NewImageService(...)
  videos := media.NewVideoService(...)
  docs := media.NewDocumentService(...)

  return media.NewHTTPServer(
    media.WithImageService(images),
    media.WithVideoService(videos),
    media.WithDocumentService(docs),
  )
}
```

```sh
POST /images # upload images
DELETE /images/{ID} # delete image
PUT /images/{ID} # replace image
PATCH /images/{ID} # update image (rename etc.)

POST /videos # upload videos
DELETE /videos/{ID} # delete video
PUT /videos/{ID} # replace video
PATCH /videos/{ID} # update video (rename etc.)

POST /documents # upload documents
DELETE /documents/{ID} # delete document
PUT /documents/{ID} # replace document
PATCH /documents/{ID} # update document (rename etc.)
```
