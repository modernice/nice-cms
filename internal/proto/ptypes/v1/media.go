package ptypes

import (
	protomedia "github.com/modernice/nice-cms/internal/proto/gen/media/v1"
	"github.com/modernice/nice-cms/internal/slice"
	"github.com/modernice/nice-cms/media"
	"github.com/modernice/nice-cms/media/document"
	"github.com/modernice/nice-cms/media/image/gallery"
)

// StorageFileProto encodes a File.
func StorageFileProto(f media.File) *protomedia.StorageFile {
	return &protomedia.StorageFile{
		Name:     f.Name,
		Disk:     f.Disk,
		Path:     f.Path,
		Filesize: int64(f.Filesize),
		Tags:     f.Tags,
	}
}

// StorageFile decodes a File.
func StorageFile(f *protomedia.StorageFile) media.File {
	return media.NewFile(f.GetName(), f.GetDisk(), f.GetPath(), int(f.GetFilesize()), f.GetTags()...)
}

// StorageImageProto encodes an Image.
func StorageImageProto(img media.Image) *protomedia.StorageImage {
	return &protomedia.StorageImage{
		File:   StorageFileProto(img.File),
		Width:  int64(img.Width),
		Height: int64(img.Height),
	}
}

// StorageImage decodes an Image.
func StorageImage(img *protomedia.StorageImage) media.Image {
	return media.NewImage(
		int(img.GetWidth()),
		int(img.GetHeight()),
		img.GetFile().GetName(),
		img.GetFile().GetDisk(),
		img.GetFile().GetPath(),
		int(img.GetFile().Filesize),
	)
}

// StorageDocumentProto encodes a Document.
func StorageDocumentProto(doc media.Document) *protomedia.StorageDocument {
	return &protomedia.StorageDocument{
		File: StorageFileProto(doc.File),
	}
}

// StorageDocument decodes a Document.
func StorageDocument(doc *protomedia.StorageDocument) media.Document {
	return media.NewDocument(
		doc.GetFile().GetName(),
		doc.GetFile().GetDisk(),
		doc.GetFile().GetPath(),
		int(doc.GetFile().Filesize),
	)
}

func ShelfProto(s document.JSONShelf) *protomedia.Shelf {
	return &protomedia.Shelf{
		Id:        UUIDProto(s.ID),
		Name:      s.Name,
		Documents: slice.Map(s.Documents, ShelfDocumentProto).([]*protomedia.ShelfDocument),
	}
}

func Shelf(s *protomedia.Shelf) document.JSONShelf {
	return document.JSONShelf{
		ID:        UUID(s.GetId()),
		Name:      s.GetName(),
		Documents: slice.Map(s.GetDocuments(), ShelfDocument).([]document.Document),
	}
}

// ShelfDocumentProto encodes a Document.
func ShelfDocumentProto(doc document.Document) *protomedia.ShelfDocument {
	return &protomedia.ShelfDocument{
		Document:   StorageDocumentProto(doc.Document),
		Id:         UUIDProto(doc.ID),
		UniqueName: doc.UniqueName,
	}
}

// ShelfDocument decodes a Document.
func ShelfDocument(doc *protomedia.ShelfDocument) document.Document {
	return document.Document{
		Document:   StorageDocument(doc.GetDocument()),
		ID:         UUID(doc.GetId()),
		UniqueName: doc.GetUniqueName(),
	}
}

func GalleryProto(g gallery.JSONGallery) *protomedia.Gallery {
	return &protomedia.Gallery{
		Id:     UUIDProto(g.ID),
		Name:   g.Name,
		Stacks: slice.Map(g.Stacks, GalleryStackProto).([]*protomedia.Stack),
	}
}

func Gallery(g *protomedia.Gallery) gallery.JSONGallery {
	return gallery.JSONGallery{
		ID:     UUID(g.GetId()),
		Name:   g.GetName(),
		Stacks: slice.Map(g.GetStacks(), GalleryStack).([]gallery.Stack),
	}
}

func GalleryStackProto(s gallery.Stack) *protomedia.Stack {
	return &protomedia.Stack{
		Id:     UUIDProto(s.ID),
		Images: slice.Map(s.Images, GalleryImageProto).([]*protomedia.StackImage),
	}
}

func GalleryStack(s *protomedia.Stack) gallery.Stack {
	return gallery.Stack{
		ID:     UUID(s.GetId()),
		Images: slice.Map(s.GetImages(), GalleryImage).([]gallery.Image),
	}
}

func GalleryImageProto(img gallery.Image) *protomedia.StackImage {
	return &protomedia.StackImage{
		Image:    StorageImageProto(img.Image),
		Original: img.Original,
		Size:     img.Size,
	}
}

func GalleryImage(img *protomedia.StackImage) gallery.Image {
	return gallery.Image{
		Image:    StorageImage(img.GetImage()),
		Original: img.GetOriginal(),
		Size:     img.GetSize(),
	}
}
