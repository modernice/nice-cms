package gallery

import (
	"github.com/google/uuid"
	"github.com/modernice/goes/codec"
)

const (
	Created       = "cms.media.image.gallery.created"
	ImageUploaded = "cms.media.image.gallery.image_uploaded"
	ImageReplaced = "cms.media.image.gallery.stack_replaced"
	StackDeleted  = "cms.media.image.gallery.stack_deleted"
	StackTagged   = "cms.media.image.gallery.stack_tagged"
	StackUntagged = "cms.media.image.gallery.stack_untagged"
	StackRenamed  = "cms.media.image.gallery.stack_renamed"
	StackUpdated  = "cms.media.image.gallery.stack_updated"
	Sorted        = "cms.media.image.gallery.sorted"
)

type CreatedData struct {
	Name string
}

type ImageUploadedData struct {
	Stack Stack
}

type ImageReplacedData struct {
	Stack Stack
}

type StackDeletedData struct {
	Stack Stack
}

type StackTaggedData struct {
	StackID uuid.UUID
	Tags    []string
}

type StackUntaggedData struct {
	StackID uuid.UUID
	Tags    []string
}

type StackRenamedData struct {
	StackID uuid.UUID
	OldName string
	Name    string
}

type StackUpdatedData struct {
	Stack Stack
}

type SortedData struct {
	Sorting []uuid.UUID
}

func RegisterEvents(r *codec.GobRegistry) {
	r.GobRegister(Created, func() interface{} { return CreatedData{} })
	r.GobRegister(ImageUploaded, func() interface{} { return ImageUploadedData{} })
	r.GobRegister(ImageReplaced, func() interface{} { return ImageReplacedData{} })
	r.GobRegister(StackDeleted, func() interface{} { return StackDeletedData{} })
	r.GobRegister(StackTagged, func() interface{} { return StackTaggedData{} })
	r.GobRegister(StackUntagged, func() interface{} { return StackUntaggedData{} })
	r.GobRegister(StackRenamed, func() interface{} { return StackRenamedData{} })
	r.GobRegister(StackUpdated, func() interface{} { return StackUpdatedData{} })
	r.GobRegister(Sorted, func() interface{} { return SortedData{} })
}
