package mediarpc_test

import (
	"context"
	"image/color"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/modernice/goes/event"
	"github.com/modernice/nice-cms/internal/grpctest"
	"github.com/modernice/nice-cms/internal/imggen"
	protomedia "github.com/modernice/nice-cms/internal/proto/gen/media/v1"
	"github.com/modernice/nice-cms/internal/testutil"
	"github.com/modernice/nice-cms/media"
	"github.com/modernice/nice-cms/media/document"
	"github.com/modernice/nice-cms/media/mediarpc"
	"google.golang.org/grpc"
)

func TestServer_LookupDocumentByName(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupEvents, _, setupAggregates := testutil.Goes()
	ebus, estore, _ := setupEvents()
	aggregates := setupAggregates()

	shelfs := document.GoesRepository(aggregates)

	shelf := document.NewShelf(uuid.New())
	shelf.Create("foo")

	if err := shelfs.Save(ctx, shelf); err != nil {
		t.Fatalf("save shelf: %v", err)
	}

	lookup := newDocumentLookup(ctx, ebus, estore)
	storage := media.NewStorage(media.ConfigureDisk("foo-disk", media.MemoryDisk()))

	_, dial := grpctest.NewServer(func(s *grpc.Server) {
		protomedia.RegisterMediaServiceServer(s, mediarpc.NewServer(shelfs, lookup, storage))
	})
	conn := dial()
	defer conn.Close()

	client := mediarpc.NewClient(conn)

	id, ok, err := client.LookupShelfByName(ctx, "foo")
	if err != nil {
		t.Fatalf("LookupShelfByName failed with %q", err)
	}

	if !ok {
		t.Fatalf("LookupShelfByName returned false")
	}

	if id != shelf.ID {
		t.Fatalf("ID should be %q; is %q", shelf.ID, id)
	}
}

func TestServer_UploadDocument(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupEvents, _, setupAggregates := testutil.Goes()
	ebus, estore, _ := setupEvents()
	aggregates := setupAggregates()

	shelfs := document.GoesRepository(aggregates)

	shelf := document.NewShelf(uuid.New())
	shelf.Create("foo")

	if err := shelfs.Save(ctx, shelf); err != nil {
		t.Fatalf("save shelf: %v", err)
	}

	lookup := newDocumentLookup(ctx, ebus, estore)
	storage := media.NewStorage(media.ConfigureDisk("foo-disk", media.MemoryDisk()))

	_, dial := grpctest.NewServer(func(s *grpc.Server) {
		protomedia.RegisterMediaServiceServer(s, mediarpc.NewServer(shelfs, lookup, storage))
	})
	conn := dial()
	defer conn.Close()

	client := mediarpc.NewClient(conn)

	_, buf := imggen.ColoredRectangle(600, 400, color.Black)

	uniqueName := "foo"
	name := "Foo"
	disk := "foo-disk"
	path := "/foo.png"

	doc, err := client.UploadDocument(ctx, shelf.ID, buf, uniqueName, name, disk, path)
	if err != nil {
		t.Fatalf("UploadDocument failed with %q", err)
	}

	if doc.UniqueName != uniqueName {
		t.Fatalf("UniqueName should be %q; is %q", uniqueName, doc.UniqueName)
	}

	if doc.Name != name {
		t.Fatalf("Name should be %q; is %q", name, doc.Name)
	}

	if doc.Disk != disk {
		t.Fatalf("Disk should be %q; is %q", disk, doc.Disk)
	}

	if doc.Path != path {
		t.Fatalf("Path should be %q; is %q", path, doc.Path)
	}

	rshelf, err := shelfs.Fetch(ctx, shelf.ID)
	if err != nil {
		t.Fatalf("fetch shelf: %v", err)
	}

	rdoc, err := rshelf.Document(doc.ID)
	if err != nil {
		t.Fatalf("get document: %v", err)
	}

	if !cmp.Equal(doc, rdoc) {
		t.Fatal(cmp.Diff(doc, rdoc))
	}
}

func TestServer_ReplaceDocument(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupEvents, _, setupAggregates := testutil.Goes()
	ebus, estore, _ := setupEvents()
	aggregates := setupAggregates()

	lookup := newDocumentLookup(ctx, ebus, estore)
	storage := media.NewStorage(media.ConfigureDisk("foo-disk", media.MemoryDisk()))

	shelfs := document.GoesRepository(aggregates)

	shelf := document.NewShelf(uuid.New())
	shelf.Create("foo")

	_, buf := imggen.ColoredRectangle(600, 400, color.Black)

	doc, err := shelf.Add(ctx, storage, buf, "unique-foo", "foo", "foo-disk", "/foo.png")
	if err != nil {
		t.Fatalf("add document: %v", err)
	}

	if err := shelfs.Save(ctx, shelf); err != nil {
		t.Fatalf("save shelf: %v", err)
	}

	_, dial := grpctest.NewServer(func(s *grpc.Server) {
		protomedia.RegisterMediaServiceServer(s, mediarpc.NewServer(shelfs, lookup, storage))
	})
	conn := dial()
	defer conn.Close()

	client := mediarpc.NewClient(conn)

	_, buf = imggen.ColoredRectangle(800, 600, color.Black)
	filesize := buf.Len()

	replaced, err := client.ReplaceDocument(ctx, shelf.ID, doc.ID, buf)
	if err != nil {
		t.Fatalf("UploadDocument failed with %q", err)
	}

	if replaced.Filesize != filesize {
		t.Fatalf("Filesize should be %d; is %d", filesize, replaced.Filesize)
	}

	rshelf, err := shelfs.Fetch(ctx, shelf.ID)
	if err != nil {
		t.Fatalf("fetch shelf from repository: %v", err)
	}

	rdoc, err := rshelf.Document(doc.ID)
	if err != nil {
		t.Fatalf("get document: %v", err)
	}

	if !cmp.Equal(replaced, rdoc) {
		t.Fatal(cmp.Diff(replaced, rdoc))
	}
}

func newDocumentLookup(ctx context.Context, bus event.Bus, store event.Store) *document.Lookup {
	l := document.NewLookup()
	go l.Project(ctx, bus, store)
	return l
}
