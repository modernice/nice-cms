package mediaserver_test

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/modernice/goes/aggregate"
	"github.com/modernice/goes/aggregate/repository"
	"github.com/modernice/goes/event"
	"github.com/modernice/goes/event/eventbus/chanbus"
	"github.com/modernice/goes/event/eventstore"
	"github.com/modernice/goes/event/eventstore/memstore"
	"github.com/modernice/nice-cms/media"
	"github.com/modernice/nice-cms/media/document"
	"github.com/modernice/nice-cms/media/mediaserver"
)

//go:embed testdata/example.pdf
var examplePDF []byte

//go:embed testdata/example2.pdf
var examplePDF2 []byte

var (
	exampleDisk = "foo-disk"
	exampleName = "Example document"
	examplePath = "/example/example.pdf"
)

func TestMediaServer_lookupShelf(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, shelfs, _, _ := newDocumentTest(t, ctx)

	var tests []struct {
		shelfID   uuid.UUID
		shelfName string
	}

	names := []string{"foo", "bar", "baz"}
	for _, name := range names {
		shelf := document.NewShelf(uuid.New())
		shelf.Create(name)

		if err := shelfs.Save(ctx, shelf); err != nil {
			t.Fatalf("save %q Shelf: %v", shelf.Name, err)
		}

		tests = append(tests, struct {
			shelfID   uuid.UUID
			shelfName string
		}{
			shelfID:   shelf.ID,
			shelfName: shelf.Name,
		})
	}

	<-time.After(50 * time.Millisecond)

	for _, tt := range tests {
		t.Run(tt.shelfName, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", fmt.Sprintf("/shelfs/lookup/name/%s", tt.shelfName), nil)

			srv.ServeHTTP(rec, req)

			var resp struct {
				ShelfID uuid.UUID `json:"shelfId"`
			}

			if rec.Result().StatusCode != http.StatusOK {
				b, _ := io.ReadAll(rec.Result().Body)
				t.Fatalf("Response should have status code %d; has %d\n\n%s", http.StatusOK, rec.Result().StatusCode, b)
			}

			if err := json.NewDecoder(rec.Result().Body).Decode(&resp); err != nil {
				t.Fatalf("decode response: %v", err)
			}

			if resp.ShelfID == uuid.Nil {
				t.Fatalf("ShelfID should not be nil")
			}

			if _, err := shelfs.Fetch(ctx, resp.ShelfID); err != nil {
				t.Fatalf("failed to fetch Shelf %q: %v", resp.ShelfID, err)
			}
		})
	}
}

func TestMediaServer_uploadDocument(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, shelfs, _, _ := newDocumentTest(t, ctx)

	shelf := document.NewShelf(uuid.New())
	shelf.Create("foo")

	if err := shelfs.Save(ctx, shelf); err != nil {
		t.Fatalf("save Shelf: %v", err)
	}

	rec := httptest.NewRecorder()

	var formData bytes.Buffer
	multipartWriter := multipart.NewWriter(&formData)
	formFile, _ := multipartWriter.CreateFormFile("document", "document")
	io.Copy(formFile, newPDF())

	name := "Example document"
	nameField, _ := multipartWriter.CreateFormField("name")
	nameField.Write([]byte(name))

	uniqueName := "unique-doc"
	uniqueNameField, _ := multipartWriter.CreateFormField("uniqueName")
	uniqueNameField.Write([]byte(uniqueName))

	diskField, _ := multipartWriter.CreateFormField("disk")
	diskField.Write([]byte(exampleDisk))

	path := "/examples/example.pdf"
	pathField, _ := multipartWriter.CreateFormField("path")
	pathField.Write([]byte(path))

	multipartWriter.Close()

	req := httptest.NewRequest("POST", fmt.Sprintf("/shelfs/%s/documents", shelf.ID), &formData)
	req.Header.Add("Content-Type", multipartWriter.FormDataContentType())

	srv.ServeHTTP(rec, req)

	var resp document.Document

	if err := json.NewDecoder(rec.Result().Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.ID == uuid.Nil {
		t.Fatalf("response contains a non-nil DocumentID")
	}

	shelf, err := shelfs.Fetch(ctx, shelf.ID)
	if err != nil {
		t.Fatalf("fetch Shelf %q: %v", shelf.ID, err)
	}

	doc, err := shelf.Document(resp.ID)
	if err != nil {
		t.Fatalf("Shelf should contain Document %q; got error %q", resp.ID, err)
	}

	if doc.Disk != exampleDisk {
		t.Fatalf("Document.Disk should be %q; is %q", exampleDisk, doc.Disk)
	}

	if doc.Path != path {
		t.Fatalf("Document.Path should be %q; is %q", path, doc.Path)
	}

	if doc.Name != name {
		t.Fatalf("Document.Name should be %q; is %q", name, doc.Name)
	}

	if doc.UniqueName != uniqueName {
		t.Fatalf("Document.UniqueName should be %q; is %q", uniqueName, doc.UniqueName)
	}

	if !reflect.DeepEqual(doc, resp) {
		t.Fatalf("invalid response. want=%v got=%v", doc, resp)
	}
}

func TestMediaServer_deleteDocument(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, shelfs, _, _ := newDocumentTest(t, ctx)

	shelf := document.NewShelf(uuid.New())
	shelf.Create("foo")

	storage := newStorage()

	pdf := newPDF()

	doc, err := shelf.Add(ctx, storage, pdf, "", exampleName, exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("Add failed with %q", err)
	}

	if err := shelfs.Save(ctx, shelf); err != nil {
		t.Fatalf("save Shelf: %v", err)
	}

	req := httptest.NewRequest("DELETE", fmt.Sprintf("/shelfs/%s/documents/%s", shelf.ID, doc.ID), nil)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	res := rec.Result()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("response should have code %d; has code %d", http.StatusNoContent, res.StatusCode)
	}

	shelf, err = shelfs.Fetch(ctx, shelf.ID)
	if err != nil {
		t.Fatalf("fetch shelf: %v", err)
	}

	if _, err := shelf.Document(doc.ID); !errors.Is(err, document.ErrNotFound) {
		t.Fatalf("shelf.Document should return %q; got %q", document.ErrNotFound, err)
	}
}

func TestMediaServer_tagDocument(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, shelfs, _, _ := newDocumentTest(t, ctx)

	shelf := document.NewShelf(uuid.New())
	shelf.Create("foo")

	storage := newStorage()

	pdf := newPDF()

	doc, err := shelf.Add(ctx, storage, pdf, "", exampleName, exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("Add failed with %q", err)
	}

	if err := shelfs.Save(ctx, shelf); err != nil {
		t.Fatalf("save Shelf: %v", err)
	}

	body := strings.NewReader(`{"tags": ["foo", "bar", "baz"]}`)

	req := httptest.NewRequest("POST", fmt.Sprintf("/shelfs/%s/documents/%s/tags", shelf.ID, doc.ID), body)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	res := rec.Result()

	var resp document.Document
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf("response should have code %d; has code %d", http.StatusOK, res.StatusCode)
	}

	shelf, err = shelfs.Fetch(ctx, shelf.ID)
	if err != nil {
		t.Fatalf("fetch shelf: %v", err)
	}

	doc, err = shelf.Document(doc.ID)
	if err != nil {
		t.Fatalf("shelf.Document failed with %q", err)
	}

	if !reflect.DeepEqual(doc, resp) {
		t.Fatalf("invalid response. want=%v got=%v", doc, resp)
	}

	if !doc.HasTag("foo", "bar", "baz") {
		t.Fatalf("Document should have %v tags; has %v", []string{"foo", "bar", "baz"}, doc.Tags)
	}

	req = httptest.NewRequest("DELETE", fmt.Sprintf("/shelfs/%s/documents/%s/tags/foo,baz", shelf.ID, doc.ID), nil)
	rec = httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	res = rec.Result()

	resp = document.Document{}
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	shelf, err = shelfs.Fetch(ctx, shelf.ID)
	if err != nil {
		t.Fatalf("fetch shelf: %v", err)
	}

	doc, err = shelf.Document(doc.ID)
	if err != nil {
		t.Fatalf("shelf.Document failed with %q", err)
	}

	if !reflect.DeepEqual(doc, resp) {
		t.Fatalf("invalid response. want=%v got=%v", shelf, resp)
	}

	if !doc.HasTag("bar") {
		t.Fatalf("Document should have %q tag", "bar")
	}

	if doc.HasTag("foo") || doc.HasTag("baz") {
		t.Fatalf("Document shouldn't have %v tags", []string{"foo", "baz"})
	}
}

func TestMediaServer_replaceDocument(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, shelfs, _, helper := newDocumentTest(t, ctx)
	_, _, _, storage, _ := helper()

	shelf := document.NewShelf(uuid.New())
	shelf.Create("foo")

	pdf := newPDF()

	doc, err := shelf.Add(ctx, storage, pdf, "", exampleName, exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("Add failed with %q", err)
	}

	if err := shelfs.Save(ctx, shelf); err != nil {
		t.Fatalf("save Shelf: %v", err)
	}

	var formData bytes.Buffer
	formWriter := multipart.NewWriter(&formData)

	documentField, _ := formWriter.CreateFormFile("document", "document")
	documentField.Write(examplePDF2)

	formWriter.Close()

	req := httptest.NewRequest("PUT", fmt.Sprintf("/shelfs/%s/documents/%s", shelf.ID, doc.ID), &formData)
	req.Header.Set("Content-Type", formWriter.FormDataContentType())
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	res := rec.Result()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("StatusCode should be %d; is %d\n%s", http.StatusOK, res.StatusCode, body)
	}

	var resp document.Document
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	shelf, _ = shelfs.Fetch(ctx, shelf.ID)
	doc, _ = shelf.Document(doc.ID)

	if !reflect.DeepEqual(doc, resp) {
		t.Fatalf("invalid response. want=%v got=%v\n\n%s", doc, resp, cmp.Diff(doc, resp))
	}

	disk, _ := storage.Disk(doc.Disk)
	content, _ := disk.Get(ctx, doc.Path)

	if !bytes.Equal(content, examplePDF2) {
		t.Fatalf("storage file should have been replaced\n\n%s", cmp.Diff(content, examplePDF2))
	}
}

func TestMediaServer_renameDocument(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, shelfs, _, _ := newDocumentTest(t, ctx)

	shelf := document.NewShelf(uuid.New())
	shelf.Create("foo")

	storage := newStorage()

	pdf := newPDF()

	doc, err := shelf.Add(ctx, storage, pdf, "", exampleName, exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("Add failed with %q", err)
	}

	if err := shelfs.Save(ctx, shelf); err != nil {
		t.Fatalf("save Shelf: %v", err)
	}

	newName := "New name"

	body := strings.NewReader(fmt.Sprintf(`{"name": %q}`, newName))

	req := httptest.NewRequest("PATCH", fmt.Sprintf("/shelfs/%s/documents/%s", shelf.ID, doc.ID), body)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	res := rec.Result()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("response should have code %d; has code %d", http.StatusOK, res.StatusCode)
	}

	var resp document.Document
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	shelf, err = shelfs.Fetch(ctx, shelf.ID)
	if err != nil {
		t.Fatalf("fetch shelf: %v", err)
	}

	doc, err = shelf.Document(doc.ID)
	if err != nil {
		t.Fatalf("shelf.Document failed with %q", err)
	}

	if !reflect.DeepEqual(doc, resp) {
		t.Fatalf("invalid response. want=%v got=%v", doc, resp)
	}

	if doc.Name != newName {
		t.Fatalf("Document name should be %q; is %q", newName, doc.Name)
	}
}

func TestMediaServer_makeDocumentUnique(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, shelfs, _, _ := newDocumentTest(t, ctx)

	shelf := document.NewShelf(uuid.New())
	shelf.Create("foo")

	storage := newStorage()

	pdf := newPDF()

	doc, err := shelf.Add(ctx, storage, pdf, "", exampleName, exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("Add failed with %q", err)
	}

	if err := shelfs.Save(ctx, shelf); err != nil {
		t.Fatalf("save Shelf: %v", err)
	}

	uniqueName := "unique-doc"

	body := strings.NewReader(fmt.Sprintf(`{"uniqueName": %q}`, uniqueName))

	req := httptest.NewRequest("PATCH", fmt.Sprintf("/shelfs/%s/documents/%s", shelf.ID, doc.ID), body)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	res := rec.Result()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("response should have code %d; has code %d", http.StatusOK, res.StatusCode)
	}

	var resp document.Document
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	shelf, err = shelfs.Fetch(ctx, shelf.ID)
	if err != nil {
		t.Fatalf("fetch shelf: %v", err)
	}

	doc, err = shelf.Document(doc.ID)
	if err != nil {
		t.Fatalf("shelf.Document failed with %q", err)
	}

	if !reflect.DeepEqual(doc, resp) {
		t.Fatalf("invalid response. want=%v got=%v", doc, resp)
	}

	if doc.UniqueName != uniqueName {
		t.Fatalf("Document uniqueName should be %q; is %q", uniqueName, doc.UniqueName)
	}
}

func TestMediaServer_makeDocumentNonUnique(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, shelfs, _, _ := newDocumentTest(t, ctx)

	shelf := document.NewShelf(uuid.New())
	shelf.Create("foo")

	storage := newStorage()

	pdf := newPDF()

	doc, err := shelf.Add(ctx, storage, pdf, "unique-doc", exampleName, exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("Add failed with %q", err)
	}

	if err := shelfs.Save(ctx, shelf); err != nil {
		t.Fatalf("save Shelf: %v", err)
	}

	body := strings.NewReader(`{"uniqueName": ""}`)

	req := httptest.NewRequest("PATCH", fmt.Sprintf("/shelfs/%s/documents/%s", shelf.ID, doc.ID), body)
	rec := httptest.NewRecorder()

	srv.ServeHTTP(rec, req)

	res := rec.Result()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("response should have code %d; has code %d", http.StatusOK, res.StatusCode)
	}

	var resp document.Document
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	shelf, err = shelfs.Fetch(ctx, shelf.ID)
	if err != nil {
		t.Fatalf("fetch shelf: %v", err)
	}

	doc, err = shelf.Document(doc.ID)
	if err != nil {
		t.Fatalf("shelf.Document failed with %q", err)
	}

	if !reflect.DeepEqual(doc, resp) {
		t.Fatalf("invalid response. want=%v got=%v", doc, resp)
	}

	if doc.UniqueName != "" {
		t.Fatalf("Document uniqueName should be %q; is %q", "", doc.UniqueName)
	}
}

func newDocumentTest(t *testing.T, ctx context.Context) (
	http.Handler,
	document.Repository,
	*document.Lookup,
	func() (event.Bus, event.Store, aggregate.Repository, media.Storage, <-chan error),
) {
	ebus := chanbus.New()
	estore := eventstore.WithBus(memstore.New(), ebus)
	repo := repository.New(estore)

	shelfs := document.GoesRepository(repo)

	lookup, errs := newDocumentLookup(t, ctx, ebus, estore)

	storage := newStorage()
	srv := mediaserver.New(mediaserver.WithDocuments(shelfs, lookup, storage))

	return srv, shelfs, lookup, func() (event.Bus, event.Store, aggregate.Repository, media.Storage, <-chan error) {
		return ebus, estore, repo, storage, errs
	}
}

func newDocumentLookup(t *testing.T, ctx context.Context, ebus event.Bus, estore event.Store) (*document.Lookup, <-chan error) {
	lookup := document.NewLookup()
	errs, err := lookup.Project(ctx, ebus, estore)
	if err != nil {
		t.Fatalf("Lookup.Project failed with %q", err)
	}
	return lookup, errs
}

func newPDF() *bytes.Reader {
	return bytes.NewReader(examplePDF)
}

func newStorage() media.Storage {
	return media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))
}
