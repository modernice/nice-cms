package media

//go:generate mockgen -source=media.go -destination=./mock_media/media.go

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"io"
)

// File is a file that is stored in a storage backend.
type File struct {
	Name     string   `json:"name"`
	Disk     string   `json:"disk"`
	Path     string   `json:"path"`
	Filesize int      `json:"filesize"`
	Tags     []string `json:"tags"`
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

	Width  int `json:"width"`
	Height int `json:"height"`
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

	cfg, _, err := image.DecodeConfig(r)
	if err != nil {
		return img, fmt.Errorf("decode image: %w", err)
	}

	img.Width = cfg.Width
	img.Height = cfg.Height

	f, err := img.File.Upload(ctx, &buf, storage)
	if err != nil {
		return img, err
	}
	img.File = f

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
}

// NewDocument returns a Document with the given data.
func NewDocument(name, disk, path string, filesize int) Document {
	return Document{
		File: NewFile(name, disk, path, filesize),
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
