package gallery

//go:generate mockgen -source=gallery.go -destination=./mock_gallery/gallery.go

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/modernice/goes/aggregate"
	"github.com/modernice/goes/event"
	"github.com/modernice/nice-cms/internal/concurrent"
	"github.com/modernice/nice-cms/internal/unique"
	"github.com/modernice/nice-cms/media"
)

// Aggregate is the name of the Gallery aggregate.
const Aggregate = "cms.media.image.gallery"

var (
	// ErrEmptyName is returned when an empty name is provided to a Gallery.
	ErrEmptyName = errors.New("empty name")

	// ErrAlreadyCreated is returned when creating a Gallery that was created already.
	ErrAlreadyCreated = errors.New("already created")

	// ErrNotCreated is returned when trying to use a Gallery that hasn't been created yet.
	ErrNotCreated = errors.New("Gallery not created")

	// ErrNotFound is returned when a Gallery cannot be found within a Repository.
	ErrNotFound = errors.New("Gallery not found")

	// ErrStackNotFound is returned when a Stack cannot be found within a Gallery.
	ErrStackNotFound = errors.New("Stack not found")

	// ErrStackCorrupted is returned updating a Stack in an illegal way.
	ErrStackCorrupted = errors.New("stack corrupted")
)

// Repository handles persistence of Galleries.
type Repository interface {
	// Save saves a Gallery.
	Save(context.Context, *Gallery) error

	// Fetch fetches the Gallery with the given UUID or ErrNotFound.
	Fetch(context.Context, uuid.UUID) (*Gallery, error)

	// Delete deletes a Gallery.
	Delete(context.Context, *Gallery) error

	// Use fetches the Gallery with the given UUID, calls the provided function
	// with that Gallery and saves the Gallery into the repository. If the
	// function returns a non-nil error, the Gallery is not saved and that error
	// is returned.
	Use(context.Context, uuid.UUID, func(*Gallery) error) error
}

// A Gallery is a collection of image stacks. A Stack may contain multiple
// variants of the same image in different sizes.
type Gallery struct {
	*aggregate.Base
	*Implementation
}

// Implementation can be embedded into structs to implement a Gallery.
type Implementation struct {
	Name   string  `json:"name"`
	Stacks []Stack `json:"stacks"`

	gallery aggregate.Aggregate
}

// New returns a new Gallery.
func New(id uuid.UUID) *Gallery {
	g := &Gallery{Base: aggregate.New(Aggregate, id)}
	g.Implementation = NewImplementation(g)
	return g
}

// NewImplementation returns the implementation for the provided Gallery.
func NewImplementation(gallery aggregate.Aggregate) *Implementation {
	return &Implementation{
		Stacks:  make([]Stack, 0),
		gallery: gallery,
	}
}

// Stack returns the Stack with the given UUID or ErrStackNotFound.
func (g *Implementation) Stack(id uuid.UUID) (Stack, error) {
	for _, stack := range g.Stacks {
		if stack.ID == id {
			return stack.copy(), nil
		}
	}
	return Stack{}, ErrStackNotFound
}

// FindByTag returns the Stacks that have all provided tags.
func (g *Implementation) FindByTag(tags ...string) []Stack {
	out := make([]Stack, 0)
	for _, s := range g.Stacks {
		if s.Original().HasTag(tags...) {
			out = append(out, s)
		}
	}
	return out
}

// ApplyEvent applies aggregate events.
func (g *Gallery) ApplyEvent(evt event.Event) {
	switch evt.Name() {
	case Created:
		g.create(evt)
	case ImageUploaded:
		g.uploadImage(evt)
	case ImageReplaced:
		g.replaceImage(evt)
	case StackDeleted:
		g.deleteStack(evt)
	case StackTagged:
		g.tagStack(evt)
	case StackUntagged:
		g.untagStack(evt)
	case StackRenamed:
		g.renameStack(evt)
	case StackUpdated:
		g.updateStack(evt)
	}
}

// Create creates the Gallery with the given name.
func (g *Implementation) Create(name string) error {
	if g.Name != "" {
		return ErrAlreadyCreated
	}
	if name = strings.TrimSpace(name); name == "" {
		return ErrEmptyName
	}
	aggregate.NextEvent(g.gallery, Created, CreatedData{Name: name})
	return nil
}

func (g *Implementation) create(evt event.Event) {
	data := evt.Data().(CreatedData)
	g.Name = data.Name
}

// Upload uploads the image in r to storage and returns the Stack for that image.
func (g *Implementation) Upload(ctx context.Context, storage media.Storage, r io.Reader, name, diskName, path string) (Stack, error) {
	stack, err := g.uploadWithID(ctx, storage, r, name, diskName, path, uuid.New())
	if err != nil {
		return stack, err
	}

	aggregate.NextEvent(g.gallery, ImageUploaded, ImageUploadedData{Stack: stack})

	return stack, nil
}

func (g *Implementation) uploadWithID(
	ctx context.Context,
	storage media.Storage,
	r io.Reader,
	name, diskName, path string,
	id uuid.UUID,
) (Stack, error) {
	if err := g.checkCreated(); err != nil {
		return Stack{}, err
	}

	img := media.NewImage(0, 0, name, diskName, path, 0)

	var err error
	if img, err = img.Upload(ctx, r, storage); err != nil {
		return Stack{}, fmt.Errorf("upload to %q storage: %w", diskName, err)
	}

	stack := Stack{
		ID:     id,
		Images: []Image{{Image: img, Original: true}},
	}

	return stack, nil
}

func (g *Implementation) checkCreated() error {
	if g.Name == "" {
		return ErrNotCreated
	}
	return nil
}

func (g *Implementation) uploadImage(evt event.Event) {
	data := evt.Data().(ImageUploadedData)
	g.Stacks = append(g.Stacks, data.Stack)
}

// Replace replaced the Images in the given Stack with the image in r.
func (g *Implementation) Replace(ctx context.Context, storage media.Storage, r io.Reader, stackID uuid.UUID) (Stack, error) {
	stack, err := g.Stack(stackID)
	if err != nil {
		return stack, err
	}

	org := stack.Original()
	if org.Path == "" {
		return stack, ErrStackCorrupted
	}

	replaced, err := g.uploadWithID(ctx, storage, r, org.Name, org.Disk, org.Path, stack.ID)
	if err != nil {
		return stack, fmt.Errorf("upload image: %w", err)
	}

	aggregate.NextEvent(g.gallery, ImageReplaced, ImageReplacedData{Stack: replaced})

	return g.Stack(replaced.ID)
}

func (g *Implementation) replaceImage(evt event.Event) {
	data := evt.Data().(ImageReplacedData)
	g.replace(data.Stack.ID, data.Stack)
}

// Delete deletes the given Stack from the Gallery and Storage.
func (g *Implementation) Delete(ctx context.Context, storage media.Storage, stack Stack) error {
	if err := g.checkCreated(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(len(stack.Images))
	for _, img := range stack.Images {
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
		aggregate.NextEvent(g.gallery, StackDeleted, StackDeletedData{Stack: stack})
		return nil
	}
}

func (g *Implementation) deleteStack(evt event.Event) {
	data := evt.Data().(StackDeletedData)
	g.remove(data.Stack.ID)
}

func (g *Implementation) remove(id uuid.UUID) {
	for i, stack := range g.Stacks {
		if stack.ID == id {
			g.Stacks = append(g.Stacks[:i], g.Stacks[i+1:]...)
			return
		}
	}
}

// Tag tags each Image in the provided Stack through the ImageService.
func (g *Implementation) Tag(ctx context.Context, stack Stack, tags ...string) (Stack, error) {
	if err := g.checkCreated(); err != nil {
		return Stack{}, err
	}
	tags = unique.Strings(tags...)
	aggregate.NextEvent(g.gallery, StackTagged, StackTaggedData{
		StackID: stack.ID,
		Tags:    tags,
	})
	return g.Stack(stack.ID)
}

func (g *Implementation) tagStack(evt event.Event) {
	data := evt.Data().(StackTaggedData)
	stack, err := g.Stack(data.StackID)
	if err != nil {
		return
	}
	stack = stack.WithTag(data.Tags...)
	g.replace(stack.ID, stack)
}

func (g *Implementation) replace(id uuid.UUID, stack Stack) error {
	for i, s := range g.Stacks {
		if s.ID != id {
			continue
		}

		if stack.ID != s.ID {
			return fmt.Errorf("illegal StackID update: %w", ErrStackCorrupted)
		}

		g.Stacks[i] = stack
		return nil
	}

	return ErrStackNotFound
}

// Untag removes tags from each Image of the provided Stack.
func (g *Implementation) Untag(ctx context.Context, stack Stack, tags ...string) (Stack, error) {
	if err := g.checkCreated(); err != nil {
		return Stack{}, err
	}
	tags = unique.Strings(tags...)
	aggregate.NextEvent(g.gallery, StackUntagged, StackUntaggedData{
		StackID: stack.ID,
		Tags:    tags,
	})
	return g.Stack(stack.ID)
}

func (g *Implementation) untagStack(evt event.Event) {
	data := evt.Data().(StackUntaggedData)
	stack, err := g.Stack(data.StackID)
	if err != nil {
		return
	}
	stack = stack.WithoutTag(data.Tags...)
	g.replace(stack.ID, stack)
}

// RenameStack renames each Image in the given Stack to name.
func (g *Implementation) RenameStack(ctx context.Context, stackID uuid.UUID, name string) (Stack, error) {
	if err := g.checkCreated(); err != nil {
		return Stack{}, err
	}

	stack, err := g.Stack(stackID)
	if err != nil {
		return Stack{}, err
	}

	aggregate.NextEvent(g.gallery, StackRenamed, StackRenamedData{
		StackID: stack.ID,
		OldName: stack.Original().Name,
		Name:    name,
	})

	return g.Stack(stack.ID)
}

func (g *Implementation) renameStack(evt event.Event) {
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

// Update updates the Stack with the given UUID by calling update with the
// current Stack and replacing that Stack with the one returned by update.
//
// It is illegal to update the UUID of a Stack. In such cases ErrStackCorrupted
// is returned.
func (g *Implementation) Update(id uuid.UUID, update func(Stack) Stack) error {
	stack, err := g.Stack(id)
	if err != nil {
		return err
	}

	stack = update(stack)
	aggregate.NextEvent(g.gallery, StackUpdated, StackUpdatedData{Stack: stack})

	return nil
}

func (g *Implementation) updateStack(evt event.Event) {
	data := evt.Data().(StackUpdatedData)
	g.replace(data.Stack.ID, data.Stack)
}

type snapshot struct {
	Stacks []Stack `json:"stacks"`
}

// MarshalSnapshot implements snapshot.Marshaler.
func (g *Implementation) MarshalSnapshot() ([]byte, error) {
	return json.Marshal(snapshot{Stacks: g.Stacks})
}

// UnmarshalSnapshot implements snapshot.Unmarshaler.
func (g *Implementation) UnmarshalSnapshot(b []byte) error {
	var snap snapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		return err
	}
	g.Stacks = snap.Stacks
	if g.Stacks == nil {
		g.Stacks = make([]Stack, 0)
	}
	return nil
}

// A Stack represents an image in a gallery. A Stack may have multiple variants
// of an image.
type Stack struct {
	ID     uuid.UUID `json:"id"`
	Images []Image   `json:"images"`
}

// Image is an image of a Stack.
type Image struct {
	media.Image

	Original bool   `json:"original"`
	Size     string `json:"size"`
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

type goesRepository struct {
	repo aggregate.Repository
}

// GoesRepository returns a Repository that uses an aggregate.Repository under
// the hood.
func GoesRepository(repo aggregate.Repository) Repository {
	return &goesRepository{repo: repo}
}

func (r *goesRepository) Save(ctx context.Context, g *Gallery) error {
	return r.repo.Save(ctx, g)
}

func (r *goesRepository) Fetch(ctx context.Context, id uuid.UUID) (*Gallery, error) {
	g := New(id)
	if err := r.repo.Fetch(ctx, g); err != nil {
		return nil, err
	}
	return g, nil
}

func (r *goesRepository) Delete(ctx context.Context, g *Gallery) error {
	return r.repo.Delete(ctx, g)
}

func (r *goesRepository) Use(ctx context.Context, id uuid.UUID, fn func(*Gallery) error) error {
	g, err := r.Fetch(ctx, id)
	if err != nil {
		return fmt.Errorf("fetch gallery: %w", err)
	}
	if err := fn(g); err != nil {
		return err
	}
	if err := r.Save(ctx, g); err != nil {
		return fmt.Errorf("save gallery: %w", err)
	}
	return nil
}
