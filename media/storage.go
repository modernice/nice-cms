package media

//go:generate mockgen -source=storage.go -destination=./mock_media/storage.go

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/bounoable/godrive"
)

// Storage is a storage for files.
type Storage interface {
	// Disk returns the StorageDisk that was configured with the given name or
	// ErrUnconfiguredDisk if the disk wasn't configured.
	Disk(string) (StorageDisk, error)
}

// StorageDisk is a disk of a Storage.
type StorageDisk interface {
	// Put uploads a file to the specified storage path or ErrFileNotFound if
	// the file does not exist.
	Put(context.Context, string, []byte) error

	// Get returns the contents of the file at the specified path or
	// ErrFileNotFound if the file does not exist.
	Get(context.Context, string) ([]byte, error)

	// Delete deletes the file at the specified path. Delete returns no error
	// if the specified file does not exist.
	Delete(context.Context, string) error
}

// File is a file that is stored in a storage backend.
type File struct {
	Name string
	Disk string
	Path string
	Size int
	Tags []string
}

// NewFile returns a File with the specified data. NewFile ensures that returned
// File f has a non-nil f.Tags slice.
func NewFile(name, disk, path string, size int, tags ...string) File {
	if tags == nil {
		tags = make([]string, 0)
	}
	return File{
		Name: name,
		Disk: disk,
		Path: path,
		Size: size,
		Tags: tags,
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

// StorageOption is an option for creating a Storage.
type StorageOption func(*storage)

type storage struct {
	mux   sync.RWMutex
	disks map[string]StorageDisk
}

// ConfigureDisk returns a StorageOption that configures a StorageDisk under the
// provided name.
func ConfigureDisk(name string, disk StorageDisk) StorageOption {
	return func(s *storage) {
		s.disks[name] = disk
	}
}

// NewStorage returns a Storage, configured by opts.
func NewStorage(opts ...StorageOption) Storage {
	s := storage{disks: make(map[string]StorageDisk)}
	for _, opt := range opts {
		opt(&s)
	}
	return &s
}

func (s *storage) Disk(name string) (StorageDisk, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	if disk, ok := s.disks[name]; ok {
		return disk, nil
	}

	return nil, ErrUnconfiguredDisk
}

type godriveStorage struct {
	manager *godrive.Manager
}

// GoDriveStorage returns a Storage that uses `godrive` as the storage engine.
func GoDriveStorage(manager *godrive.Manager) Storage {
	return &godriveStorage{manager: manager}
}

func (s *godriveStorage) Disk(name string) (StorageDisk, error) {
	disk, err := s.manager.Disk(name)
	if err != nil {
		var unconfiguredError godrive.UnconfiguredDiskError
		if errors.As(err, &unconfiguredError) {
			return nil, ErrUnconfiguredDisk
		}
		return nil, fmt.Errorf("godrive: %w", err)
	}
	return disk, nil
}

type memoryDisk struct {
	mux   sync.RWMutex
	files map[string][]byte
}

// MemoryDisk returns an in-memory StorageDisk.
func MemoryDisk() StorageDisk {
	return &memoryDisk{
		files: make(map[string][]byte),
	}
}

func (d *memoryDisk) Put(_ context.Context, path string, b []byte) error {
	d.mux.Lock()
	defer d.mux.Unlock()
	d.files[path] = b
	return nil
}

func (d *memoryDisk) Get(_ context.Context, path string) ([]byte, error) {
	d.mux.RLock()
	defer d.mux.RUnlock()
	if b, ok := d.files[path]; ok {
		out := make([]byte, len(b))
		copy(out, b)
		return out, nil
	}
	return nil, ErrFileNotFound
}

func (d *memoryDisk) Delete(_ context.Context, path string) error {
	d.mux.Lock()
	defer d.mux.Unlock()
	delete(d.files, path)
	return nil
}
