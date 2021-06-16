package media

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
	Download(_ context.Context, disk, path string) (image.Image, error)

	// Replace replaces the image at the specified disk and path with the image
	// in Reader and returns the updated Image.
	//
	// Replace returns ErrUnknownImage if the image does not exist in the
	// ImageRepository or ErrUploadFailed if the upload of the new image fails.
	Replace(_ context.Context, _ io.Reader, disk, path string) (Image, error)

	// Rename renames the image at the specified disk and path to the given name
	// and returns the renamed Image.
	//
	// Rename returns ErrUnknownImage if the image does not exist in the
	// ImageRepository.
	Rename(_ context.Context, name, disk, path string) (Image, error)

	// Delete deletes the image at the specified disk and path. Delete returns
	// ErrFileNotFound if the image does not exist.
	Delete(_ context.Context, disk, path string) error

	// Tag tags the specified image with the given tags and returns the updated
	// Image. Tag returns ErrUnknownImage if the image does not exist.
	Tag(_ context.Context, disk, path string, tags ...string) (Image, error)

	// Untag removes the given tags from the specified image and returns the
	// updated Image. Untag returns ErrUnknownImage if the image does not exist.
	Untag(_ context.Context, disk, path string, tags ...string) (Image, error)
}

// Image is an uploaded image.
type Image struct {
	File

	Width  int
	Height int
}
