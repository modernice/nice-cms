package media_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/modernice/cms/media"
	"github.com/modernice/cms/media/mock_media"
)

var (
	exampleImagePath     = "/example/example.png"
	exampleImageName     = "example.png"
	exampleImageDiskName = "test"
)

func TestImageService_Upload(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	disk := mock_media.NewMockStorageDisk(ctrl)
	repo := mock_media.NewMockImageRepository(ctrl)
	storage := media.NewStorage(media.ConfigureDisk(exampleImageDiskName, disk))

	svc := media.NewImageService(repo, storage)

	width := 200
	height := 100
	color := color.RGBA{200, 20, 80, 0xff}
	img := coloredRectangleBuffer(width, height, color)
	b := img.Bytes()

	repo.EXPECT().Save(gomock.Any(), media.Image{
		File:   media.NewFile(exampleImageName, exampleImageDiskName, exampleImagePath, len(b)),
		Width:  width,
		Height: height,
	}).Return(nil)

	disk.EXPECT().
		Put(gomock.Any(), exampleImagePath, b).
		Return(nil)

	simg, err := svc.Upload(context.Background(), img, exampleImageName, exampleImageDiskName, exampleImagePath)
	if err != nil {
		t.Fatalf("image upload failed: %v", err)
	}

	if simg.Disk != exampleImageDiskName {
		t.Fatalf("disk of uploaded image should be %q; is %q", exampleImageDiskName, simg.Disk)
	}

	if simg.Path != exampleImagePath {
		t.Fatalf("path of uploaded image should be %q; is %q", exampleImagePath, simg.Path)
	}

	if simg.Name != exampleImageName {
		t.Fatalf("name of uploaded image should be %q; is %q", exampleImageName, simg.Name)
	}

	if simg.Size != len(b) {
		t.Fatalf("size of uploaded image should be %d bytes; is %d bytes", len(b), simg.Size)
	}

	if simg.Width != width {
		t.Fatalf("width of uploaded image should be %d; is %d", width, simg.Width)
	}

	if simg.Height != height {
		t.Fatalf("height of uploaded image should be %d; is %d", height, simg.Height)
	}
}

func TestImageService_Upload_rollbackOnRepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock_media.NewMockImageRepository(ctrl)
	disk := mock_media.NewMockStorageDisk(ctrl)
	storage := media.NewStorage(media.ConfigureDisk(exampleImageDiskName, disk))
	svc := media.NewImageService(repo, storage)

	disk.EXPECT().Put(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	mockSaveError := errors.New("mock save error")
	repo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(mockSaveError)

	disk.EXPECT().Delete(gomock.Any(), exampleImagePath).Return(nil)

	img := coloredRectangleBuffer(400, 200, color.RGBA{200, 20, 80, 0xff})

	_, err := svc.Upload(context.Background(), img, exampleImageName, exampleImageDiskName, exampleImagePath)
	if !errors.Is(err, media.ErrUploadFailed) {
		t.Fatalf("Upload should return %q when the ImageRepository fails to save the Image; got %q", media.ErrUploadFailed, err)
	}
}

func TestImageService_Download_errFileNotFound(t *testing.T) {
	repo := media.MemoryImageRepository()
	disk := media.MemoryDisk()
	storage := media.NewStorage(media.ConfigureDisk(exampleImageDiskName, disk))
	svc := media.NewImageService(repo, storage)

	img, err := svc.Download(context.Background(), exampleImageDiskName, exampleImagePath)
	if !errors.Is(err, media.ErrFileNotFound) {
		t.Fatalf("Download should return %q for a non-existing image; got %q", media.ErrFileNotFound, err)
	}

	if img != nil {
		t.Fatalf("Download should return a nil image if the image does not exist; got %v", img)
	}
}

func TestImageService_Download(t *testing.T) {
	repo := media.MemoryImageRepository()
	disk := media.MemoryDisk()
	storage := media.NewStorage(media.ConfigureDisk(exampleImageDiskName, disk))
	svc := media.NewImageService(repo, storage)

	img := coloredRectangleBuffer(400, 200, color.RGBA{200, 20, 80, 0xff})
	b := img.Bytes()

	if _, err := svc.Upload(context.Background(), img, exampleImageName, exampleImageDiskName, exampleImagePath); err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	downloadedImage, err := svc.Download(context.Background(), exampleImageDiskName, exampleImagePath)
	if err != nil {
		t.Fatalf("Download shouldn't fail for an existing file; failed with %q", err)
	}

	originalImage, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("failed to decode original image: %v", err)
	}

	assertEqualImages(t, originalImage, downloadedImage)
}

func TestImageService_Replace(t *testing.T) {
	repo := media.MemoryImageRepository()
	disk := media.MemoryDisk()
	storage := media.NewStorage(media.ConfigureDisk(exampleImageDiskName, disk))
	svc := media.NewImageService(repo, storage)

	img := coloredRectangleBuffer(400, 200, color.RGBA{200, 20, 80, 0xff})

	uploadedImage, err := svc.Upload(context.Background(), img, exampleImageName, exampleImageDiskName, exampleImagePath)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	img = coloredRectangleBuffer(600, 300, color.RGBA{100, 150, 200, 0xff})
	b := img.Bytes()

	replacedImage, err := svc.Replace(context.Background(), img, uploadedImage.Disk, uploadedImage.Path)
	if err != nil {
		t.Fatalf("Replace should not fail; failed with %q", err)
	}

	if replacedImage.Size != len(b) {
		t.Fatalf("image size should now be %d bytes; is %d", len(b), replacedImage.Size)
	}

	if replacedImage.Width != 600 {
		t.Fatalf("image width should now be %d; is %d", 600, replacedImage.Width)
	}

	if replacedImage.Height != 300 {
		t.Fatalf("image height should now be %d; is %d", 300, replacedImage.Height)
	}
}

func TestImageService_Rename(t *testing.T) {
	repo := media.MemoryImageRepository()
	disk := media.MemoryDisk()
	storage := media.NewStorage(media.ConfigureDisk(exampleImageDiskName, disk))
	svc := media.NewImageService(repo, storage)

	img := coloredRectangleBuffer(400, 200, color.RGBA{100, 100, 100, 0xff})

	uploadedImage, err := svc.Upload(context.Background(), img, exampleImageName, exampleImageDiskName, exampleImagePath)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	newName := "renamed.png"
	renamedImage, err := svc.Rename(context.Background(), newName, uploadedImage.Disk, uploadedImage.Path)
	if err != nil {
		t.Fatalf("Rename shouldn't fail for an existing image; failed with %q", err)
	}

	wantImage := uploadedImage
	wantImage.Name = newName

	if !reflect.DeepEqual(renamedImage, wantImage) {
		t.Fatalf("Rename should return %v; got %v", wantImage, renamedImage)
	}

	repoImage, err := repo.Get(context.Background(), renamedImage.Disk, renamedImage.Path)
	if err != nil {
		t.Fatalf("ImageRepository should contain the image; failed with %q", err)
	}

	if !reflect.DeepEqual(repoImage, wantImage) {
		t.Fatalf("ImageRepository returned wrong image. want=%v got=%v", wantImage, repoImage)
	}
}

func TestImageService_Delete_errUnknownImage(t *testing.T) {
	repo := media.MemoryImageRepository()
	disk := media.MemoryDisk()
	storage := media.NewStorage(media.ConfigureDisk(exampleImageDiskName, disk))
	svc := media.NewImageService(repo, storage)

	if err := svc.Delete(context.Background(), exampleImageDiskName, exampleImagePath); !errors.Is(err, media.ErrUnknownImage) {
		t.Fatalf("deleting an unknown image should fail with %q; got %q", media.ErrUnknownImage, err)
	}
}

func TestImageService_Delete(t *testing.T) {
	repo := media.MemoryImageRepository()
	disk := media.MemoryDisk()
	storage := media.NewStorage(media.ConfigureDisk(exampleImageDiskName, disk))
	svc := media.NewImageService(repo, storage)

	img := coloredRectangleBuffer(400, 200, color.RGBA{100, 100, 100, 0xff})

	uploadedImage, err := svc.Upload(context.Background(), img, exampleImageName, exampleImageDiskName, exampleImagePath)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	if err := svc.Delete(context.Background(), uploadedImage.Disk, uploadedImage.Path); err != nil {
		t.Fatalf("deleting an existing image shouldn't fail; failed with %q", err)
	}

	if _, err := repo.Get(context.Background(), uploadedImage.Disk, uploadedImage.Path); !errors.Is(err, media.ErrUnknownImage) {
		t.Fatalf("ImageRepository.Get should return %q for a deleted image; got %q", media.ErrUnknownImage, err)
	}

	d, err := storage.Disk(uploadedImage.Disk)
	if err != nil {
		t.Fatalf("get storage disk %q: %v", uploadedImage.Disk, err)
	}

	if _, err := d.Get(context.Background(), uploadedImage.Path); !errors.Is(err, media.ErrFileNotFound) {
		t.Fatalf("StorageDisk.Get should return %q for a deleted image; got %q", media.ErrFileNotFound, err)
	}
}

func TestImageService_Tag_errUnknownImage(t *testing.T) {
	repo := media.MemoryImageRepository()
	disk := media.MemoryDisk()
	storage := media.NewStorage(media.ConfigureDisk(exampleImageDiskName, disk))
	svc := media.NewImageService(repo, storage)

	if _, err := svc.Tag(context.Background(), exampleImageDiskName, exampleImagePath, "foo", "bar", "baz"); !errors.Is(err, media.ErrUnknownImage) {
		t.Fatalf("tagging a non-existent image should fail with %q; got %q", media.ErrUnknownImage, err)
	}
}

func TestImageService_Tag(t *testing.T) {
	repo := media.MemoryImageRepository()
	disk := media.MemoryDisk()
	storage := media.NewStorage(media.ConfigureDisk(exampleImageDiskName, disk))
	svc := media.NewImageService(repo, storage)

	img := coloredRectangleBuffer(400, 200, color.RGBA{100, 100, 100, 0xff})

	uploadedImage, err := svc.Upload(context.Background(), img, exampleImageName, exampleImageDiskName, exampleImagePath)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	tags := []string{"foo", "bar", "baz"}

	taggedImage, err := svc.Tag(context.Background(), uploadedImage.Disk, uploadedImage.Path, tags...)
	if err != nil {
		t.Fatalf("tagging shouldn't fail on an existing image; failed with %q", err)
	}

	if !taggedImage.HasTag(tags...) {
		t.Fatalf("tagged image should have %v tags; has %v", tags, taggedImage.Tags)
	}

	repoTaggedImage, err := repo.Get(context.Background(), taggedImage.Disk, taggedImage.Path)
	if err != nil {
		t.Fatalf("failed to get image: %v", err)
	}

	if !reflect.DeepEqual(taggedImage, repoTaggedImage) {
		t.Fatalf("image repository should return %#v; got %#v", taggedImage, repoTaggedImage)
	}
}

func TestImageService_Untag(t *testing.T) {
	repo := media.MemoryImageRepository()
	disk := media.MemoryDisk()
	storage := media.NewStorage(media.ConfigureDisk(exampleImageDiskName, disk))
	svc := media.NewImageService(repo, storage)

	img := coloredRectangleBuffer(400, 200, color.RGBA{100, 100, 100, 0xff})

	uploadedImage, err := svc.Upload(context.Background(), img, exampleImageName, exampleImageDiskName, exampleImagePath)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	tags := []string{"foo", "bar", "baz"}

	taggedImage, err := svc.Tag(context.Background(), uploadedImage.Disk, uploadedImage.Path, tags...)
	if err != nil {
		t.Fatalf("tagging failed: %v", err)
	}

	untaggedImage, err := svc.Untag(context.Background(), taggedImage.Disk, taggedImage.Path, "bar")
	if err != nil {
		t.Fatalf("untagging shouldn't fail on an existing image; failed with %q", err)
	}

	if !untaggedImage.HasTag("foo", "baz") || untaggedImage.HasTag("bar") {
		t.Fatalf("untagged image should have %v tags; has %v", []string{"foo", "baz"}, untaggedImage.Tags)
	}

	repoUntaggedImage, err := repo.Get(context.Background(), untaggedImage.Disk, untaggedImage.Path)
	if err != nil {
		t.Fatalf("failed to get image: %v", err)
	}

	if !reflect.DeepEqual(untaggedImage, repoUntaggedImage) {
		t.Fatalf("image repository should return %#v; got %#v", untaggedImage, repoUntaggedImage)
	}
}

func coloredRectangle(width, height int, color color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, color)
		}
	}
	return img
}

func coloredRectangleBuffer(width, height int, color color.Color) *bytes.Buffer {
	img := coloredRectangle(width, height, color)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(fmt.Errorf("encode png: %w", err))
	}
	return &buf
}

func assertEqualImages(t *testing.T, images ...image.Image) {
	if len(images) < 2 {
		return
	}

	for i, img := range images[1:] {
		prev := images[i]

		if prev.Bounds() != img.Bounds() {
			t.Fatalf("bounds of image#%d (%v) differs from bounds of image #%d (%v)", i+1, i, img.Bounds(), prev.Bounds())
		}

		for x := 0; x < prev.Bounds().Dx(); x++ {
			for y := 0; y < prev.Bounds().Dy(); y++ {
				if prev.At(x, y) != img.At(x, y) {
					t.Fatalf("image#%d differs from image#%d at point [%d,%d]", i+1, i, x, y)
				}
			}
		}
	}
}
