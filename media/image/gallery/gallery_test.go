package gallery_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/color"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/modernice/goes/test"
	"github.com/modernice/nice-cms/internal/imggen"
	"github.com/modernice/nice-cms/internal/slice"
	"github.com/modernice/nice-cms/media"
	"github.com/modernice/nice-cms/media/image/gallery"
	"github.com/modernice/nice-cms/media/mock_media"
)

var (
	exampleName = "Example"
	exampleDisk = "foo-disk"
	examplePath = "/example/example.png"
)

func TestNew(t *testing.T) {
	id := uuid.New()
	g := gallery.New(id)

	if g.AggregateID() != id {
		t.Fatalf("AggregateID should return %v; got %v", id, g.AggregateID())
	}

	if g.AggregateName() != "cms.media.image.gallery" {
		t.Fatalf("AggregateName should return %q; got %q", "cms.media.image.gallery", g.AggregateName())
	}
}

func TestGallery_Create(t *testing.T) {
	g := gallery.New(uuid.New())

	name := "foo"
	if err := g.Create(name); err != nil {
		t.Fatalf("Create shouldn't fail with a %q name; failed with %q", name, err)
	}

	if g.Implementation.Name != name {
		t.Fatalf("Name should be %q; is %q", name, g.Implementation.Name)
	}

	test.Change(t, g, gallery.Created, test.EventData(gallery.CreatedData{Name: name}))
}

func TestGallery_Create_emptyName(t *testing.T) {
	g := gallery.New(uuid.New())

	name := "   "
	if err := g.Create(name); !errors.Is(err, gallery.ErrEmptyName) {
		t.Fatalf("Create with an empty name should fail with %q; got %q", gallery.ErrEmptyName, err)
	}

	if g.Implementation.Name != "" {
		t.Fatalf("Name should be %q; is %q", "", g.Implementation.Name)
	}

	test.NoChange[any](t, g, gallery.Created)
}

func TestCreate_alreadyCreated(t *testing.T) {
	g := gallery.New(uuid.New())
	g.Create("foo")
	if err := g.Create("foo"); !errors.Is(err, gallery.ErrAlreadyCreated) {
		t.Fatalf("Create should fail with %q if the Gallery has been created already; got %q", gallery.ErrAlreadyCreated, err)
	}
}

func TestGallery_Upload_notCreated(t *testing.T) {
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))

	g := gallery.New(uuid.New())

	_, buf := imggen.ColoredRectangle(800, 600, color.RGBA{100, 100, 100, 0xff})

	if _, err := g.Upload(context.Background(), storage, buf, exampleName, exampleDisk, examplePath); !errors.Is(err, gallery.ErrNotCreated) {
		t.Fatalf("Upload should fail with %q if the Gallery hasn't been created yet; got %q", gallery.ErrNotCreated, err)
	}

	test.NoChange[any](t, g, gallery.ImageUploaded)
}

func TestGallery_Upload(t *testing.T) {
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))

	g := gallery.New(uuid.New())
	g.Create("foo")

	_, buf := imggen.ColoredRectangle(400, 200, color.RGBA{100, 100, 100, 0xff})
	b := buf.Bytes()

	stack, err := g.Upload(
		context.Background(),
		storage,
		buf,
		exampleName,
		exampleDisk,
		examplePath,
	)
	if err != nil {
		t.Fatalf("upload shouldn't fail; failed with %q", err)
	}

	if stack.ID == uuid.Nil {
		t.Fatalf("Stack should have non-nil UUID")
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

	galleryStack, err := g.Stack(stack.ID)
	if err != nil {
		t.Fatalf("Gallery should contain Stack %q; failed with %q", stack.ID, err)
	}

	if !reflect.DeepEqual(galleryStack, stack) {
		t.Fatalf("Gallery returned wrong Stack. want=%v got=%v", stack, galleryStack)
	}

	expectStorageFileContents(t, storage, galleryStack.Images[0].Disk, galleryStack.Images[0].Path, b)

	test.Change(t, g, gallery.ImageUploaded, test.EventData(gallery.ImageUploadedData{Stack: stack}))
}

func TestGallery_Stack(t *testing.T) {
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))

	g := gallery.New(uuid.New())
	g.Create("foo")

	_, buf := imggen.ColoredRectangle(800, 600, color.RGBA{100, 100, 100, 0xff})

	uploaded, err := g.Upload(
		context.Background(),
		storage,
		buf,
		exampleName,
		exampleDisk,
		examplePath,
	)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	stack, err := g.Stack(uploaded.ID)
	if err != nil {
		t.Fatalf("Get should return the uploaded Stack; failed with %q", err)
	}

	if !reflect.DeepEqual(stack, uploaded) {
		t.Fatalf("Get should return %v; got %v", uploaded, stack)
	}
}

func TestGallery_Delete_notCreated(t *testing.T) {
	storage := media.NewStorage()
	g := gallery.New(uuid.New())

	if err := g.Delete(context.Background(), storage, gallery.Stack{}); !errors.Is(err, gallery.ErrNotCreated) {
		t.Fatalf("Delete should fail with %q if the Gallery hasn't been created yet; got %q", gallery.ErrNotCreated, err)
	}

	test.NoChange[any](t, g, gallery.StackDeleted)
}

func TestGallery_Delete(t *testing.T) {
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))

	g := gallery.New(uuid.New())
	g.Create("foo")

	_, buf := imggen.ColoredRectangle(800, 600, color.RGBA{100, 100, 100, 0xff})

	uploaded, err := g.Upload(
		context.Background(),
		storage,
		buf,
		exampleName,
		exampleDisk,
		examplePath,
	)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	if err := g.Delete(context.Background(), storage, uploaded); err != nil {
		t.Fatalf("deleting an existing Stack shouldn't fail; failed with %q", err)
	}

	if _, err := g.Stack(uploaded.ID); !errors.Is(err, gallery.ErrStackNotFound) {
		t.Fatalf("Stack should return %q for a deleted Stack; got %q", gallery.ErrStackNotFound, err)
	}

	for _, img := range uploaded.Images {
		expectNoStorageFile(t, storage, img.Disk, img.Path)
	}

	test.Change(t, g, gallery.StackDeleted, test.EventData(gallery.StackDeletedData{Stack: uploaded}))
}

func TestGallery_Delete_failingStorage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))
	mockDisk := mock_media.NewMockStorageDisk(ctrl)
	mockStorage := media.NewStorage(media.ConfigureDisk(exampleDisk, mockDisk))

	g := gallery.New(uuid.New())
	g.Create("foo")

	_, buf := imggen.ColoredRectangle(800, 600, color.RGBA{100, 100, 100, 0xff})

	uploaded, err := g.Upload(
		context.Background(),
		storage,
		buf,
		exampleName,
		exampleDisk,
		examplePath,
	)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	mockError := errors.New("mock error")
	mockDisk.EXPECT().Delete(gomock.Any(), examplePath).Return(mockError)

	if err := g.Delete(context.Background(), mockStorage, uploaded); err != nil {
		t.Fatalf("Delete should continue executing when the StorageDisk fails to delete images; got %q", err)
	}

	if _, err := g.Stack(uploaded.ID); !errors.Is(err, gallery.ErrStackNotFound) {
		t.Fatalf("Get should return %q for a deleted Stack; got %q", gallery.ErrStackNotFound, err)
	}

	test.Change(t, g, gallery.StackDeleted, test.EventData(gallery.StackDeletedData{Stack: uploaded}))
}

func TestGallery_Tag_Untag_notCreated(t *testing.T) {
	g := gallery.New(uuid.New())
	var stack gallery.Stack

	if _, err := g.Tag(context.Background(), stack, "foo", "bar", "baz"); !errors.Is(err, gallery.ErrNotCreated) {
		t.Fatalf("Tag should fail with %q if the Gallery hasn't been created yet; got %q", gallery.ErrNotCreated, err)
	}

	if _, err := g.Untag(context.Background(), stack, "foo", "bar", "baz"); !errors.Is(err, gallery.ErrNotCreated) {
		t.Fatalf("Untag should fail with %q if the Gallery hasn't been created yet; got %q", gallery.ErrNotCreated, err)
	}
}

func TestGallery_Replace(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))

	g := gallery.New(uuid.New())
	g.Create("foo")

	_, buf := imggen.ColoredRectangle(800, 600, color.RGBA{100, 100, 100, 0xff})

	uploaded, err := g.Upload(
		ctx,
		storage,
		buf,
		exampleName,
		exampleDisk,
		examplePath,
	)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	_, buf = imggen.ColoredRectangle(400, 500, color.RGBA{130, 100, 100, 0xff})
	b := buf.Bytes()

	replaced, err := g.Replace(ctx, storage, buf, uploaded.ID)
	if err != nil {
		t.Fatalf("Replace failed with %q", err)
	}

	if replaced.Original().Filesize != len(b) {
		t.Fatalf("Filesize should be %d; is %d", len(b), replaced.Original().Filesize)
	}

	disk, _ := storage.Disk(uploaded.Original().Disk)
	file, err := disk.Get(ctx, uploaded.Original().Path)
	if err != nil {
		t.Fatalf("get storage file: %v", err)
	}

	if !bytes.Equal(file, b) {
		t.Fatalf("storage file should have been replaced")
	}

	test.Change(t, g, gallery.ImageReplaced, test.EventData(gallery.ImageReplacedData{
		Stack: replaced,
	}))
}

func TestGallery_Tag_Untag(t *testing.T) {
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))

	g := gallery.New(uuid.New())
	g.Create("foo")

	_, buf := imggen.ColoredRectangle(800, 600, color.RGBA{100, 100, 100, 0xff})

	uploaded, err := g.Upload(
		context.Background(),
		storage,
		buf,
		exampleName,
		exampleDisk,
		examplePath,
	)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
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

	test.Change(t, g, gallery.StackTagged, test.EventData(gallery.StackTaggedData{
		StackID: tagged.ID,
		Tags:    []string{"foo", "bar", "baz"},
	}))

	test.Change(t, g, gallery.StackUntagged, test.EventData(gallery.StackUntaggedData{
		StackID: untagged.ID,
		Tags:    []string{"bar", "foo"},
	}))
}

func TestGallery_RenameStack_notCreated(t *testing.T) {
	g := gallery.New(uuid.New())

	if _, err := g.RenameStack(context.Background(), uuid.New(), "new name"); !errors.Is(err, gallery.ErrNotCreated) {
		t.Fatalf("Rename should fail with %q if the Gallery hasn't been created yet; got %q", gallery.ErrNotCreated, err)
	}
}

func TestGallery_RenameStack(t *testing.T) {
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))
	g := gallery.New(uuid.New())
	g.Create("foo")

	_, buf := imggen.ColoredRectangle(800, 600, color.RGBA{100, 100, 100, 0xff})

	uploaded, err := g.Upload(
		context.Background(),
		storage,
		buf,
		exampleName,
		exampleDisk,
		examplePath,
	)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	renamed, err := g.RenameStack(context.Background(), uploaded.ID, "New name")
	if err != nil {
		t.Fatalf("Rename failed with %q", err)
	}

	stack, err := g.Stack(renamed.ID)
	if err != nil {
		t.Fatalf("Gallery should contain Stack %q; failed with %q", renamed.ID, err)
	}

	if !reflect.DeepEqual(stack, renamed) {
		t.Fatalf("Stack returned wrong Stack. want=%v got=%v", renamed, stack)
	}

	test.Change(t, g, gallery.StackRenamed, test.EventData(gallery.StackRenamedData{
		StackID: renamed.ID,
		OldName: exampleName,
		Name:    "New name",
	}))
}

func TestGallery_Update(t *testing.T) {
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))

	g := gallery.New(uuid.New())
	g.Create("foo")

	_, buf := imggen.ColoredRectangle(800, 600, color.RGBA{100, 100, 100, 0xff})

	stack, err := g.Upload(context.Background(), storage, buf, exampleName, exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	replacement := stack.WithTag("foo")

	if err := g.Update(stack.ID, func(gallery.Stack) gallery.Stack {
		return replacement
	}); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	got, err := g.Stack(stack.ID)
	if err != nil {
		t.Fatalf("Stack(%q) failed with %q", stack.ID, err)
	}

	if !reflect.DeepEqual(got, replacement) {
		t.Fatalf("Stack returned wrong Stack. want=%v got=%v", replacement, got)
	}

	test.Change(t, g, gallery.StackUpdated, test.EventData(gallery.StackUpdatedData{Stack: replacement}))
}

func TestGallery_Sort(t *testing.T) {

	stacks := gallery.Stacks{
		{ID: uuid.New()},
		{ID: uuid.New()},
		{ID: uuid.New()},
		{ID: uuid.New()},
	}

	ids := slice.Map(stacks, func(s gallery.Stack) uuid.UUID { return s.ID }).([]uuid.UUID)

	tests := []struct {
		sort []uuid.UUID
		want gallery.Stacks
	}{
		{
			sort: nil,
			want: stacks,
		},
		{
			sort: ids,
			want: stacks,
		},
		{
			sort: ids[2:4],
			want: gallery.Stacks{
				stacks[2],
				stacks[3],
				stacks[0],
				stacks[1],
			},
		},
		{
			sort: ids[1:3],
			want: gallery.Stacks{
				stacks[1],
				stacks[2],
				stacks[0],
				stacks[3],
			},
		},
		{
			sort: []uuid.UUID{ids[1], ids[3]},
			want: gallery.Stacks{
				stacks[1],
				stacks[3],
				stacks[0],
				stacks[2],
			},
		},
		{
			sort: []uuid.UUID{ids[3], ids[1]},
			want: gallery.Stacks{
				stacks[3],
				stacks[1],
				stacks[0],
				stacks[2],
			},
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			g := gallery.New(uuid.New())
			g.Create("foo")

			g.Stacks = make(gallery.Stacks, len(stacks))
			copy(g.Stacks, stacks)

			sorting := g.Sort(tt.sort)

			if !cmp.Equal(tt.want, g.Stacks) {
				t.Fatalf("Stacks have wrong order.\n\n%s", cmp.Diff(tt.want, g.Stacks))
			}

			if len(sorting) == 0 {
				test.NoChange[any](t, g, gallery.Sorted)
				return
			}

			test.Change(t, g, gallery.Sorted, test.EventData(gallery.SortedData{Sorting: sorting}))
		})
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

func expectNoStorageFile(t *testing.T, storage media.Storage, diskName, path string) {
	disk, err := storage.Disk(diskName)
	if err != nil {
		t.Fatalf("get %q storage disk: %v", diskName, err)
	}

	if _, err := disk.Get(context.Background(), path); !errors.Is(err, media.ErrFileNotFound) {
		t.Fatalf("storage should return %q for a non-existing file; got %q", media.ErrFileNotFound, err)
	}
}
