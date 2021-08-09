package ptypes

import (
	protomedia "github.com/modernice/nice-cms/internal/proto/gen/media/v1"
	"github.com/modernice/nice-cms/media"
	"github.com/modernice/nice-cms/media/document"
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
