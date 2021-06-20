package media

//go:generate mockgen -source=media.go -destination=./mock_media/media.go

import (
	"context"
	"image"
	"io"
)

// ImageService provides uploads and management of images.
type ImageService interface {
	// Upload uploads the image in Reader to the given storage disk and path
	// with the provided name and returns the uploaded Image.
	//
	// If the upload fails, ErrUploadFailed is returned.
	Upload(_ context.Context, _ io.Reader, name, disk, path string) (Image, error)

	// Download downloads the Image from the specified disk and path.
	//
	// Download returns ErrFileNotFound if the file does not exist in the
	// storage.
	Download(_ context.Context, disk, path string) (image.Image, string, error)

	// Replace replaces the image at the specified disk and path with the image
	// in Reader and returns the updated Image.
	Replace(_ context.Context, _ io.Reader, disk, path string) (Image, error)

	// Rename renames the image at the specified disk and path to the given name
	// and returns the updated Image.
	Rename(_ context.Context, name, disk, path string) (Image, error)

	// Delete deletes the image at the specified disk and path. Delete returns
	// ErrFileNotFound if the image does not exist.
	Delete(_ context.Context, disk, path string) error

	// Tag tags the specified image with the given tags and returns the updated
	// Image.
	Tag(_ context.Context, disk, path string, tags ...string) (Image, error)

	// Untag removes the given tags from the specified image and returns the
	// updated Image.
	Untag(_ context.Context, disk, path string, tags ...string) (Image, error)
}

// DocumentService provides upload and management of Documents.
type DocumentService interface {
	// Upload uploads the document in Reader to the given storage disk and path
	// with the provided documentName and filename and returns the uploaded
	// Document.
	//
	// If the upload fails, ErrUploadFailed is returned.
	Upload(_ context.Context, documentName, filename, disk, path string, filesize int) (Document, error)

	// Download downloads the Document from the specified disk and path.
	//
	// Download returns ErrFileNotFound if the file does not exist in the
	// storage.
	Download(context.Context, Document) ([]byte, string, error)

	// Replace replaces the document at the specified disk and path with the
	// document in Reader and returns the updated Document.
	Replace(_ context.Context, _ io.Reader, disk, path string) (Image, error)

	// Rename renames the document at the specified disk and path to the given name
	// and returns the updated Document.
	Rename(_ context.Context, name, disk, path string) (Image, error)

	// Delete deletes the document at the specified disk and path. Delete returns
	// ErrFileNotFound if the document does not exist.
	Delete(context.Context, Document) error

	// Tag tags the specified document with the given tags and returns the
	// updated Document.
	Tag(_ context.Context, disk, path string, tags ...string) (Document, error)

	// Untag removes the given tags from the specified document and returns the
	// updated Document.
	Untag(_ context.Context, disk, path string, tags ...string) (Document, error)
}

// ImageEncoder encodes images, using the appropriate encoder for the specified
// image format.
type ImageEncoder interface {
	Encode(io.Writer, image.Image, string) error
}

// File is a file that is stored in a storage backend.
type File struct {
	Name     string
	Disk     string
	Path     string
	Filesize int
	Tags     []string
}

// NewFile returns a File with the specified data. NewFile ensures that returned
// File f has a non-nil f.Tags slice.
func NewFile(name, disk, path string, size int, tags ...string) File {
	if tags == nil {
		tags = make([]string, 0)
	}
	return File{
		Name:     name,
		Disk:     disk,
		Path:     path,
		Filesize: size,
		Tags:     tags,
	}
}

// WithTag adds the given tags and returns the updated File.
func (f File) WithTag(tags ...string) File {
	for _, tag := range tags {
		var found bool
		for _, ftag := range f.Tags {
			if ftag == tag {
				found = true
				break
			}
		}

		if !found {
			f.Tags = append(f.Tags, tag)
		}
	}

	return f
}

// WithoutTags removes the given tags and returns the updated File.
func (f File) WithoutTag(tags ...string) File {
	for _, tag := range tags {
		for i, ftag := range f.Tags {
			if ftag != tag {
				continue
			}
			f.Tags = append(f.Tags[:i], f.Tags[i+1:]...)
			break
		}
	}
	return f
}

// HasTag returns whether the File has the given tags. HasTag returns true if
// the File has all provided tags or if len(tags) == 0.
func (f File) HasTag(tags ...string) bool {
	for _, tag := range tags {
		var found bool
		for _, ftag := range f.Tags {
			if ftag == tag {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// Same returns true if other points to the same file as f (same Disk and Path).
func (f File) Same(other File) bool {
	return f.Disk == other.Disk && f.Path == other.Path
}

// Image is storage image.
type Image struct {
	File

	Width  int
	Height int
}

// NewImage returns an Image with the given data.
func NewImage(width, height int, name, disk, path string, filesize int) Image {
	return Image{
		File:   NewFile(name, disk, path, filesize),
		Width:  width,
		Height: height,
	}
}

// Document is an arbitrary storage file.
type Document struct {
	File

	// DocumentName is the unique name of the document. DocumentName may be
	// empty but if it is not, it must be unique.
	DocumentName string
}

// NewDocument returns a Document with the given data.
func NewDocument(documentName, name, disk, path string, filesize int) Document {
	return Document{
		File:         NewFile(name, disk, path, filesize),
		DocumentName: documentName,
	}
}
