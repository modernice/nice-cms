package gallery

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
	ErrStackNotFound = errors.New("Stack not found")
)

type Gallery struct {
	Name   string
	Stacks []Stack
}

func New(name string) *Gallery {
	return &Gallery{
		Name:   name,
		Stacks: make([]Stack, 0),
	}
}

func (g *Gallery) Stack(id uuid.UUID) (Stack, error) {
	for _, stack := range g.Stacks {
		if stack.ID == id {
			return stack, nil
		}
	}
	return Stack{}, ErrStackNotFound
}

func (g *Gallery) Refresh(stack *Stack) error {
	if stack == nil {
		return nil
	}

	s, err := g.Stack(stack.ID)
	if err != nil {
		return err
	}
	*stack = s

	return nil
}

type UploadOption func(*uploadConfig)

type uploadConfig struct {
	pipeline          ProcessingPipeline
	processingTimeout time.Duration
}

func ProcessStack(pipe ProcessingPipeline, timeout time.Duration) UploadOption {
	return func(cfg *uploadConfig) {
		cfg.pipeline = pipe
		cfg.processingTimeout = timeout
	}
}

func (g *Gallery) Upload(
	ctx context.Context,
	storage media.Storage,
	images image.Repository,
	imageService media.ImageService,
	enc media.ImageEncoder,
	r io.Reader,
	name, disk, path string,
	opts ...UploadOption,
) (Stack, <-chan error, error) {
	var cfg uploadConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	img, err := imageService.Upload(ctx, r, name, disk, path)
	if err != nil {
		return Stack{}, nil, fmt.Errorf("upload to storage: %w", err)
	}

	stack := Stack{
		ID:      uuid.New(),
		Gallery: g.Name,
		Images:  []Image{{Image: img, Original: true}},
	}

	for _, img := range stack.Images {
		if err := images.Save(ctx, img.Image); err != nil {
			return stack, nil, fmt.Errorf("save Image of Stack: %w", err)
		}
	}

	g.Stacks = append(g.Stacks, stack)

	processed := make(chan error, 1)
	go g.process(imageService, enc, cfg, stack, processed)

	return stack, processed, nil
}

func (g *Gallery) process(imageService media.ImageService, enc media.ImageEncoder, cfg uploadConfig, stack Stack, out chan<- error) {
	ctx := context.Background()
	if cfg.processingTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.processingTimeout)
		defer cancel()
	}

	fail := func(err error) {
		select {
		case <-ctx.Done():
		case out <- err:
		}
	}

	processed, err := cfg.pipeline.Process(ctx, stack, imageService, enc)
	if err != nil {
		fail(fmt.Errorf("ProcessingPipeline: %w", err))
		return
	}

	g.replace(processed.ID, processed)

	select {
	case <-ctx.Done():
	case out <- nil:
	}
}

func (g *Gallery) replace(id uuid.UUID, replacement Stack) {
	for i, stack := range g.Stacks {
		if stack.ID == id {
			g.Stacks[i] = replacement
			return
		}
	}
}

func (g *Gallery) Delete(ctx context.Context, imageService media.ImageService, stack Stack) error {
	var wg sync.WaitGroup
	for _, img := range stack.Images {
		wg.Add(1)
		go func(img Image) {
			defer wg.Done()
			// TODO: report error (?)
			imageService.Delete(ctx, img.Disk, img.Path)
		}(img)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-concurrent.WhenDone(&wg):
		g.remove(stack.ID)
		return nil
	}
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
func (g *Gallery) Tag(ctx context.Context, imageService media.ImageService, stack Stack, tags ...string) (Stack, error) {
	images := make([]Image, len(stack.Images))
	copy(images, stack.Images)

	for i, img := range images {
		tagged, err := imageService.Tag(ctx, img.Disk, img.Path, tags...)
		if err != nil {
			return stack, fmt.Errorf("tag %q (%s) with tags %v: %w", img.Path, img.Disk, tags, err)
		}
		images[i].Image = tagged
	}
	stack.Images = images

	return stack, nil
}

// Untag removes tags from each Image of the provided Stack.
func (g *Gallery) Untag(ctx context.Context, imageService media.ImageService, stack Stack, tags ...string) (Stack, error) {
	images := make([]Image, len(stack.Images))
	copy(images, stack.Images)

	for i, img := range images {
		tagged, err := imageService.Untag(ctx, img.Disk, img.Path, tags...)
		if err != nil {
			return stack, fmt.Errorf("remove tags %v from %q (%s): %w", tags, img.Path, img.Disk, err)
		}
		images[i].Image = tagged
	}
	stack.Images = images

	return stack, nil
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
