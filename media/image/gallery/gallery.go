package gallery

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/modernice/cms/internal/concurrent"
	"github.com/modernice/cms/internal/unique"
	"github.com/modernice/cms/media"
	"github.com/modernice/goes/aggregate"
	"github.com/modernice/goes/event"
)

const Aggregate = "cms.media.image.gallery"

var (
	ErrEmptyName     = errors.New("empty name")
	ErrUnnamed       = errors.New("unnamed gallery")
	ErrStackNotFound = errors.New("Stack not found")
)

type Gallery struct {
	*aggregate.Base

	Name   string
	Stacks []Stack
}

// New returns a new Gallery.
func New(id uuid.UUID) *Gallery {
	return &Gallery{
		Base:   aggregate.New(Aggregate, id),
		Stacks: make([]Stack, 0),
	}
}

// Stack returns the Stack with the given UUID or ErrStackNotFound.
func (g *Gallery) Stack(id uuid.UUID) (Stack, error) {
	for _, stack := range g.Stacks {
		if stack.ID == id {
			return stack.copy(), nil
		}
	}
	return Stack{}, ErrStackNotFound
}

// ApplyEvent applies aggregate events.
func (g *Gallery) ApplyEvent(evt event.Event) {
	switch evt.Name() {
	case Created:
		g.create(evt)
	case ImageUploaded:
		g.uploadImage(evt)
	case StackDeleted:
		g.deleteStack(evt)
	case StackTagged:
		g.tagStack(evt)
	case StackUntagged:
		g.untagStack(evt)
	case StackRenamed:
		g.renameStack(evt)
	}
}

// Create creates the Gallery with the given name.
func (g *Gallery) Create(name string) error {
	if name = strings.TrimSpace(name); name == "" {
		return ErrEmptyName
	}
	aggregate.NextEvent(g, Created, CreatedData{Name: name})
	return nil
}

func (g *Gallery) create(evt event.Event) {
	data := evt.Data().(CreatedData)
	g.Name = data.Name
}

// Upload uploads the image in r to storage and returns the Stack for that image.
func (g *Gallery) Upload(ctx context.Context, storage media.Storage, r io.Reader, name, diskName, path string) (Stack, error) {
	if err := g.checkCreated(); err != nil {
		return Stack{}, err
	}

	img := media.NewImage(0, 0, name, diskName, path, 0)

	var err error
	if img, err = img.Upload(ctx, r, storage); err != nil {
		return Stack{}, fmt.Errorf("upload to %q storage: %w", diskName, err)
	}

	stack := Stack{
		ID:     uuid.New(),
		Images: []Image{{Image: img, Original: true}},
	}

	aggregate.NextEvent(g, ImageUploaded, ImageUploadedData{Stack: stack})

	return stack, nil
}

func (g *Gallery) checkCreated() error {
	if g.Name == "" {
		return ErrUnnamed
	}
	return nil
}

func (g *Gallery) uploadImage(evt event.Event) {
	data := evt.Data().(ImageUploadedData)
	g.Stacks = append(g.Stacks, data.Stack)
}

// Delete deletes the given Stack from the Gallery and Storage.
func (g *Gallery) Delete(ctx context.Context, storage media.Storage, stack Stack) error {
	if err := g.checkCreated(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	for _, img := range stack.Images {
		wg.Add(1)
		go func(img Image) {
			defer wg.Done()
			// TODO: report error (?)
			img.Delete(ctx, storage)
		}(img)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-concurrent.Wait(&wg):
		aggregate.NextEvent(g, StackDeleted, StackDeletedData{Stack: stack})
		return nil
	}
}

func (g *Gallery) deleteStack(evt event.Event) {
	data := evt.Data().(StackDeletedData)
	g.remove(data.Stack.ID)
}

func (g *Gallery) remove(id uuid.UUID) {
	for i, stack := range g.Stacks {
		if stack.ID == id {
			g.Stacks = append(g.Stacks[:i], g.Stacks[i+1:]...)
			return
		}
	}
}

// Tag tags each Image in the provided Stack through the ImageService.
func (g *Gallery) Tag(ctx context.Context, stack Stack, tags ...string) (Stack, error) {
	if err := g.checkCreated(); err != nil {
		return Stack{}, err
	}
	tags = unique.Strings(tags...)
	aggregate.NextEvent(g, StackTagged, StackTaggedData{
		StackID: stack.ID,
		Tags:    tags,
	})
	return g.Stack(stack.ID)
}

func (g *Gallery) tagStack(evt event.Event) {
	data := evt.Data().(StackTaggedData)
	stack, err := g.Stack(data.StackID)
	if err != nil {
		return
	}
	stack = stack.WithTag(data.Tags...)
	g.replace(stack.ID, stack)
}

func (g *Gallery) replace(id uuid.UUID, stack Stack) {
	for i, s := range g.Stacks {
		if s.ID == id {
			g.Stacks[i] = stack
			return
		}
	}
}

// Untag removes tags from each Image of the provided Stack.
func (g *Gallery) Untag(ctx context.Context, stack Stack, tags ...string) (Stack, error) {
	if err := g.checkCreated(); err != nil {
		return Stack{}, err
	}
	tags = unique.Strings(tags...)
	aggregate.NextEvent(g, StackUntagged, StackUntaggedData{
		StackID: stack.ID,
		Tags:    tags,
	})
	return g.Stack(stack.ID)
}

func (g *Gallery) untagStack(evt event.Event) {
	data := evt.Data().(StackUntaggedData)
	stack, err := g.Stack(data.StackID)
	if err != nil {
		return
	}
	stack = stack.WithoutTag(data.Tags...)
	g.replace(stack.ID, stack)
}

// Rename renames each Image in the given Stack to name.
func (g *Gallery) Rename(ctx context.Context, stackID uuid.UUID, name string) (Stack, error) {
	if err := g.checkCreated(); err != nil {
		return Stack{}, err
	}

	stack, err := g.Stack(stackID)
	if err != nil {
		return Stack{}, err
	}

	aggregate.NextEvent(g, StackRenamed, StackRenamedData{
		StackID: stack.ID,
		OldName: stack.Original().Name,
		Name:    name,
	})

	return g.Stack(stack.ID)
}

func (g *Gallery) renameStack(evt event.Event) {
	data := evt.Data().(StackRenamedData)
	stack, err := g.Stack(data.StackID)
	if err != nil {
		return
	}
	for i := range stack.Images {
		stack.Images[i].Name = data.Name
	}
	g.replace(stack.ID, stack)
}

// A Stack represents an image in a gallery. A Stack may have multiple variants
// of an image.
type Stack struct {
	ID     uuid.UUID
	Images []Image
}

// Image is an image of a Stack.
type Image struct {
	media.Image

	Original bool
	Size     string
}

// Original returns the original image in the Stack.
func (s Stack) Original() Image {
	for _, img := range s.Images {
		if img.Original {
			return img
		}
	}
	return Image{}
}

// WithTag adds the given tags to each Image in the Stack and returns the
// updated Stack. The original Stack is not modified.
func (s Stack) WithTag(tags ...string) Stack {
	s = s.copy()
	for i, img := range s.Images {
		s.Images[i].Image = img.WithTag(tags...)
	}
	return s
}

// WithoutTag removes the given tags from each Image and returns the updated
// Stack. The original Stack is not modified.
func (s Stack) WithoutTag(tags ...string) Stack {
	s = s.copy()
	for i, img := range s.Images {
		s.Images[i].Image = img.WithoutTag(tags...)
	}
	return s
}

func (s Stack) copy() Stack {
	images := make([]Image, len(s.Images))
	copy(images, s.Images)
	s.Images = images
	return s
}
