# Media management

## Introduction

A typical website needs various assets that can be presented in on-page. These
assets may be images, videos or documents.

In a standard web development process, assets are being incorporated by the
developer who builds the site. During this process, developers hard-code not
only the text content, but also the images into the sites. They may also use
"low-level" technologies/libraries that optimize those assets during the build
step. Such hard-coded content is only ever editable by a developer, not by a
website owner.

## Images

Website owners need to be able to **upload and manage images** and **integrate
them into the content** of their pages and posts.

### Image processing

Images must be automatically send through a **processing pipeline** that resizes
images into different, configurable sizes.

### Metadata

Images need metadata that can be managed by the website owner:

- name
- alt description (for SEO purposes)
- dimensions (uneditable / automatically detected)
- filesize (uneditable / automatically detected)

## Videos

Just like with images, website owners need to **upload and manage videos** that
they can integrate in actual content of pages or posts.

### Metadata

Videos need metadata that can be managed by the website owner:

- name
- dimensions (uneditable / automatically detected)
- filesize (uneditable / automatically detected)
- duration (uneditable / automatically detected)

## Documents

Documents are any files besides images or videos. Website owners need to be able
to upload any kind of document so that they can use them or refer to them in
actual content.

### Metadata

- name
- alt description (for SEO purposes)
- filesize (uneditable / automatically detected)

## Tagging

All asset types must be taggable with arbitrary,
[user-managable tags](./tagging.md).

## Storage & CDN

GCS, AWS, Azure & filesystem storage implementations should be provided together
with their corresponding CDN implementations.

## [Galleries](./galleries.md)

End-users should access and manage images through a gallery. Galleries provide
some additional features around simple image uploads like image processing.

## Code examples

### Setup image service

```go
package example

func NewImageService(images image.Repository, storage media.Storage) *image.Service {
  return image.NewService(images, storage)
}
```

### Setup media server

```go
package example

func NewMediaServer() http.Handler {
  images := image.NewService(...)
  videos := video.NewService(...)
  docs := document.NewService(...)

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
