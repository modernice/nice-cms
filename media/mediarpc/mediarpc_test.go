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
	"github.com/modernice/nice-cms/media/image/gallery"
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
		protomedia.RegisterMediaServiceServer(s, mediarpc.NewServer(shelfs, lookup, nil, nil, storage))
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
		protomedia.RegisterMediaServiceServer(s, mediarpc.NewServer(shelfs, lookup, nil, nil, storage))
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
		protomedia.RegisterMediaServiceServer(s, mediarpc.NewServer(shelfs, lookup, nil, nil, storage))
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

func TestServer_FetchShelf(t *testing.T) {
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

	if _, err := shelf.Add(ctx, storage, buf, "unique-foo", "foo", "foo-disk", "/foo.png"); err != nil {
		t.Fatalf("add document: %v", err)
	}

	if err := shelfs.Save(ctx, shelf); err != nil {
		t.Fatalf("save shelf: %v", err)
	}

	_, dial := grpctest.NewServer(func(s *grpc.Server) {
		protomedia.RegisterMediaServiceServer(s, mediarpc.NewServer(shelfs, lookup, nil, nil, storage))
	})
	conn := dial()
	defer conn.Close()

	client := mediarpc.NewClient(conn)

	fetched, err := client.FetchShelf(ctx, shelf.ID)
	if err != nil {
		t.Fatalf("FetchShelf failed with %q", err)
	}

	if !cmp.Equal(shelf.JSON(), fetched) {
		t.Fatal(cmp.Diff(shelf.JSON(), fetched))
	}
}

func TestServer_LookupGalleryByName(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupEvents, _, setupAggregates := testutil.Goes()
	ebus, estore, _ := setupEvents()
	aggregates := setupAggregates()

	galleries := gallery.GoesRepository(aggregates)
	lookup := gallery.NewLookup()
	go lookup.Project(ctx, ebus, estore)

	g := gallery.New(uuid.New())
	g.Create("foo")

	if err := galleries.Save(ctx, g); err != nil {
		t.Fatalf("save gallery: %v", err)
	}

	storage := media.NewStorage(media.ConfigureDisk("foo-disk", media.MemoryDisk()))
	_, dial := grpctest.NewServer(func(s *grpc.Server) {
		protomedia.RegisterMediaServiceServer(s, mediarpc.NewServer(nil, nil, galleries, lookup, storage))
	})
	conn := dial()
	defer conn.Close()

	client := mediarpc.NewClient(conn)

	id, found, err := client.LookupGalleryByName(ctx, "foo")
	if err != nil {
		t.Fatalf("LookupGalleryByName failed with %q", err)
	}

	if !found {
		t.Fatalf("LookupGalleryByName could not find UUID for %q", "foo")
	}

	if id != g.ID {
		t.Fatalf("ID should be %q; is %q", g.ID, id)
	}
}

func TestServer_LookupGalleryStackByName(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupEvents, _, setupAggregates := testutil.Goes()
	ebus, estore, _ := setupEvents()
	aggregates := setupAggregates()

	galleries := gallery.GoesRepository(aggregates)
	lookup := gallery.NewLookup()
	lookup.Project(ctx, ebus, estore)
	storage := media.NewStorage(media.ConfigureDisk("foo-disk", media.MemoryDisk()))

	g := gallery.New(uuid.New())
	g.Create("foo")

	_, buf := imggen.ColoredRectangle(800, 600, color.Black)

	stack, err := g.Upload(ctx, storage, buf, "foo", "foo-disk", "/foo.png")
	if err != nil {
		t.Fatalf("upload image: %v", err)
	}

	if err := galleries.Save(ctx, g); err != nil {
		t.Fatalf("save gallery: %v", err)
	}

	_, dial := grpctest.NewServer(func(s *grpc.Server) {
		protomedia.RegisterMediaServiceServer(s, mediarpc.NewServer(nil, nil, galleries, lookup, storage))
	})
	conn := dial()
	defer conn.Close()

	client := mediarpc.NewClient(conn)

	id, found, err := client.LookupGalleryStackByName(ctx, g.ID, "foo")
	if err != nil {
		t.Fatalf("LookupGalleryStackByName failed with %q", err)
	}

	if !found {
		t.Fatalf("LookupGalleryStackByName could not find UUID for %q", "foo")
	}

	if id != stack.ID {
		t.Fatalf("ID should be %q; is %q", stack.ID, id)
	}
}

func TestServer_UploadImage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, _, setupAggregates := testutil.Goes()
	// ebus, estore, _ := setupEvents()
	aggregates := setupAggregates()

	galleries := gallery.GoesRepository(aggregates)

	g := gallery.New(uuid.New())
	g.Create("foo")

	if err := galleries.Save(ctx, g); err != nil {
		t.Fatalf("save gallery: %v", err)
	}

	storage := media.NewStorage(media.ConfigureDisk("foo-disk", media.MemoryDisk()))
	_, dial := grpctest.NewServer(func(s *grpc.Server) {
		protomedia.RegisterMediaServiceServer(s, mediarpc.NewServer(nil, nil, galleries, nil, storage))
	})
	conn := dial()
	defer conn.Close()

	client := mediarpc.NewClient(conn)

	_, buf := imggen.ColoredRectangle(800, 600, color.Black)
	filesize := buf.Len()

	name := "foo"
	disk := "foo-disk"
	path := "/foo.png"
	stack, err := client.UploadImage(ctx, g.ID, buf, name, disk, path)
	if err != nil {
		t.Fatalf("UploadImage failed with %q", err)
	}

	if stack.Original().Filesize != filesize {
		t.Fatalf("Filesize of original image should be %d; is %d", filesize, stack.Original().Filesize)
	}
}

func TestServer_ReplaceImage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, _, setupAggregates := testutil.Goes()
	// ebus, estore, _ := setupEvents()
	aggregates := setupAggregates()

	storage := media.NewStorage(media.ConfigureDisk("foo-disk", media.MemoryDisk()))
	galleries := gallery.GoesRepository(aggregates)

	g := gallery.New(uuid.New())
	g.Create("foo")

	_, buf := imggen.ColoredRectangle(800, 600, color.Black)
	name := "foo"
	disk := "foo-disk"
	path := "/foo.png"

	stack, err := g.Upload(ctx, storage, buf, name, disk, path)
	if err != nil {
		t.Fatalf("upload image: %v", err)
	}

	if err := galleries.Save(ctx, g); err != nil {
		t.Fatalf("save gallery: %v", err)
	}

	_, dial := grpctest.NewServer(func(s *grpc.Server) {
		protomedia.RegisterMediaServiceServer(s, mediarpc.NewServer(nil, nil, galleries, nil, storage))
	})
	conn := dial()
	defer conn.Close()

	client := mediarpc.NewClient(conn)

	_, buf = imggen.ColoredRectangle(1200, 1000, color.White)

	replaced, err := client.ReplaceImage(ctx, g.ID, stack.ID, buf)
	if err != nil {
		t.Fatalf("ReplacImage failed with %q", err)
	}

	if replaced.Original().Width != 1200 || replaced.Original().Height != 1000 {
		t.Fatalf(
			"replaced image should have Width=%d Height=%d; got Width=%d Height=%d",
			1200, 1000,
			replaced.Original().Width, replaced.Original().Height,
		)
	}

	g, err = galleries.Fetch(ctx, g.ID)
	if err != nil {
		t.Fatalf("fetch gallery: %v", err)
	}

	gstack, err := g.Stack(stack.ID)
	if err != nil {
		t.Fatalf("get stack: %v", err)
	}

	if !cmp.Equal(replaced, gstack) {
		t.Fatal(cmp.Diff(replaced, gstack))
	}
}

func TestServer_FetchGallery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, _, setupAggregates := testutil.Goes()
	// ebus, estore, _ := setupEvents()
	aggregates := setupAggregates()

	storage := media.NewStorage(media.ConfigureDisk("foo-disk", media.MemoryDisk()))
	galleries := gallery.GoesRepository(aggregates)

	g := gallery.New(uuid.New())
	g.Create("foo")

	_, buf := imggen.ColoredRectangle(800, 600, color.Black)
	name := "foo"
	disk := "foo-disk"
	path := "/foo.png"

	if _, err := g.Upload(ctx, storage, buf, name, disk, path); err != nil {
		t.Fatalf("upload image: %v", err)
	}

	if err := galleries.Save(ctx, g); err != nil {
		t.Fatalf("save gallery: %v", err)
	}

	_, dial := grpctest.NewServer(func(s *grpc.Server) {
		protomedia.RegisterMediaServiceServer(s, mediarpc.NewServer(nil, nil, galleries, nil, storage))
	})
	conn := dial()
	defer conn.Close()

	client := mediarpc.NewClient(conn)

	fetched, err := client.FetchGallery(ctx, g.ID)
	if err != nil {
		t.Fatalf("FetchGallery failed with %q", err)
	}

	want := g.JSON()
	if !cmp.Equal(want, fetched) {
		t.Fatal(cmp.Diff(want, fetched))
	}
}

func newDocumentLookup(ctx context.Context, bus event.Bus, store event.Store) *document.Lookup {
	l := document.NewLookup()
	go l.Project(ctx, bus, store)
	return l
}
