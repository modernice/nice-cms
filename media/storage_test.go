package media_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/modernice/nice-cms/media"
	"github.com/modernice/nice-cms/media/mock_media"
)

func TestFile_WithTag(t *testing.T) {
	var f media.File
	f = f.WithTag("foo", "bar")
	f = f.WithTag("bar", "baz")

	want := []string{"foo", "bar", "baz"}
	if !reflect.DeepEqual(f.Tags, want) {
		t.Fatalf("File should have %v tags; has %v", want, f.Tags)
	}
}

func TestFile_WithoutTag(t *testing.T) {
	var f media.File
	f = f.WithTag("foo", "bar")
	f = f.WithoutTag("bar")
	f = f.WithTag("foobar", "bazbar")
	f = f.WithTag("bazbar", "foobarbaz")
	f = f.WithoutTag("foobar")

	want := []string{"foo", "bazbar", "foobarbaz"}
	if !reflect.DeepEqual(f.Tags, want) {
		t.Fatalf("File should have %v tags; has %v", want, f.Tags)
	}
}

func TestFile_HasTag(t *testing.T) {
	var f media.File

	tags := []string{"foo", "bar", "baz"}

	for _, tag := range tags {
		if f.HasTag(tag) {
			t.Fatalf("File should not have %q tag", tag)
		}
	}

	if f.HasTag(tags...) {
		t.Fatalf("File should not have %v tags", tags)
	}

	f = f.WithTag(tags...)

	for _, tag := range tags {
		if !f.HasTag(tag) {
			t.Fatalf("File should have %q tag", tag)
		}
	}

	if !f.HasTag(tags...) {
		t.Fatalf("File should have %v tags", tags)
	}

	f = f.WithoutTag("baz")

	if f.HasTag(tags...) {
		t.Fatalf("File should not have %v tags", tags)
	}

	if !f.HasTag(tags[:2]...) {
		t.Fatalf("File should have %v tags", tags[:2])
	}
}

func TestStorage_Disk_unconfigured(t *testing.T) {
	storage := media.NewStorage()

	disk, err := storage.Disk("foo")
	if !errors.Is(err, media.ErrUnconfiguredDisk) {
		t.Fatalf("storage.Disk should return false for an unconfigured disk; got %v", err)
	}

	if disk != nil {
		t.Fatalf("storage.Disk should return nil for an unconfigured disk; got %v", disk)
	}
}

func TestStorage_Disk(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	name := "foo"
	disk := mock_media.NewMockStorageDisk(ctrl)
	storage := media.NewStorage(media.ConfigureDisk(name, disk))

	disk2, err := storage.Disk(name)
	if err != nil {
		t.Fatalf("storage.Disk should return no error for a configured disk; got %v", err)
	}

	if disk2 != disk {
		t.Fatalf("storage.Disk should return the configured disk %v; got %v", disk, disk2)
	}
}
