package image

//go:generate mockgen -source=service.go -destination=./mock_image/service.go

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"sync"

	"github.com/modernice/cms/media"
)

var (
	// ErrUnknownImage is returned when an image cannot be found in a Repository.
	ErrUnknownImage = errors.New("unknown image")
)

// Repository stores and retrieves Images.
type Repository interface {
	// Save stores an Image in the underlying store.
	Save(context.Context, media.Image) error

	// Get returns the Image located at the specified disk and path or
	// ErrUnknownImage.
	Get(_ context.Context, disk, path string) (media.Image, error)

	// Delete removes the Image at the specified disk and path from the
	// repository.
	//
	// Delete returns ErrUnknownImage if the specified Image does not exist in
	// the repository.
	Delete(_ context.Context, disk, path string) error
}

type imageService struct {
	media.ImageService

	repo    Repository
	storage media.Storage
}

// NewService creates the ImageService that uses the provided Storage to store
// the images and the provided ImageRepository to store the corresponding image
// metadata.
func NewService(repo Repository, storage media.Storage) *imageService {
	return &imageService{
		repo:    repo,
		storage: storage,
	}
}

func (svc *imageService) Upload(ctx context.Context, r io.Reader, name, diskName, path string) (media.Image, error) {
	errorf := func(format string, v ...interface{}) error {
		return fmt.Errorf("upload %q: %w", name, fmt.Errorf(format, v...))
	}

	b, err := io.ReadAll(r)
	if err != nil {
		return media.Image{}, fmt.Errorf("read image: %w", err)
	}

	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return media.Image{}, fmt.Errorf("decode image: %w", err)
	}

	disk, err := svc.disk(diskName)
	if err != nil {
		return media.Image{}, err
	}

	if err := disk.Put(ctx, path, b); err != nil {
		return media.Image{}, errorf("upload to %q storage: %w", diskName, err)
	}

	out := media.Image{
		File:   media.NewFile(name, diskName, path, len(b)),
		Width:  img.Bounds().Dx(),
		Height: img.Bounds().Dy(),
	}

	if err := svc.repo.Save(ctx, out); err != nil {
		if err := svc.rollbackUpload(ctx, out); err != nil {
			return out, fmt.Errorf("rollback upload after ImageRepository failed to save: %w", err)
		}
		return out, media.ErrUploadFailed
	}

	return out, nil
}

func (svc *imageService) rollbackUpload(ctx context.Context, img media.Image) error {
	disk, err := svc.disk(img.Disk)
	if err != nil {
		return err
	}

	if err := disk.Delete(ctx, img.Path); err != nil {
		return fmt.Errorf("delete file %q (%s): %w", img.Path, img.Disk, err)
	}

	return nil
}

func (svc *imageService) Download(ctx context.Context, diskName, path string) (image.Image, string, error) {
	disk, err := svc.disk(diskName)
	if err != nil {
		return nil, "", err
	}

	b, err := disk.Get(ctx, path)
	if err != nil {
		return nil, "", fmt.Errorf("get storage file %q (%s): %w", path, diskName, err)
	}

	img, format, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, "", fmt.Errorf("decode image: %w", err)
	}

	return img, format, nil
}

func (svc *imageService) Replace(ctx context.Context, r io.Reader, diskName, path string) (media.Image, error) {
	img, err := svc.repo.Get(ctx, diskName, path)
	if err != nil {
		if errors.Is(err, ErrUnknownImage) {
			return media.Image{}, err
		}
		return media.Image{}, fmt.Errorf("fetch image from repository: %w", err)
	}
	return svc.Upload(ctx, r, img.Name, img.Disk, img.Path)
}

func (svc *imageService) Rename(ctx context.Context, name, disk, path string) (media.Image, error) {
	img, err := svc.repo.Get(ctx, disk, path)
	if err != nil {
		return media.Image{}, fmt.Errorf("fetch image from repository: %w", err)
	}

	img.Name = name

	if err := svc.repo.Save(ctx, img); err != nil {
		return media.Image{}, fmt.Errorf("save image: %w", err)
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

func (svc *imageService) Tag(ctx context.Context, disk, path string, tags ...string) (media.Image, error) {
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

func (svc *imageService) Untag(ctx context.Context, disk, path string, tags ...string) (media.Image, error) {
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

func (svc *imageService) disk(name string) (media.StorageDisk, error) {
	disk, err := svc.storage.Disk(name)
	if err != nil {
		return nil, fmt.Errorf("get storage disk %q: %w", name, err)
	}
	return disk, nil
}

type memoryImageRepository struct {
	mux   sync.RWMutex
	disks map[string]map[string]media.Image // map[DISK]map[PATH]Image
}

// InMemoryRepository returns an in-memory ImageRepository.
func InMemoryRepository() Repository {
	return &memoryImageRepository{
		disks: make(map[string]map[string]media.Image),
	}
}

func (r *memoryImageRepository) Save(_ context.Context, img media.Image) error {
	r.mux.Lock()
	defer r.mux.Unlock()

	images, ok := r.disks[img.Disk]
	if !ok {
		images = make(map[string]media.Image)
		r.disks[img.Disk] = images
	}
	images[img.Path] = img

	return nil
}

func (r *memoryImageRepository) Get(_ context.Context, disk, path string) (media.Image, error) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	images, ok := r.disks[disk]
	if !ok {
		return media.Image{}, ErrUnknownImage
	}
	img, ok := images[path]
	if !ok {
		return media.Image{}, ErrUnknownImage
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
