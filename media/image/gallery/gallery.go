package gallery

//go:generate mockgen -source=gallery.go -destination=./mock_gallery/gallery.go

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/modernice/cms/internal/concurrent"
	"github.com/modernice/cms/media"
	"github.com/modernice/cms/media/image"
)

var (
	// ErrStackNotFound is returned when a Stack doesn't exist in a StackRepository.
	ErrStackNotFound = errors.New("stack not found")
)

// A StackRepository contains the Stacks of galleries.
type StackRepository interface {
	// Save saves a Stack.
	Save(context.Context, Stack) error

	// Get returns the Stack with the given UUID or ErrStackNotFound if the
	// Stack does not exist in the StackRepository.
	Get(context.Context, uuid.UUID) (Stack, error)

	// Delete deletes a Stack from the StackRepository. It does not delete the
	// actual file from the storage.
	Delete(context.Context, Stack) error
}

// A Stack represents an image in a gallery. A Stack may have multiple variants
// of an image.
type Stack struct {
	ID      uuid.UUID
	Gallery string
	Images  []Image
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

// Image is an image of a Stack.
type Image struct {
	media.Image

	Original bool
	Size     string
}

// Gallery provides gallery-based image management.
type Gallery struct {
	Name string

	stacks       StackRepository
	images       image.Repository
	imageService media.ImageService
	enc          media.ImageEncoder

	pipeline          ProcessingPipeline
	processingTimeout time.Duration
}

// Option is a Gallery option.
type Option func(*Gallery)

// WithProcessing returns an Option that adds a ProcessingPipeline to a Gallery.
// Stacks of uploaded images are passed to the pipeline for post-processing.
// The provided Duration is the timeout for processing of a single Stack. A zero
// Duration means no timeout.
func WithProcessing(pipeline ProcessingPipeline, timeout time.Duration) Option {
	return func(g *Gallery) {
		g.pipeline = pipeline
		g.processingTimeout = timeout
	}
}

// New initializes a Gallery with the given name.
func New(
	name string,
	stacks StackRepository,
	images image.Repository,
	imageService media.ImageService,
	enc media.ImageEncoder,
	opts ...Option,
) *Gallery {
	g := Gallery{
		Name:         name,
		stacks:       stacks,
		images:       images,
		imageService: imageService,
		enc:          enc,
	}
	for _, opt := range opts {
		opt(&g)
	}
	return &g
}

// Upload uploads the image in r through the ImageService and returns the
// created Stack and a channel that receives the processing error or nil as soon
// as processing of the Stack is done. If no ProcessingPipeline was provided
// when creating the Gallery, the channel immediately receives a nil-error. The
// channel has a buffer size of 1, so it is not necessary to receive from it.
//
//	var img io.Reader
//	g := gallery.New("foo", ...)
//	stack, processed, err := g.Upload(context.TODO(), img, "Example image", "s3", "/example/example.png")
//	// handle err
//	err = <-processed
//	if err != nil {
//		log.Printf("processing failed: %v", err)
//	}
//	err = g.Refresh(context.TODO(), &stack) // fetch the processed Stack
//	// handle err
func (g *Gallery) Upload(ctx context.Context, r io.Reader, name, disk, path string) (Stack, <-chan error, error) {
	img, err := g.imageService.Upload(ctx, r, name, disk, path)
	if err != nil {
		return Stack{}, nil, fmt.Errorf("upload to storage: %w", err)
	}

	stack := Stack{
		ID:      uuid.New(),
		Gallery: g.Name,
		Images:  []Image{{Image: img, Original: true}},
	}

	for _, img := range stack.Images {
		if err := g.images.Save(ctx, img.Image); err != nil {
			return stack, nil, fmt.Errorf("save Image of Stack: %w", err)
		}
	}

	if err := g.stacks.Save(ctx, stack); err != nil {
		return stack, nil, fmt.Errorf("save Stack: %w", err)
	}

	processed := make(chan error, 1)
	go g.process(stack, processed)

	return stack, processed, nil
}

func (g *Gallery) process(stack Stack, out chan<- error) {
	ctx := context.Background()

	if g.processingTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), g.processingTimeout)
		defer cancel()
	}

	fail := func(err error) {
		select {
		case <-ctx.Done():
		case out <- err:
		}
	}

	processed, err := g.pipeline.Process(ctx, stack, g.imageService, g.enc)
	if err != nil {
		fail(fmt.Errorf("ProcessingPipeline: %w", err))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := g.stacks.Save(ctx, processed); err != nil {
		fail(fmt.Errorf("save processed Stack: %w", err))
		return
	}

	select {
	case <-ctx.Done():
	case out <- nil:
	}
}

// Stack returns the Stack with the given UUID or ErrStackNotFound if the Stack
// does not exist in the StackRepository.
func (g *Gallery) Stack(ctx context.Context, id uuid.UUID) (Stack, error) {
	return g.stacks.Get(ctx, id)
}

// Refresh fetches the given Stack from the StackRepository and updates the
// provided Stack with the fetched one.
func (g *Gallery) Refresh(ctx context.Context, stack *Stack) error {
	if stack == nil {
		return nil
	}

	s, err := g.stacks.Get(ctx, stack.ID)
	if err != nil {
		return fmt.Errorf("fetch Stack %q: %w", stack.ID, err)
	}

	*stack = s

	return nil
}

// Delete deletes a Stack by first deleting each Image in the Stack through the
// ImageService and then deleting the Stack from the StackRepository.
//
// Delete does not return an error if the ImageRepository fails to delete an
// Image, because a dangling file does not do as much harm as a corrupted Stack.
func (g *Gallery) Delete(ctx context.Context, stack Stack) error {
	var wg sync.WaitGroup
	for _, img := range stack.Images {
		wg.Add(1)
		go func(img Image) {
			defer wg.Done()
			// TODO: report error (?)
			g.imageService.Delete(ctx, img.Disk, img.Path)
		}(img)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-concurrent.WhenDone(&wg):
	}

	if err := g.stacks.Delete(ctx, stack); err != nil {
		return fmt.Errorf("delete from StackRepository: %w", err)
	}

	return nil
}

// Tag tags each Image in the provided Stack through the ImageService.
func (g *Gallery) Tag(ctx context.Context, stack Stack, tags ...string) (Stack, error) {
	images := make([]Image, len(stack.Images))
	copy(images, stack.Images)

	for i, img := range images {
		tagged, err := g.imageService.Tag(ctx, img.Disk, img.Path, tags...)
		if err != nil {
			return stack, fmt.Errorf("tag %q (%s) with tags %v: %w", img.Path, img.Disk, tags, err)
		}
		images[i].Image = tagged
	}
	stack.Images = images

	if err := g.stacks.Save(ctx, stack); err != nil {
		return stack, fmt.Errorf("save Stack: %w", err)
	}

	return stack, nil
}

// Untag removes tags from each Image of the provided Stack.
func (g *Gallery) Untag(ctx context.Context, stack Stack, tags ...string) (Stack, error) {
	images := make([]Image, len(stack.Images))
	copy(images, stack.Images)

	for i, img := range images {
		tagged, err := g.imageService.Untag(ctx, img.Disk, img.Path, tags...)
		if err != nil {
			return stack, fmt.Errorf("remove tags %v from %q (%s): %w", tags, img.Path, img.Disk, err)
		}
		images[i].Image = tagged
	}
	stack.Images = images

	if err := g.stacks.Save(ctx, stack); err != nil {
		return stack, fmt.Errorf("save Stack: %w", err)
	}

	return stack, nil
}

type stackRepository struct {
	mux    sync.RWMutex
	stacks map[uuid.UUID]Stack
}

// InMemoryStackRepository returns an in-memory StackRepository.
func InMemoryStackRepository() StackRepository {
	return &stackRepository{
		stacks: make(map[uuid.UUID]Stack),
	}
}

func (r *stackRepository) Save(_ context.Context, s Stack) error {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.stacks[s.ID] = s
	return nil
}

func (r *stackRepository) Get(_ context.Context, id uuid.UUID) (Stack, error) {
	r.mux.RLock()
	defer r.mux.RUnlock()
	if stack, ok := r.stacks[id]; ok {
		return stack, nil
	}
	return Stack{}, ErrStackNotFound
}

func (r *stackRepository) Delete(_ context.Context, s Stack) error {
	r.mux.Lock()
	defer r.mux.Unlock()
	delete(r.stacks, s.ID)
	return nil
}
