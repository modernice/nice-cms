package gallery

import (
	"github.com/google/uuid"
	"github.com/modernice/goes/event"
)

const (
	Created       = "cms.media.image.gallery.created"
	ImageUploaded = "cms.media.image.gallery.image_uploaded"
	StackDeleted  = "cms.media.image.gallery.stack_deleted"
	StackTagged   = "cms.media.image.gallery.stack_tagged"
	StackUntagged = "cms.media.image.gallery.stack_untagged"
	StackRenamed  = "cms.media.image.gallery.stack_renamed"
	StackUpdated  = "cms.media.image.gallery.stack_updated"
)

type CreatedData struct {
	Name string
}

type ImageUploadedData struct {
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

func RegisterEvents(r event.Registry) {
	r.Register(Created, func() event.Data { return CreatedData{} })
	r.Register(ImageUploaded, func() event.Data { return ImageUploadedData{} })
	r.Register(StackDeleted, func() event.Data { return StackDeletedData{} })
	r.Register(StackTagged, func() event.Data { return StackTaggedData{} })
	r.Register(StackUntagged, func() event.Data { return StackUntaggedData{} })
	r.Register(StackRenamed, func() event.Data { return StackRenamedData{} })
	r.Register(StackUpdated, func() event.Data { return StackUpdatedData{} })
}
