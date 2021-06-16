package media

import "errors"

var (
	// ErrUnconfiguredDisk is returned when trying to get a StorageDisk that
	// isn't configured.
	ErrUnconfiguredDisk = errors.New("unconfigured disk")

	// ErrFileNotFound is returned when a file cannot be found in a Storage.
	ErrFileNotFound = errors.New("file not found")

	// ErrUploadFailed is returned when an upload of a file fails.
	ErrUploadFailed = errors.New("upload failed")

	// ErrEmptyName is returned when providing an empty string as a name.
	// Whitespce-only strings count as empty.
	ErrEmptyName = errors.New("empty name")
)
