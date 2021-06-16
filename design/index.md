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

### Dynamic content management

- create/delete/update/publish/unpublish reviews

### Media management

- upload/delete/tag/update images/videos/documents
- image processing (resizing)
- optional CDN integration

### Frontend tooling

- typed api
- query tools
- testing tools (?)

### Non-goals

- authorization (should be implemented by the developers themselves)
- visual content builder
- web UI

## Detailed Design

- [Media](./media.md)
- [Navigations](./navigations.md)
- [Static web pages](./static.md)
- [Blogs](./blogs.md)
- [Reviews](./reviews.md)
- [API](./api.md)
