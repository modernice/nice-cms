package document_test

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/modernice/goes/test"
	"github.com/modernice/nice-cms/media"
	"github.com/modernice/nice-cms/media/document"
	"github.com/modernice/nice-cms/media/mock_media"
)

//go:embed testdata/example.pdf
var examplePDF []byte

//go:embed testdata/example2.pdf
var examplePDF2 []byte

var (
	exampleShelfName  = "foo-shelf"
	exampleUniqueName = "example-doc"
	exampleName       = "Example Document"
	exampleDisk       = "foo-disk"
	examplePath       = "/example/example.pdf"
)

func TestNewShelf(t *testing.T) {
	id := uuid.New()
	doc := document.NewShelf(id)

	if doc.AggregateName() != "cms.media.document.shelf" {
		t.Fatalf("AggregateName should return %q; got %q", "cms.media.document.shelf", doc.AggregateName())
	}

	if doc.AggregateID() != id {
		t.Fatalf("AggregateID should return %q; got %q", id, doc.AggregateID())
	}
}

func TestShelf_Create(t *testing.T) {
	shelf := document.NewShelf(uuid.New())

	if err := shelf.Create(exampleShelfName); err != nil {
		t.Fatalf("Create shouldn't fail; failed with %q", err)
	}

	if shelf.Name != exampleShelfName {
		t.Fatalf("Name should be %q; is %q", exampleShelfName, shelf.Name)
	}

	test.Change(t, shelf, document.ShelfCreated, test.WithEventData(document.ShelfCreatedData{Name: exampleShelfName}))
}

func TestShelf_Create_alreadyCreated(t *testing.T) {
	shelf := document.NewShelf(uuid.New())

	shelf.Create(exampleShelfName)

	if err := shelf.Create(exampleShelfName); !errors.Is(err, document.ErrShelfAlreadyCreated) {
		t.Fatalf("Create should fail with %q if the Shelf was already created; got %q", document.ErrShelfAlreadyCreated, err)
	}

	test.Change(t, shelf, document.ShelfCreated, test.WithEventData(document.ShelfCreatedData{Name: exampleShelfName}), test.Exactly(1))
}

func TestShelf_Create_emptyName(t *testing.T) {
	shelf := document.NewShelf(uuid.New())

	name := "   	"
	if err := shelf.Create(name); !errors.Is(err, document.ErrEmptyName) {
		t.Fatalf("Create should fail with %q when provided an empty name; got %q", document.ErrEmptyName, err)
	}

	if shelf.Name != "" {
		t.Fatalf("Name should be %q; got %q", "", shelf.Name)
	}

	test.NoChange(t, shelf, document.ShelfCreated)
}

func TestShelf_Add_notCreated(t *testing.T) {
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))
	shelf := document.NewShelf(uuid.New())

	pdf := newPDF()

	if _, err := shelf.Add(context.Background(), storage, pdf, exampleUniqueName, exampleName, exampleDisk, examplePath); !errors.Is(err, document.ErrShelfNotCreated) {
		t.Fatalf("Add should fail with %q if the Shelf wasn't created yet; got %q", document.ErrShelfNotCreated, err)
	}

	test.NoChange(t, shelf, document.DocumentAdded)
}

func TestShelf_Add(t *testing.T) {
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))
	shelf := document.NewShelf(uuid.New())
	shelf.Create(exampleShelfName)

	pdf := newPDF()

	doc, err := shelf.Add(context.Background(), storage, pdf, exampleUniqueName, exampleName, exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("Add shouldn't fail; failed with %q", err)
	}

	if doc.ID == uuid.Nil {
		t.Fatalf("ID should be non-nil; is %q", doc.ID)
	}

	if doc.UniqueName != exampleUniqueName {
		t.Fatalf("DocumentName should be %q; is %q", exampleUniqueName, doc.UniqueName)
	}

	if doc.Name != exampleName {
		t.Fatalf("Name should be %q; is %q", exampleName, doc.Name)
	}

	if doc.Disk != exampleDisk {
		t.Fatalf("Disk should be %q; is %q", exampleDisk, doc.Disk)
	}

	if doc.Path != examplePath {
		t.Fatalf("Path should be %q; is %q", examplePath, doc.Path)
	}

	if doc.Filesize != len(examplePDF) {
		t.Fatalf("Filesize should be %d; is %d", len(examplePDF), doc.Filesize)
	}

	test.Change(t, shelf, document.DocumentAdded, test.WithEventData(document.DocumentAddedData{Document: doc}))

	disk, err := storage.Disk(doc.Disk)
	if err != nil {
		t.Fatalf("get %q storage disk: %v", doc.Disk, err)
	}

	b, err := disk.Get(context.Background(), doc.Path)
	if err != nil {
		t.Fatalf("failed to get storage file %q. does the Shelf upload the file?", doc.Path)
	}

	if !bytes.Equal(b, examplePDF) {
		t.Fatalf("file in storage is not the same as uploaded file")
	}
}

func TestShelf_Add_duplicateUniqueName(t *testing.T) {
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))

	shelf := document.NewShelf(uuid.New())
	shelf.Create(exampleShelfName)

	pdf := newPDF()

	doc, err := shelf.Add(context.Background(), storage, pdf, exampleUniqueName, exampleName, exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("Add failed with %q", err)
	}

	pdf = newPDF()

	if _, err := shelf.Add(context.Background(), storage, pdf, exampleUniqueName, exampleName, exampleDisk, examplePath); !errors.Is(err, document.ErrDuplicateUniqueName) {
		t.Fatalf("Add should fail with %q; got %q", document.ErrDuplicateUniqueName, err)
	}

	test.Change(t, shelf, document.DocumentAdded, test.WithEventData(document.DocumentAddedData{Document: doc}), test.Exactly(1))
}

func TestShelf_Remove_notCreated(t *testing.T) {
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))
	shelf := document.NewShelf(uuid.New())

	if err := shelf.Remove(context.Background(), storage, uuid.New()); !errors.Is(err, document.ErrShelfNotCreated) {
		t.Fatalf("Remove should fail with %q if the Shelf wasn't created yet; got %q", document.ErrShelfNotCreated, err)
	}

	test.NoChange(t, shelf, document.DocumentRemoved)
}

func TestShelf_Remove(t *testing.T) {
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))
	shelf := document.NewShelf(uuid.New())
	shelf.Create(exampleShelfName)

	pdf := newPDF()

	doc, err := shelf.Add(context.Background(), storage, pdf, exampleUniqueName, exampleName, exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("Add failed with %q", err)
	}

	if err := shelf.Remove(context.Background(), storage, doc.ID); err != nil {
		t.Fatalf("Remove shouldn't fail; failed with %q", err)
	}

	if _, err := shelf.Document(doc.ID); !errors.Is(err, document.ErrNotFound) {
		t.Fatalf("Document should return %q for a deleted Document; got %q", document.ErrNotFound, err)
	}

	disk, err := storage.Disk(doc.Disk)
	if err != nil {
		t.Fatalf("get %q storage disk: %v", doc.Disk, err)
	}

	if _, err := disk.Get(context.Background(), doc.Path); err == nil {
		t.Fatalf("document should be deleted from storage")
	}

	test.Change(t, shelf, document.DocumentRemoved, test.WithEventData(document.DocumentRemovedData{Document: doc}))
}

func TestShelf_Remove_failingStorage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))
	shelf := document.NewShelf(uuid.New())
	shelf.Create(exampleShelfName)

	pdf := newPDF()

	doc, err := shelf.Add(context.Background(), storage, pdf, exampleUniqueName, exampleName, exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("Add failed with %q", err)
	}

	mockDisk := mock_media.NewMockStorageDisk(ctrl)
	mockStorage := media.NewStorage(media.ConfigureDisk(exampleDisk, mockDisk))

	mockError := errors.New("mock error")
	mockDisk.EXPECT().Delete(gomock.Any(), doc.Path).Return(mockError)

	if err := shelf.Remove(context.Background(), mockStorage, doc.ID); err != nil {
		t.Fatalf("Remove shouldn't fail; failed with %q", err)
	}

	if _, err := shelf.Document(doc.ID); !errors.Is(err, document.ErrNotFound) {
		t.Fatalf("Document should return %q for a deleted Document; got %q", document.ErrNotFound, err)
	}

	test.Change(t, shelf, document.DocumentRemoved, test.WithEventData(document.DocumentRemovedData{
		Document:    doc,
		DeleteError: mockError.Error(),
	}))
}

func TestShelf_Replace(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))
	shelf := document.NewShelf(uuid.New())
	shelf.Create(exampleShelfName)

	pdf := newPDF()

	doc, err := shelf.Add(ctx, storage, pdf, exampleUniqueName, exampleName, exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("Add failed with %q", err)
	}

	pdf = newPDF2()

	replaced, err := shelf.Replace(ctx, storage, pdf, doc.ID)
	if err != nil {
		t.Fatalf("Replace failed with %q", err)
	}

	disk, _ := storage.Disk(doc.Disk)
	content, err := disk.Get(ctx, doc.Path)
	if err != nil {
		t.Fatalf("storage failed with %q", err)
	}

	if !bytes.Equal(content, examplePDF2) {
		t.Fatalf("storage file should have been replaced")
	}

	if replaced.Filesize != len(examplePDF2) {
		t.Fatalf("Filesize should be %d; is %d", len(examplePDF2), replaced.Filesize)
	}

	test.Change(t, shelf, document.DocumentReplaced, test.WithEventData(document.DocumentReplacedData{
		Document: replaced,
	}))
}

func TestShelf_RenameDocument(t *testing.T) {
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))
	shelf := document.NewShelf(uuid.New())
	shelf.Create(exampleShelfName)

	pdf := newPDF()

	doc, err := shelf.Add(context.Background(), storage, pdf, exampleUniqueName, exampleName, exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("Add failed with %q", err)
	}

	renamed, err := shelf.RenameDocument(doc.ID, "New name")
	if err != nil {
		t.Fatalf("RenameDocument failed with %q", err)
	}

	if renamed.Name != "New name" {
		t.Fatalf("renamed Name should be %q; is %q", "New name", renamed.Name)
	}

	doc, err = shelf.Document(renamed.ID)
	if err != nil {
		t.Fatalf("Document failed with %q", err)
	}

	if !reflect.DeepEqual(doc, renamed) {
		t.Fatalf("Rename returned wrong Document. want=%v got=%v", renamed, doc)
	}

	test.Change(t, shelf, document.DocumentRenamed, test.WithEventData(document.DocumentRenamedData{
		DocumentID: doc.ID,
		OldName:    exampleName,
		Name:       "New name",
	}))
}

func TestShelf_MakeUnique(t *testing.T) {
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))
	shelf := document.NewShelf(uuid.New())
	shelf.Create(exampleShelfName)

	pdf := newPDF()

	doc, err := shelf.Add(context.Background(), storage, pdf, "", exampleName, exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("Add failed with %q", err)
	}

	unique, err := shelf.MakeUnique(doc.ID, exampleUniqueName)
	if err != nil {
		t.Fatalf("MakeUnique failed with %q", err)
	}

	if unique.UniqueName != exampleUniqueName {
		t.Fatalf("UniqueName should be %q; is %q", exampleUniqueName, unique.UniqueName)
	}

	doc, err = shelf.Find(exampleUniqueName)
	if err != nil {
		t.Fatalf("Find failed with %q", err)
	}

	if !reflect.DeepEqual(doc, unique) {
		t.Fatalf("Find returned wrong Document. want=%v got=%v", unique, doc)
	}

	test.Change(t, shelf, document.DocumentMadeUnique, test.WithEventData(document.DocumentMadeUniqueData{
		DocumentID: doc.ID,
		UniqueName: exampleUniqueName,
	}))
}

func TestShelf_MakeUnique_duplicateUniqueName(t *testing.T) {
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))
	shelf := document.NewShelf(uuid.New())
	shelf.Create(exampleShelfName)

	pdf := newPDF()

	if _, err := shelf.Add(context.Background(), storage, pdf, exampleUniqueName, exampleName, exampleDisk, examplePath); err != nil {
		t.Fatalf("Add failed with %q", err)
	}

	doc, err := shelf.Add(context.Background(), storage, pdf, "", exampleName, exampleDisk, "/example/example2.pdf")
	if err != nil {
		t.Fatalf("Add failed with %q", err)
	}

	if _, err := shelf.MakeUnique(doc.ID, exampleUniqueName); !errors.Is(err, document.ErrDuplicateUniqueName) {
		t.Fatalf("MakeUnique should fail with %q; got %q", document.ErrDuplicateUniqueName, err)
	}

	test.NoChange(t, shelf, document.DocumentMadeUnique)
}

func TestShelf_MakeUnique_rename(t *testing.T) {
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))
	shelf := document.NewShelf(uuid.New())
	shelf.Create(exampleShelfName)

	pdf := newPDF()

	doc, err := shelf.Add(context.Background(), storage, pdf, exampleUniqueName, exampleName, exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("Add failed with %q", err)
	}

	name := "new-unique-name"
	unique, err := shelf.MakeUnique(doc.ID, name)
	if err != nil {
		t.Fatalf("MakeUnique failed with %q", err)
	}

	if unique.UniqueName != name {
		t.Fatalf("UniqueName should be %q; is %q", name, unique.UniqueName)
	}

	doc, err = shelf.Find(name)
	if err != nil {
		t.Fatalf("Find failed with %q", err)
	}

	if !reflect.DeepEqual(doc, unique) {
		t.Fatalf("Find returned wrong Document. want=%v got=%v", unique, doc)
	}
}

func TestShelf_MakeUnique_emptyName(t *testing.T) {
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))
	shelf := document.NewShelf(uuid.New())
	shelf.Create(exampleShelfName)

	pdf := newPDF()

	doc, err := shelf.Add(context.Background(), storage, pdf, "", exampleName, exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("Add failed with %q", err)
	}

	name := ""
	if _, err := shelf.MakeUnique(doc.ID, name); !errors.Is(err, document.ErrEmptyName) {
		t.Fatalf("MakeUnique should fail with %q; got %q", document.ErrEmptyName, err)
	}

	test.NoChange(t, shelf, document.DocumentMadeUnique)
}

func TestShelf_MakeNonUnique(t *testing.T) {
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))
	shelf := document.NewShelf(uuid.New())
	shelf.Create(exampleShelfName)

	pdf := newPDF()

	doc, err := shelf.Add(context.Background(), storage, pdf, exampleUniqueName, exampleName, exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("Add failed with %q", err)
	}

	nonunique, err := shelf.MakeNonUnique(doc.ID)
	if err != nil {
		t.Fatalf("MakeNonUnique failed with %q", err)
	}

	if nonunique.UniqueName != "" {
		t.Fatalf("UniqueName should be %q; is %q", "", nonunique.UniqueName)
	}

	test.Change(t, shelf, document.DocumentMadeNonUnique, test.WithEventData(document.DocumentMadeNonUniqueData{
		DocumentID: doc.ID,
		UniqueName: exampleUniqueName,
	}))
}

func TestShelf_Tag_Untag(t *testing.T) {
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))
	shelf := document.NewShelf(uuid.New())
	shelf.Create(exampleShelfName)

	pdf := newPDF()

	doc, err := shelf.Add(context.Background(), storage, pdf, exampleUniqueName, exampleName, exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("Add failed with %q", err)
	}

	tagged, err := shelf.Tag(doc.ID, "foo", "bar", "baz", "baz")
	if err != nil {
		t.Fatalf("Tag failed with %q", err)
	}

	if len(tagged.Tags) != 3 {
		t.Fatalf("Document should have %d tags; has %d", 3, len(tagged.Tags))
	}

	want := []string{"foo", "bar", "baz"}
	if !tagged.HasTag(want...) {
		t.Fatalf("Document should have %v tags; has %v", want, tagged.Tags)
	}

	doc, err = shelf.Document(tagged.ID)
	if err != nil {
		t.Fatalf("Document failed with %q", err)
	}

	if !reflect.DeepEqual(doc, tagged) {
		t.Fatalf("Document returned wrong Document. want=%v got=%v", tagged, doc)
	}

	test.Change(t, shelf, document.DocumentTagged, test.WithEventData(document.DocumentTaggedData{
		DocumentID: tagged.ID,
		Tags:       want,
	}))

	untagged, err := shelf.Untag(doc.ID, "bar")
	if err != nil {
		t.Fatalf("Untag failed with %q", err)
	}

	if len(untagged.Tags) != 2 {
		t.Fatalf("Document should have %d tags; has %d", 2, len(untagged.Tags))
	}

	want = []string{"foo", "baz"}
	if !untagged.HasTag(want...) {
		t.Fatalf("Document should have %v tags; has %v", want, untagged.Tags)
	}

	doc, err = shelf.Document(untagged.ID)
	if err != nil {
		t.Fatalf("Document failed with %q", err)
	}

	if !reflect.DeepEqual(untagged, doc) {
		t.Fatalf("Document returned wrong Document. want=%v got=%v", untagged, doc)
	}

	test.Change(t, shelf, document.DocumentUntagged, test.WithEventData(document.DocumentUntaggedData{
		DocumentID: untagged.ID,
		Tags:       []string{"bar"},
	}))
}

func TestShelf_Search(t *testing.T) {
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))
	shelf := document.NewShelf(uuid.New())
	shelf.Create(exampleShelfName)

	pdf := newPDF()

	uniqueNames := []string{"foo", "bar", "baz"}

	makeName := func(uniqueName string) string { return uniqueName + "-doc" }
	makePath := func(uniqueName string) string { return "/examples/" + uniqueName + ".png" }

	for _, uniqueName := range uniqueNames {
		name := makeName(uniqueName)
		path := makePath(uniqueName)
		if _, err := shelf.Add(context.Background(), storage, pdf, uniqueName, name, exampleDisk, path); err != nil {
			t.Fatalf("Add failed with %q", err)
		}
	}

	for _, uniqueName := range uniqueNames {
		docs := shelf.Search(document.ForName(makeName(uniqueName)))

		if len(docs) != 1 {
			t.Fatalf("Search should return 1 Document; got %d", len(docs))
		}

		want, err := shelf.Find(uniqueName)
		if err != nil {
			t.Fatalf("Find failed with %q", err)
		}

		expectDocument(t, want, docs...)

		docs = shelf.Search(document.ForName(makeName(uniqueName), "other-unique-name"))

		if len(docs) != 1 {
			t.Fatalf("Search should return 1 Document; got %d", len(docs))
		}

		expectDocument(t, want, docs...)
	}

	docs := shelf.Search(document.ForName(makeName("foo"), makeName("baz")))

	if len(docs) != 2 {
		t.Fatalf("Search should return %d Documents; got %d", 2, len(docs))
	}

	for _, uniqueName := range []string{"foo", "baz"} {
		want, err := shelf.Find(uniqueName)
		if err != nil {
			t.Fatalf("Find failed with %q", err)
		}
		expectDocument(t, want, docs...)
	}
}

func newPDF() *bytes.Reader {
	return bytes.NewReader(examplePDF)
}

func newPDF2() *bytes.Reader {
	return bytes.NewReader(examplePDF2)
}

func expectDocument(t *testing.T, doc document.Document, docs ...document.Document) {
	for _, d := range docs {
		if reflect.DeepEqual(d, doc) {
			return
		}
	}
	t.Fatalf("expected Document %v in %v", doc, docs)
}
