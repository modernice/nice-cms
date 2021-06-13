package media

//go:generate mockgen -source=image.go -destination=./mock_media/image.go

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"sync"
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

// ImageRepository stores and retrieves Images.
type ImageRepository interface {
	// Save stores an Image in the underlying store.
	Save(context.Context, Image) error

	// Get returns the Image located at the specified disk and path from the
	// underlying store.
	Get(_ context.Context, disk, path string) (Image, error)

	// Delete removes the Image at the specified disk and path from the
	// repository.
	//
	// Delete returns ErrUnknownImage if the specified Image does not exist in
	// the repository.
	Delete(_ context.Context, disk, path string) error
}

// Image is an uploaded image.
type Image struct {
	File

	Width  int
	Height int
}

type imageService struct {
	ImageService

	repo    ImageRepository
	storage Storage
}

func NewImageService(repo ImageRepository, storage Storage) ImageService {
	return &imageService{
		repo:    repo,
		storage: storage,
	}
}

func (svc *imageService) Upload(ctx context.Context, r io.Reader, name, diskName, path string) (Image, error) {
	errorf := func(format string, v ...interface{}) error {
		return fmt.Errorf("upload %q: %w", name, fmt.Errorf(format, v...))
	}

	b, err := io.ReadAll(r)
	if err != nil {
		return Image{}, fmt.Errorf("read image: %w", err)
	}

	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return Image{}, fmt.Errorf("decode image: %w", err)
	}

	disk, err := svc.disk(diskName)
	if err != nil {
		return Image{}, err
	}

	if err := disk.Put(ctx, path, b); err != nil {
		return Image{}, errorf("upload to %q storage: %w", diskName, err)
	}

	out := Image{
		File:   NewFile(name, diskName, path, len(b)),
		Width:  img.Bounds().Dx(),
		Height: img.Bounds().Dy(),
	}

	if err := svc.repo.Save(ctx, out); err != nil {
		if err := svc.rollbackUpload(ctx, out); err != nil {
			return out, fmt.Errorf("rollback upload after ImageRepository failed to save: %w", err)
		}
		return out, ErrUploadFailed
	}

	return out, nil
}

func (svc *imageService) rollbackUpload(ctx context.Context, img Image) error {
	disk, err := svc.disk(img.Disk)
	if err != nil {
		return err
	}

	if err := disk.Delete(ctx, img.Path); err != nil {
		return fmt.Errorf("delete file %q (%s): %w", img.Path, img.Disk, err)
	}

	return nil
}

func (svc *imageService) Download(ctx context.Context, diskName, path string) (image.Image, error) {
	disk, err := svc.disk(diskName)
	if err != nil {
		return nil, err
	}

	b, err := disk.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("get storage file %q (%s): %w", path, diskName, err)
	}

	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	return img, nil
}

func (svc *imageService) Replace(ctx context.Context, r io.Reader, diskName, path string) (Image, error) {
	img, err := svc.repo.Get(ctx, diskName, path)
	if err != nil {
		if errors.Is(err, ErrUnknownImage) {
			return Image{}, err
		}
		return Image{}, fmt.Errorf("fetch image from repository: %w", err)
	}
	return svc.Upload(ctx, r, img.Name, img.Disk, img.Path)
}

func (svc *imageService) Rename(ctx context.Context, name, disk, path string) (Image, error) {
	img, err := svc.repo.Get(ctx, disk, path)
	if err != nil {
		return Image{}, fmt.Errorf("fetch image from repository: %w", err)
	}

	img.Name = name

	if err := svc.repo.Save(ctx, img); err != nil {
		return Image{}, fmt.Errorf("save image: %w", err)
	}

	return img, nil
}

func (svc *imageService) Delete(ctx context.Context, diskName, path string) error {
	if err := svc.repo.Delete(ctx, diskName, path); err != nil {
		return fmt.Errorf("delete from ImageRepository: %w", err)
	}

	disk, err := svc.disk(diskName)
	if err != nil {
		return fmt.Errorf("get storage disk %q: %w", diskName, err)
	}

	if err := disk.Delete(ctx, path); err != nil {
		return fmt.Errorf("delete from storage: %w", err)
	}

	return nil
}

func (svc *imageService) Tag(ctx context.Context, disk, path string, tags ...string) (Image, error) {
	img, err := svc.repo.Get(ctx, disk, path)
	if err != nil {
		return img, fmt.Errorf("fetch image from repository: %w", err)
	}

	if len(tags) == 0 {
		return img, nil
	}

	img.File = img.WithTag(tags...)

	if err := svc.repo.Save(ctx, img); err != nil {
		return img, fmt.Errorf("save image: %w", err)
	}

	return img, nil
}

func (svc *imageService) Untag(ctx context.Context, disk, path string, tags ...string) (Image, error) {
	img, err := svc.repo.Get(ctx, disk, path)
	if err != nil {
		return img, fmt.Errorf("fetch image from repository: %w", err)
	}

	if len(tags) == 0 {
		return img, nil
	}

	img.File = img.WithoutTag(tags...)

	if err := svc.repo.Save(ctx, img); err != nil {
		return img, fmt.Errorf("save image: %w", err)
	}

	return img, nil
}

func (svc *imageService) disk(name string) (StorageDisk, error) {
	disk, err := svc.storage.Disk(name)
	if err != nil {
		return nil, fmt.Errorf("get storage disk %q: %w", name, err)
	}
	return disk, nil
}

type memoryImageRepository struct {
	mux   sync.RWMutex
	disks map[string]map[string]Image // map[DISK]map[PATH]Image
}

// MemoryImageRepository returns an in-memory ImageRepository.
func MemoryImageRepository() ImageRepository {
	return &memoryImageRepository{
		disks: make(map[string]map[string]Image),
	}
}

func (r *memoryImageRepository) Save(_ context.Context, img Image) error {
	r.mux.Lock()
	defer r.mux.Unlock()

	images, ok := r.disks[img.Disk]
	if !ok {
		images = make(map[string]Image)
		r.disks[img.Disk] = images
	}
	images[img.Path] = img

	return nil
}

func (r *memoryImageRepository) Get(_ context.Context, disk, path string) (Image, error) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	images, ok := r.disks[disk]
	if !ok {
		return Image{}, ErrUnknownImage
	}
	img, ok := images[path]
	if !ok {
		return Image{}, ErrUnknownImage
	}
	return img, nil
}

func (r *memoryImageRepository) Delete(_ context.Context, disk, path string) error {
	r.mux.Lock()
	defer r.mux.Unlock()

	if images, ok := r.disks[disk]; ok {
		delete(images, path)
		return nil
	}

	return ErrUnknownImage
}
