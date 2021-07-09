package gallery

import (
	"bytes"
	"context"
	"fmt"
	stdimage "image"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/modernice/cms/internal/concurrent"
	"github.com/modernice/cms/media"
	"github.com/modernice/cms/media/image"
	"github.com/modernice/goes/event"
)

// ProcessorContext is passed to Processors when they process a Stack. The
// ProcessorContext provides the Stack and image related functions or services
// to the Processors.
type ProcessorContext struct {
	context.Context

	stack   Stack
	encoder image.Encoder
	storage media.Storage
}

func newProcessorContext(
	ctx context.Context,
	stack Stack,
	imageEncoder image.Encoder,
	storage media.Storage,
) *ProcessorContext {
	if ctx == nil {
		ctx = context.Background()
	}
	return &ProcessorContext{
		Context: ctx,
		stack:   stack,
		encoder: imageEncoder,
		storage: storage,
	}
}

// Stack returns the processed Stack.
func (ctx *ProcessorContext) Stack() Stack {
	return ctx.stack
}

// Encoder returns the ImageEncoder.
func (ctx *ProcessorContext) Encoder() image.Encoder {
	return ctx.encoder
}

// Encode encodes an image using the appropriate encoder for the specified image format.
func (ctx *ProcessorContext) Encode(w io.Writer, img stdimage.Image, format string) error {
	return ctx.encoder.Encode(w, img, format)
}

// Storage returns the media storage.
func (ctx *ProcessorContext) Storage() media.Storage {
	return ctx.storage
}

// Update calls fn with the processed Stack and replaces the Stack with the one
// returned by fn.
//
// Is is illegal to update the UUID of the Stack. ErrStackCorrupted is returned
// in such cases.
func (ctx *ProcessorContext) Update(fn func(Stack) Stack) error {
	updated := fn(ctx.stack.copy())
	if updated.ID != ctx.stack.ID {
		return fmt.Errorf("illegal update of StackID: %w", ErrStackCorrupted)
	}
	ctx.stack = updated
	return nil
}

// A ProcessingPipeline is a collection of Processors that are run sequentially
// on a given Stack to post-process an image.
type ProcessingPipeline []Processor

// Process calls each Processor in the ProcessingPipeline with a ProcesingContext.
func (pipe ProcessingPipeline) Process(
	ctx context.Context,
	stack Stack,
	imageEncoder image.Encoder,
	storage media.Storage,
) (Stack, error) {
	pctx := newProcessorContext(ctx, stack, imageEncoder, storage)
	for i, proc := range pipe {
		if err := proc.Process(pctx); err != nil {
			return pctx.stack, fmt.Errorf("processor #%d failed: %w", i+1, err)
		}
	}
	return pctx.stack, nil
}

// A Processor processes an image through a ProcessorContext.
type Processor interface {
	Process(*ProcessorContext) error
}

// ProcessorFunc allows a function to be used as a Processor.
type ProcessorFunc func(*ProcessorContext) error

// Process calls fn(ctx).
func (fn ProcessorFunc) Process(ctx *ProcessorContext) error {
	return fn(ctx)
}

// A Resizer is a Processor that resizes the original Image of a Stack into
// additional dimensions.
type Resizer image.Resizer

// Process runs the Resizer on the Stack in the given ProcessorContext.
func (r Resizer) Process(ctx *ProcessorContext) error {
	s := ctx.Stack()
	org := s.Original()
	storage := ctx.Storage()

	original, format, err := org.Download(ctx, storage)
	if err != nil {
		return fmt.Errorf("download original image %q (%s): %w", org.Path, org.Disk, err)
	}

	resizer := (image.Resizer)(r)
	resized := resizer.Resize(original)

	encoded := make(map[string]*bytes.Buffer)

	for size, resizedImage := range resized {
		var buf bytes.Buffer
		if err := ctx.Encode(&buf, resizedImage, format); err != nil {
			return fmt.Errorf("encode %q resized image in %q format: %w", size, format, err)
		}
		encoded[size] = &buf
	}

	resizedImages := make([]Image, 0, len(encoded))

	for size, buf := range encoded {
		path := r.path(org.Path, size, format)

		img := media.NewImage(0, 0, org.Name, org.Disk, path, 0)
		img, err := img.Upload(ctx, buf, storage)
		if err != nil {
			return fmt.Errorf("upload %q (%s): %w", path, org.Disk, err)
		}

		resizedImages = append(resizedImages, Image{
			Image: img,
			Size:  size,
		})
	}

	sort.SliceStable(resizedImages, func(i, j int) bool {
		return resizedImages[i].Width <= resizedImages[j].Width
	})

	if err := ctx.Update(func(s Stack) Stack {
		s.Images = append(s.Images, resizedImages...)
		return s
	}); err != nil {
		return fmt.Errorf("update Stack: %w", err)
	}

	return nil
}

func (r Resizer) path(orgPath, size, format string) string {
	ext := filepath.Ext(orgPath)
	pathWithoutExt := strings.TrimSuffix(orgPath, ext)

	if ext == "" {
		ext = fmt.Sprintf(".%s", format)
	}

	return fmt.Sprintf("%s_%s%s", pathWithoutExt, size, ext)
}

// PNGCompressor is a Processor that compresses images using a png.Encoder with
// a png.CompressionLevel to compress the given image.
//
// PNGCompressor compresses each Image in a Stack in parallel.
type PNGCompressor image.PNGCompressor

// Process runs the PNGCompressor on the given ProcessorContext.
func (comp PNGCompressor) Process(ctx *ProcessorContext) error {
	c := image.PNGCompressor(comp)
	stack := ctx.Stack()
	storage := ctx.Storage()

	errs, fail := concurrent.Errors(ctx)

	var wg sync.WaitGroup
	for i, img := range stack.Images {
		wg.Add(1)
		go func(img Image, i int) {
			defer wg.Done()

			stdimg, format, err := img.Download(ctx, storage)
			if err != nil {
				fail(fmt.Errorf("download image %q (%s): %w", img.Path, img.Disk, err))
				return
			}

			compressed, err := c.Compress(stdimg)
			if err != nil {
				fail(fmt.Errorf("compress image with compression level %d: %w", c.CompressionLevel(), err))
				return
			}

			var buf bytes.Buffer
			if err := ctx.Encode(&buf, compressed, format); err != nil {
				fail(fmt.Errorf("encode %q image: %w", format, err))
				return
			}

			replaced, err := img.Replace(ctx, &buf, storage)
			if err != nil {
				fail(fmt.Errorf("replace image %q (%s): %w", img.Path, img.Disk, err))
				return
			}

			stack.Images[i].Image = replaced
		}(img, i)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errs:
		return err
	case <-done:
	}

	if err := ctx.Update(func(Stack) Stack { return stack }); err != nil {
		return fmt.Errorf("update Stack: %w", err)
	}

	return nil
}

// PostProcessor post-processed Stacks of Galleries.
type PostProcessor struct {
	encoder   image.Encoder
	storage   media.Storage
	galleries Repository
}

// NewPostProcessor returns a PostProcessor.
func NewPostProcessor(enc image.Encoder, storage media.Storage, galleries Repository) *PostProcessor {
	return &PostProcessor{
		encoder:   enc,
		storage:   storage,
		galleries: galleries,
	}
}

// Process calls pipe.Process with the provided Stack.
func (svc *PostProcessor) Process(ctx context.Context, stack Stack, pipe ProcessingPipeline) (Stack, error) {
	return pipe.Process(ctx, stack, svc.encoder, svc.storage)
}

// PostProcessorOption is an option for PostProcessor.Run.
type PostProcessorOption func(*postProcessorConfig)

type postProcessorConfig struct {
	workers     int
	onProcessed []func(Stack, *Gallery)
}

// ProcessorWorkers returns a PostProcessorOption that configures the worker
// count for processing Stacks. Default & minimum workers is 1.
func ProcessorWorkers(workers int) PostProcessorOption {
	return func(cfg *postProcessorConfig) {
		cfg.workers = workers
	}
}

// OnProcessed returns a PostProcessorOption that registers fn as a callback
// function that is called when a Stack has been processed.
func OnProcessed(fn func(Stack, *Gallery)) PostProcessorOption {
	return func(cfg *postProcessorConfig) {
		cfg.onProcessed = append(cfg.onProcessed, fn)
	}
}

// Run starts the PostProcessor in the background and returns a channel of
// asynchronous processing errors. PostProcessor runs until ctx is canceled.
func (svc *PostProcessor) Run(
	ctx context.Context,
	bus event.Bus,
	pipe ProcessingPipeline,
	opts ...PostProcessorOption,
) (<-chan error, error) {
	cfg := newProcessorConfig(opts...)

	events, errs, err := bus.Subscribe(ctx, ImageUploaded, ImageReplaced)
	if err != nil {
		return nil, fmt.Errorf("subscribe to %q event: %w", ImageUploaded, err)
	}

	queue := make(chan processorJob)
	out := make(chan error)

	go svc.work(ctx, cfg, queue, pipe, out)
	go svc.accept(ctx, queue, events, errs, out)

	return out, nil
}

type processorJob struct {
	galleryID uuid.UUID
	stackID   uuid.UUID
}

func (svc *PostProcessor) work(
	ctx context.Context,
	cfg postProcessorConfig,
	queue chan processorJob,
	pipe ProcessingPipeline,
	out chan<- error,
) {
	defer close(out)

	fail := func(err error) {
		select {
		case <-ctx.Done():
		case out <- err:
		}
	}

	var wg sync.WaitGroup
	wg.Add(cfg.workers)
	for i := 0; i < cfg.workers; i++ {
		go func() {
			defer wg.Done()
			for job := range queue {
				g, err := svc.galleries.Fetch(ctx, job.galleryID)
				if err != nil {
					fail(fmt.Errorf("fetch Gallery %q: %w", job.galleryID, err))
					continue
				}

				stack, err := g.Stack(job.stackID)
				if err != nil {
					fail(fmt.Errorf("get Stack %q: %w", job.stackID, err))
					continue
				}

				processed, err := pipe.Process(ctx, stack, svc.encoder, svc.storage)
				if err != nil {
					fail(fmt.Errorf("ProcessingPipeline failed: %w", err))
					continue
				}

				// Re-fetch Gallery to avoid concurrency errors if the processing took long.
				g, err = svc.galleries.Fetch(ctx, g.ID)
				if err != nil {
					fail(fmt.Errorf("re-fetch Gallery %q: %w", g.ID, err))
					continue
				}

				if err := g.Update(processed.ID, func(Stack) Stack { return processed }); err != nil {
					fail(fmt.Errorf("update Stack %q: %w", processed.ID, err))
					continue
				}

				if err := svc.galleries.Save(ctx, g); err != nil {
					fail(fmt.Errorf("save Gallery %q: %w", g.ID, err))
					continue
				}

				for _, fn := range cfg.onProcessed {
					fn(processed, g)
				}
			}
		}()
	}

	wg.Wait()
}

// listen for uploaded images and enqueue the processing jobs
func (svc *PostProcessor) accept(
	ctx context.Context,
	queue chan processorJob,
	events <-chan event.Event,
	errs <-chan error,
	out chan<- error,
) {
	defer close(queue)

	fail := func(err error) {
		select {
		case <-ctx.Done():
		case out <- err:
		}
	}

	event.ForEvery(
		ctx,
		func(evt event.Event) {
			switch data := evt.Data().(type) {
			case ImageUploadedData:
				enqueue(ctx, queue, evt.AggregateID(), data.Stack.ID)
			case ImageReplacedData:
				enqueue(ctx, queue, evt.AggregateID(), data.Stack.ID)
			}
		},
		fail,
		events, errs,
	)
}

func newProcessorConfig(opts ...PostProcessorOption) postProcessorConfig {
	var cfg postProcessorConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.workers < 1 {
		cfg.workers = 1
	}
	return cfg
}

func enqueue(ctx context.Context, queue chan<- processorJob, galleryID, stackID uuid.UUID) {
	select {
	case <-ctx.Done():
	case queue <- processorJob{
		galleryID: galleryID,
		stackID:   stackID,
	}:
	}
}
