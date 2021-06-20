package gallery_test

import (
	"bytes"
	"context"
	"errors"
	"image/color"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/modernice/cms/internal/imggen"
	"github.com/modernice/cms/media"
	"github.com/modernice/cms/media/image"
	"github.com/modernice/cms/media/image/gallery"
	"github.com/modernice/cms/media/mock_media"
)

var (
	exampleName = "Example"
	exampleDisk = "foo-disk"
	examplePath = "/example/example.png"
)

func TestGallery_Upload_withoutProcessing(t *testing.T) {
	stacks := gallery.InMemoryStackRepository()
	images := image.InMemoryRepository()
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))
	imageService := image.NewService(images, storage)
	enc := image.NewEncoder()

	g := gallery.New("foo", stacks, images, imageService, enc)

	_, buf := imggen.ColoredRectangle(400, 200, color.RGBA{100, 100, 100, 0xff})
	b := buf.Bytes()

	stack, processed, err := g.Upload(context.Background(), buf, exampleName, exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("upload shouldn't fail; failed with %q", err)
	}

	if stack.ID == uuid.Nil {
		t.Fatalf("Stack should have non-nil UUID")
	}

	if stack.Gallery != "foo" {
		t.Fatalf("Stack.Gallery should be %q; is %q", "foo", stack.Gallery)
	}

	timer := time.NewTimer(time.Second)
	defer timer.Stop()

	select {
	case <-timer.C:
		t.Fatal("timed out")
	case err := <-processed:
		if err != nil {
			t.Fatalf("processing failed: %v", err)
		}
	}

	if len(stack.Images) != 1 {
		t.Fatalf("Stack should have 1 image; has %d", len(stack.Images))
	}

	img := stack.Images[0]

	if !img.Original {
		t.Fatalf("original image should be marked")
	}

	if img.Size != "" {
		t.Fatalf("size of original image should be empty")
	}

	if img.Disk != exampleDisk {
		t.Fatalf("Image should have %q disk; has %q", exampleDisk, img.Disk)
	}

	if img.Path != examplePath {
		t.Fatalf("Image should have %q path; has %q", examplePath, img.Path)
	}

	if img.Name != exampleName {
		t.Fatalf("Image should have %q name; has %q", exampleName, img.Name)
	}

	if img.Width != 400 {
		t.Fatalf("Image should have width of %d; is %d", 400, img.Width)
	}

	if img.Height != 200 {
		t.Fatalf("Image should have height of %d; is %d", 200, img.Height)
	}

	if img.Filesize != len(b) {
		t.Fatalf("Image should have filesize of %d bytes; has %d bytes", len(b), img.Filesize)
	}

	repoStack, err := stacks.Get(context.Background(), stack.ID)
	if err != nil {
		t.Fatalf("StackRepository should contain the created Stack; got %q", err)
	}

	if !reflect.DeepEqual(repoStack, stack) {
		t.Fatalf("StackRepository returned wrong Stack. want=%v got=%v", stack, repoStack)
	}

	repoImage, err := images.Get(context.Background(), img.Disk, img.Path)
	if err != nil {
		t.Fatalf("image.Repository should contain the image of the Stack; got %q", err)
	}

	if !reflect.DeepEqual(repoImage, img.Image) {
		t.Fatalf("image.Repository returned wrong Image. want=%v got=%v", img, repoImage)
	}

	expectStorageFileContents(t, storage, repoImage.Disk, repoImage.Path, b)
}

func TestGallery_Upload_withProcessing(t *testing.T) {
	stacks := gallery.InMemoryStackRepository()
	images := image.InMemoryRepository()
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))
	imageService := image.NewService(images, storage)
	enc := image.NewEncoder()

	pipe := gallery.ProcessingPipeline{
		gallery.Resizer{
			"small":  {Width: 640},
			"medium": {Width: 1280},
			"large":  {Width: 1920},
		},
	}
	g := gallery.New("foo", stacks, images, imageService, enc, gallery.WithProcessing(pipe, 0))

	_, buf := imggen.ColoredRectangle(400, 200, color.RGBA{100, 100, 100, 0xff})

	stack, processed, err := g.Upload(context.Background(), buf, exampleName, exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("upload shouldn't fail; failed with %q", err)
	}

	if len(stack.Images) != 1 {
		t.Fatalf("Stack should have 1 image until it has been processed; has %d", len(stack.Images))
	}

	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()

	select {
	case <-timer.C:
		t.Fatal("timed out")
	case err := <-processed:
		if err != nil {
			t.Fatalf("processing failed: %v", err)
		}
	}

	if err := g.Refresh(context.Background(), &stack); err != nil {
		t.Fatalf("failed to refresh Stack: %v", err)
	}

	if len(stack.Images) != 4 {
		t.Fatalf("Stack should have 4 images; has %d", len(stack.Images))
	}

	for _, img := range stack.Images {
		if _, err := images.Get(context.Background(), img.Disk, img.Path); err != nil {
			t.Fatalf("ImageRepository should contain %q (%s); failed with %q", img.Path, img.Disk, err)
		}

		expectStorageFile(t, storage, img.Disk, img.Path)
	}
}

func TestGallery_Get(t *testing.T) {
	stacks := gallery.InMemoryStackRepository()
	images := image.InMemoryRepository()
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))
	imageService := image.NewService(images, storage)
	enc := image.NewEncoder()

	g := gallery.New("foo", stacks, images, imageService, enc)

	_, buf := imggen.ColoredRectangle(800, 600, color.RGBA{100, 100, 100, 0xff})

	uploaded, _, err := g.Upload(context.Background(), buf, exampleName, exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	stack, err := g.Stack(context.Background(), uploaded.ID)
	if err != nil {
		t.Fatalf("Get should return the uploaded Stack; failed with %q", err)
	}

	if !reflect.DeepEqual(stack, uploaded) {
		t.Fatalf("Get should return %v; got %v", uploaded, stack)
	}
}

func TestGallery_Delete(t *testing.T) {
	stacks := gallery.InMemoryStackRepository()
	images := image.InMemoryRepository()
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))
	imageService := image.NewService(images, storage)
	enc := image.NewEncoder()

	g := gallery.New("foo", stacks, images, imageService, enc)

	_, buf := imggen.ColoredRectangle(800, 600, color.RGBA{100, 100, 100, 0xff})

	uploaded, _, err := g.Upload(context.Background(), buf, exampleName, exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	if err := g.Delete(context.Background(), uploaded); err != nil {
		t.Fatalf("deleting an existing Stack shouldn't fail; failed with %q", err)
	}

	if _, err := g.Stack(context.Background(), uploaded.ID); !errors.Is(err, gallery.ErrStackNotFound) {
		t.Fatalf("Get should return %q for a deleted Stack; got %q", gallery.ErrStackNotFound, err)
	}

	for _, img := range uploaded.Images {
		if _, err := images.Get(context.Background(), img.Disk, img.Path); !errors.Is(err, image.ErrUnknownImage) {
			t.Fatalf("ImageRepository.Get should return %q for a deleted image; got %q", image.ErrUnknownImage, err)
		}

		expectNoStorageFile(t, storage, img.Disk, img.Path)
	}
}

func TestGallery_Delete_failingImageService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	stacks := gallery.InMemoryStackRepository()
	images := image.InMemoryRepository()
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))
	imageService := image.NewService(images, storage)
	mockImageService := mock_media.NewMockImageService(ctrl)
	enc := image.NewEncoder()

	g := gallery.New("foo", stacks, images, imageService, enc)

	_, buf := imggen.ColoredRectangle(800, 600, color.RGBA{100, 100, 100, 0xff})

	uploaded, _, err := g.Upload(context.Background(), buf, exampleName, exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	g = gallery.New("foo", stacks, images, mockImageService, enc)

	mockError := errors.New("mock error")
	mockImageService.EXPECT().Delete(gomock.Any(), exampleDisk, examplePath).Return(mockError)

	if err := g.Delete(context.Background(), uploaded); err != nil {
		t.Fatalf("Delete should continue executing when the ImageService fails to delete images; got %q", err)
	}

	if _, err := g.Stack(context.Background(), uploaded.ID); !errors.Is(err, gallery.ErrStackNotFound) {
		t.Fatalf("Get should return %q for a deleted Stack; got %q", gallery.ErrStackNotFound, err)
	}
}

func TestGallery_Tag_Untag(t *testing.T) {
	stacks := gallery.InMemoryStackRepository()
	images := image.InMemoryRepository()
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))
	imageService := image.NewService(images, storage)
	enc := image.NewEncoder()

	pipe := gallery.ProcessingPipeline{
		gallery.Resizer{
			"small":  {Width: 640},
			"medium": {Width: 1280},
			"large":  {Width: 1920},
		},
	}
	g := gallery.New("foo", stacks, images, imageService, enc, gallery.WithProcessing(pipe, 0))

	_, buf := imggen.ColoredRectangle(800, 600, color.RGBA{100, 100, 100, 0xff})

	uploaded, processed, err := g.Upload(context.Background(), buf, exampleName, exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()

	select {
	case <-timer.C:
		t.Fatal("timed out")
	case err := <-processed:
		if err != nil {
			t.Fatalf("processing failed: %v", err)
		}
	}

	tagged, err := g.Tag(context.Background(), uploaded, "foo", "bar", "baz", "baz")
	if err != nil {
		t.Fatalf("Tag failed with %q", err)
	}

	for _, img := range tagged.Images {
		if len(img.Tags) != 3 {
			t.Fatalf("Image should have 3 tags; has %d", len(img.Tags))
		}

		if !img.HasTag("foo", "bar", "baz") {
			t.Fatalf("Image should have %v tags; has %v", []string{"foo", "bar", "baz"}, img.Tags)
		}
	}

	untagged, err := g.Untag(context.Background(), tagged, "bar", "foo")
	if err != nil {
		t.Fatalf("Untag failed with %q", err)
	}

	for _, img := range untagged.Images {
		if len(img.Tags) != 1 {
			t.Fatalf("Image should have 1 tag; has %d", len(img.Tags))
		}

		if !img.HasTag("baz") {
			t.Fatalf("Image should have %v tags; has %v", []string{"baz"}, img.Tags)
		}
	}
}

func expectStorageFileContents(t *testing.T, storage media.Storage, diskName, path string, contents []byte) {
	disk, err := storage.Disk(diskName)
	if err != nil {
		t.Fatalf("get %q storage disk: %v", diskName, err)
	}

	storageBytes, err := disk.Get(context.Background(), path)
	if err != nil {
		t.Fatalf("storage should contain file %q (%s); failed with %q", path, diskName, err)
	}

	if !bytes.Equal(storageBytes, contents) {
		t.Fatalf("storage file has wrong contents. want=%v got=%v", contents, storageBytes)
	}
}

func expectStorageFile(t *testing.T, storage media.Storage, diskName, path string) {
	disk, err := storage.Disk(diskName)
	if err != nil {
		t.Fatalf("get %q storage disk: %v", diskName, err)
	}

	if _, err := disk.Get(context.Background(), path); err != nil {
		t.Fatalf("storage should contain file %q (%s); failed with %q", path, diskName, err)
	}
}

func expectNoStorageFile(t *testing.T, storage media.Storage, diskName, path string) {
	disk, err := storage.Disk(diskName)
	if err != nil {
		t.Fatalf("get %q storage disk: %v", diskName, err)
	}

	if _, err := disk.Get(context.Background(), path); !errors.Is(err, media.ErrFileNotFound) {
		t.Fatalf("storage should return %q for a non-existing file; got %q", media.ErrFileNotFound, err)
	}
}
