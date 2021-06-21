package media

//go:generate mockgen -source=media.go -destination=./mock_media/media.go

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"io"
)

// // DocumentService provides upload and management of Documents.
// type DocumentService interface {
// 	// Upload uploads the document in Reader to the given storage disk and path
// 	// with the provided documentName and filename and returns the uploaded
// 	// Document.
// 	//
// 	// If the upload fails, ErrUploadFailed is returned.
// 	Upload(_ context.Context, documentName, filename, disk, path string, filesize int) (Document, error)

// 	// Download downloads the Document from the specified disk and path.
// 	//
// 	// Download returns ErrFileNotFound if the file does not exist in the
// 	// storage.
// 	Download(context.Context, Document) ([]byte, string, error)

// 	// Replace replaces the document at the specified disk and path with the
// 	// document in Reader and returns the updated Document.
// 	Replace(_ context.Context, _ io.Reader, disk, path string) (Image, error)

// 	// Rename renames the document at the specified disk and path to the given name
// 	// and returns the updated Document.
// 	Rename(_ context.Context, name, disk, path string) (Image, error)

// 	// Delete deletes the document at the specified disk and path. Delete returns
// 	// ErrFileNotFound if the document does not exist.
// 	Delete(context.Context, Document) error

// 	// Tag tags the specified document with the given tags and returns the
// 	// updated Document.
// 	Tag(_ context.Context, disk, path string, tags ...string) (Document, error)

// 	// Untag removes the given tags from the specified document and returns the
// 	// updated Document.
// 	Untag(_ context.Context, disk, path string, tags ...string) (Document, error)
// }

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

// Upload uploads the file to storage and returns the File with updated Filesize.
func (f File) Upload(ctx context.Context, r io.Reader, storage Storage) (File, error) {
	disk, err := f.storageDisk(storage)
	if err != nil {
		return f, err
	}

	b, err := io.ReadAll(r)
	if err != nil {
		return f, fmt.Errorf("read file: %w", err)
	}

	if err := disk.Put(ctx, f.Path, b); err != nil {
		return f, fmt.Errorf("upload to %q storage: %w", f.Disk, err)
	}
	f.Filesize = len(b)

	return f, nil
}

// Download downloads the file from the provided Storage.
func (f File) Download(ctx context.Context, storage Storage) ([]byte, error) {
	disk, err := f.storageDisk(storage)
	if err != nil {
		return nil, err
	}
	return disk.Get(ctx, f.Path)
}

// Replace replaces the file in storage with the contents in r and returns the
// updated File.
func (f File) Replace(ctx context.Context, r io.Reader, storage Storage) (File, error) {
	return f.Upload(ctx, r, storage)
}

// Delete deletes the file from storage.
func (f File) Delete(ctx context.Context, storage Storage) error {
	disk, err := f.storageDisk(storage)
	if err != nil {
		return err
	}
	return disk.Delete(ctx, f.Path)
}

func (f File) storageDisk(storage Storage) (StorageDisk, error) {
	disk, err := storage.Disk(f.Disk)
	if err != nil {
		return nil, fmt.Errorf("get %q storage disk: %w", f.Disk, err)
	}
	return disk, nil
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

// WithTag adds the given tags and returns the updated Image.
func (img Image) WithTag(tags ...string) Image {
	img.File = img.File.WithTag(tags...)
	return img
}

// WithoutTags removes the given tags and returns the updated Image.
func (img Image) WithoutTag(tags ...string) Image {
	img.File = img.File.WithoutTag(tags...)
	return img
}

// Upload uploads the image to storage and returns the Image with updated
// Filesize, Width and Height.
func (img Image) Upload(ctx context.Context, r io.Reader, storage Storage) (Image, error) {
	var buf bytes.Buffer
	r = io.TeeReader(r, &buf)

	f, err := img.File.Upload(ctx, r, storage)
	if err != nil {
		return img, err
	}
	img.File = f

	stdimg, _, err := image.Decode(&buf)
	if err != nil {
		return img, fmt.Errorf("decode image: %w", err)
	}

	img.Width = stdimg.Bounds().Dx()
	img.Height = stdimg.Bounds().Dy()

	return img, nil
}

// Download downloads the image from the provided Storage and returns the
// decoded Image and its format. If you want to download the image without
// decoding it into an image.Image, use img.File.Download instead.
func (img Image) Download(ctx context.Context, storage Storage) (image.Image, string, error) {
	b, err := img.File.Download(ctx, storage)
	if err != nil {
		return nil, "", err
	}
	return image.Decode(bytes.NewReader(b))
}

// Replace replaces the image in Storage with the image in r and returns the
// updated Image.
func (img Image) Replace(ctx context.Context, r io.Reader, storage Storage) (Image, error) {
	return img.Upload(ctx, r, storage)
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

// WithTag adds the given tags and returns the updated Document.
func (doc Document) WithTag(tags ...string) Document {
	doc.File = doc.File.WithTag(tags...)
	return doc
}

// WithoutTags removes the given tags and returns the updated Document.
func (doc Document) WithoutTag(tags ...string) Document {
	doc.File = doc.File.WithoutTag(tags...)
	return doc
}

// Upload uploads the document to storage and returns the Document with updated Filesize.
func (doc Document) Upload(ctx context.Context, r io.Reader, storage Storage) (Document, error) {
	f, err := doc.File.Upload(ctx, r, storage)
	if err != nil {
		return doc, err
	}
	doc.File = f
	return doc, nil
}

// Replace replaces the document in Storage with the document in r and returns
// the updated Document.
func (doc Document) Replace(ctx context.Context, r io.Reader, storage Storage) (Document, error) {
	return doc.Upload(ctx, r, storage)
}
