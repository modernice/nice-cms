package media

import "errors"

var (
	// ErrUnconfiguredDisk is returned when trying to get a StorageDisk that
	// isn't configured.
	ErrUnconfiguredDisk = errors.New("unconfigured disk")

	// ErrFileNotFound is returned when a file cannot be found in a Storage.
	ErrFileNotFound = errors.New("file not found")

	// ErrUnknownImage is returned when an image does not exist in the ImageRepository.
	ErrUnknownImage = errors.New("unknown image")

	// ErrUploadFailed is returned when an upload of a file fails.
	ErrUploadFailed = errors.New("upload failed")
)
