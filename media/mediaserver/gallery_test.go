package mediaserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/modernice/cms/internal/imggen"
	"github.com/modernice/cms/media"
	"github.com/modernice/cms/media/image/gallery"
	"github.com/modernice/cms/media/mediaserver"
	"github.com/modernice/goes/aggregate"
	"github.com/modernice/goes/aggregate/repository"
	"github.com/modernice/goes/event"
	"github.com/modernice/goes/event/eventbus/chanbus"
	"github.com/modernice/goes/event/eventstore"
	"github.com/modernice/goes/event/eventstore/memstore"
)

func TestMediaServer_lookupGallery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, galleries, _, _ := newGalleryTest(t, ctx)

	var tests []struct {
		galleryID   uuid.UUID
		galleryName string
	}

	names := []string{"foo", "bar", "baz"}
	for _, name := range names {
		g := gallery.New(uuid.New())
		g.Create(name)

		if err := galleries.Save(ctx, g); err != nil {
			t.Fatalf("save %q Gallery: %v", g.Name, err)
		}

		tests = append(tests, struct {
			galleryID   uuid.UUID
			galleryName string
		}{
			galleryID:   g.ID,
			galleryName: g.Name,
		})
	}

	for _, tt := range tests {
		t.Run(tt.galleryName, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", fmt.Sprintf("/galleries/lookup/name/%s", tt.galleryName), nil)

			srv.ServeHTTP(rec, req)

			var resp struct {
				GalleryID uuid.UUID `json:"galleryId"`
			}

			if rec.Result().StatusCode != http.StatusOK {
				t.Fatalf("Response should have status code %d; has %d", http.StatusOK, rec.Result().StatusCode)
			}

			if err := json.NewDecoder(rec.Result().Body).Decode(&resp); err != nil {
				t.Fatalf("decode response: %v", err)
			}

			if resp.GalleryID == uuid.Nil {
				t.Fatalf("GalleryID should not be nil")
			}

			if _, err := galleries.Fetch(ctx, resp.GalleryID); err != nil {
				t.Fatalf("failed to fetch Gallery %q: %v", resp.GalleryID, err)
			}
		})
	}
}

func TestMediaServer_uploadImage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, galleries, _, _ := newGalleryTest(t, ctx)

	g := gallery.New(uuid.New())
	g.Create("foo")

	if err := galleries.Save(ctx, g); err != nil {
		t.Fatalf("save gallery: %v", err)
	}

	name := "Example image"
	_, img := imggen.ColoredRectangle(800, 600, color.RGBA{100, 100, 100, 0xff})

	var formData bytes.Buffer
	formWriter := multipart.NewWriter(&formData)

	nameField, _ := formWriter.CreateFormField("name")
	nameField.Write([]byte(name))

	diskField, _ := formWriter.CreateFormField("disk")
	diskField.Write([]byte(exampleDisk))

	pathField, _ := formWriter.CreateFormField("path")
	pathField.Write([]byte(examplePath))

	formFile, _ := formWriter.CreateFormFile("image", "image")
	formFile.Write(img.Bytes())

	formWriter.Close()

	req := httptest.NewRequest("POST", fmt.Sprintf("/galleries/%s/stacks", g.ID), &formData)
	req.Header.Set("Content-Type", formWriter.FormDataContentType())

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	res := rec.Result()

	if res.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("StatusCode should be %d; is %d\n\n%s", http.StatusCreated, res.StatusCode, body)
	}

	var resp gallery.Stack
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	g, err := galleries.Fetch(ctx, g.ID)
	if err != nil {
		t.Fatalf("fetch gallery: %v", err)
	}

	stack, err := g.Stack(resp.ID)
	if err != nil {
		t.Fatalf("fetch stack: %v", err)
	}

	if !reflect.DeepEqual(stack, resp) {
		t.Fatalf("invalid response. want=%v got=%v", stack, resp)
	}

	if len(stack.Images) != 1 {
		t.Fatalf("stack should have 1 image; has %d", len(stack.Images))
	}
}

func TestMediaServer_deleteStack(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, galleries, _, helper := newGalleryTest(t, ctx)
	_, _, _, storage, _ := helper()

	g := gallery.New(uuid.New())
	g.Create("foo")

	name := "Example image"
	_, img := imggen.ColoredRectangle(800, 600, color.RGBA{100, 100, 100, 0xff})

	stack, err := g.Upload(ctx, storage, img, name, exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("Upload failed with %q", err)
	}

	if err := galleries.Save(ctx, g); err != nil {
		t.Fatalf("save gallery: %v", err)
	}

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/galleries/%s/stacks/%s", g.ID, stack.ID), nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	res := rec.Result()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("StatusCode should be %d; is %d", http.StatusNoContent, res.StatusCode)
	}

	g, err = galleries.Fetch(ctx, g.ID)
	if err != nil {
		t.Fatalf("fetch gallery: %v", err)
	}

	if _, err := g.Stack(stack.ID); !errors.Is(err, gallery.ErrStackNotFound) {
		t.Fatalf("g.Stack should fail with %q; got %q", gallery.ErrStackNotFound, err)
	}

	disk, _ := storage.Disk(stack.Original().Disk)
	if _, err := disk.Get(ctx, stack.Original().Path); !errors.Is(err, media.ErrFileNotFound) {
		t.Fatalf("storage should return %q for a deleted image; got %q", media.ErrFileNotFound, err)
	}
}

func TestMediaServer_tagStack(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, galleries, _, helper := newGalleryTest(t, ctx)
	_, _, _, storage, _ := helper()

	g := gallery.New(uuid.New())
	g.Create("foo")

	_, img := imggen.ColoredRectangle(800, 600, color.RGBA{100, 100, 100, 0xff})

	stack, err := g.Upload(ctx, storage, img, "Example image", exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("Upload failed with %q", err)
	}

	if err := galleries.Save(ctx, g); err != nil {
		t.Fatalf("save gallery: %v", err)
	}

	body := strings.NewReader(`{"tags": ["foo", "bar", "baz"]}`)

	req := httptest.NewRequest("POST", fmt.Sprintf("/galleries/%s/stacks/%s/tags", g.ID, stack.ID), body)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	res := rec.Result()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode should be %d; is %d", http.StatusOK, res.StatusCode)
	}

	var resp gallery.Stack
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	g, err = galleries.Fetch(ctx, g.ID)
	if err != nil {
		t.Fatalf("fetch gallery: %v", err)
	}

	stack, err = g.Stack(stack.ID)
	if err != nil {
		t.Fatalf("get stack: %v", err)
	}

	if !reflect.DeepEqual(stack, resp) {
		t.Fatalf("invalid response. want=%v got=%v", stack, resp)
	}

	if !stack.Original().HasTag("foo", "bar", "baz") {
		t.Fatalf("stack should have %v tags; has %v", []string{"foo", "bar", "baz"}, stack.Original().Tags)
	}

	req = httptest.NewRequest("DELETE", fmt.Sprintf("/galleries/%s/stacks/%s/tags/foo,baz", g.ID, stack.ID), nil)
	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	res = rec.Result()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode should be %d; is %d", http.StatusOK, res.StatusCode)
	}

	resp = gallery.Stack{}
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	g, err = galleries.Fetch(ctx, g.ID)
	if err != nil {
		t.Fatalf("fetch gallery: %v", err)
	}

	stack, err = g.Stack(stack.ID)
	if err != nil {
		t.Fatalf("get stack: %v", err)
	}

	if !reflect.DeepEqual(stack, resp) {
		t.Fatalf("invalid response. want=%v got=%v", stack, resp)
	}

	if stack.Original().HasTag("foo") || stack.Original().HasTag("baz") {
		t.Fatalf("stack should not have %v tags; has %v", []string{"foo", "baz"}, stack.Original().Tags)
	}

	if !stack.Original().HasTag("bar") {
		t.Fatalf("stack should have %q tag", "bar")
	}
}

func TestMediaServer_replaceImage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, galleries, _, helper := newGalleryTest(t, ctx)
	_, _, _, storage, _ := helper()

	g := gallery.New(uuid.New())
	g.Create("foo")

	_, img := imggen.ColoredRectangle(800, 600, color.RGBA{100, 100, 100, 0xff})

	stack, err := g.Upload(ctx, storage, img, "Example image", exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("Upload failed with %q", err)
	}

	if err := galleries.Save(ctx, g); err != nil {
		t.Fatalf("save gallery: %v", err)
	}

	_, img = imggen.ColoredRectangle(400, 400, color.RGBA{100, 100, 100, 0xff})
	b := img.Bytes()

	var formData bytes.Buffer
	formWriter := multipart.NewWriter(&formData)

	fileField, _ := formWriter.CreateFormFile("image", "image")
	fileField.Write(b)

	formWriter.Close()

	req := httptest.NewRequest("PUT", fmt.Sprintf("/galleries/%s/stacks/%s", g.ID, stack.ID), &formData)
	req.Header.Set("Content-Type", formWriter.FormDataContentType())
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	res := rec.Result()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("StatusCode should be %d; is %d\n%s", http.StatusOK, res.StatusCode, body)
	}

	var resp gallery.Stack
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	g, err = galleries.Fetch(ctx, g.ID)
	if err != nil {
		t.Fatalf("fetch gallery: %v", err)
	}

	stack, err = g.Stack(stack.ID)
	if err != nil {
		t.Fatalf("get stack: %v", err)
	}

	if !reflect.DeepEqual(stack, resp) {
		t.Fatalf("invalid response. want=%v got=%v\n\n%s", stack, resp, cmp.Diff(stack, resp))
	}

	disk, _ := storage.Disk(stack.Original().Disk)
	file, err := disk.Get(ctx, stack.Original().Path)
	if err != nil {
		t.Fatalf("disk.Get failed with %q", err)
	}

	if !bytes.Equal(file, b) {
		t.Fatalf("storage file should be replaced!")
	}
}

func TestMediaServer_renameStack(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, galleries, _, helper := newGalleryTest(t, ctx)
	_, _, _, storage, _ := helper()

	g := gallery.New(uuid.New())
	g.Create("foo")

	_, img := imggen.ColoredRectangle(800, 600, color.RGBA{100, 100, 100, 0xff})

	stack, err := g.Upload(ctx, storage, img, "Example image", exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("Upload failed with %q", err)
	}

	if err := galleries.Save(ctx, g); err != nil {
		t.Fatalf("save gallery: %v", err)
	}

	newName := "New image"
	body := strings.NewReader(fmt.Sprintf(`{"name": %q}`, newName))

	req := httptest.NewRequest("PATCH", fmt.Sprintf("/galleries/%s/stacks/%s", g.ID, stack.ID), body)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	res := rec.Result()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode should be %d; is %d", http.StatusOK, res.StatusCode)
	}

	var resp gallery.Stack
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	g, err = galleries.Fetch(ctx, g.ID)
	if err != nil {
		t.Fatalf("fetch gallery: %v", err)
	}

	stack, err = g.Stack(stack.ID)
	if err != nil {
		t.Fatalf("get stack: %v", err)
	}

	if !reflect.DeepEqual(stack, resp) {
		t.Fatalf("invalid response. want=%v got=%v", stack, resp)
	}

	if stack.Original().Name != newName {
		t.Fatalf("Name should be %q; is %q", newName, stack.Original().Name)
	}
}

func newGalleryTest(t *testing.T, ctx context.Context) (
	http.Handler,
	gallery.Repository,
	*gallery.Lookup,
	func() (event.Bus, event.Store, aggregate.Repository, media.Storage, <-chan error),
) {
	ebus := chanbus.New()
	estore := eventstore.WithBus(memstore.New(), ebus)
	repo := repository.New(estore)

	galleries := gallery.GoesRepository(repo)

	lookup, errs := newGalleryLookup(t, ctx, ebus, estore)
	storage := newStorage()

	srv := mediaserver.New(mediaserver.WithGalleries(galleries, lookup, storage))

	return srv, galleries, lookup, func() (event.Bus, event.Store, aggregate.Repository, media.Storage, <-chan error) {
		return ebus, estore, repo, storage, errs
	}
}

func newGalleryLookup(t *testing.T, ctx context.Context, ebus event.Bus, estore event.Store) (*gallery.Lookup, <-chan error) {
	lookup := gallery.NewLookup()
	errs, err := lookup.Project(ctx, ebus, estore)
	if err != nil {
		t.Fatalf("Lookup.Project failed with %q", err)
	}
	return lookup, errs
}
