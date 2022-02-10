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
	"time"

	"github.com/google/uuid"
	"github.com/modernice/goes/event"
	"github.com/modernice/nice-cms/internal/concurrent"
	"github.com/modernice/nice-cms/media"
	"github.com/modernice/nice-cms/media/image"
)

// ProcessorContext is passed to Processors when they process a Stack. The
// ProcessorContext provides the Stack and image related functions or services
// to the Processors.
type ProcessorContext struct {
	context.Context

	cfg     processorConfig
	stack   Stack
	encoder image.Encoder
	storage media.Storage
}

// Printer is the interface for a logger.
type Printer interface {
	Print(v ...interface{})
}

func newProcessorContext(
	ctx context.Context,
	cfg processorConfig,
	stack Stack,
	imageEncoder image.Encoder,
	storage media.Storage,
) *ProcessorContext {
	if ctx == nil {
		ctx = context.Background()
	}
	return &ProcessorContext{
		Context: ctx,
		cfg:     cfg,
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

type ProcessorOption func(*processorConfig)

type processorConfig struct {
	logger Printer
}

func WithDebugger(logger Printer) ProcessorOption {
	return func(cfg *processorConfig) {
		cfg.logger = logger
	}
}

// Process calls each Processor in the ProcessingPipeline with a ProcesingContext.
func (pipe ProcessingPipeline) Process(
	ctx context.Context,
	stack Stack,
	imageEncoder image.Encoder,
	storage media.Storage,
	opts ...ProcessorOption,
) (Stack, error) {
	var cfg processorConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	pctx := newProcessorContext(ctx, cfg, stack, imageEncoder, storage)

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

	ctx.cfg.logf("[Resizer] Resize image (StackID=%v Sizes=%v)", s.ID, resizer)
	start := time.Now()
	resized := resizer.Resize(original)
	ctx.cfg.logf("[Resizer] Resize done (StackID=%v Duration=%v)", s.ID, time.Since(start))

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

		ctx.cfg.logf("[Resizer] Upload resized image (StackID=%v Size=%v)", s.ID, size)
		start := time.Now()
		img, err := img.Upload(ctx, buf, storage)
		if err != nil {
			return fmt.Errorf("upload %q (%s): %w", path, org.Disk, err)
		}
		ctx.cfg.logf("[Resizer] Upload done (StackID=%v Duration=%v)", s.ID, time.Since(start))

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

			ctx.cfg.logf("[PNGCompressor]: Compress image (StackID=%v Disk=%v Path=%v)", stack.ID, img.Disk, img.Path)
			start := time.Now()
			compressed, err := c.Compress(stdimg)
			if err != nil {
				fail(fmt.Errorf("compress image with compression level %d: %w", c.CompressionLevel(), err))
				return
			}
			ctx.cfg.logf("[PNGCompressor]: Image compressed (StackID=%v Disk=%v Path=%v Duration=%v)", stack.ID, img.Disk, img.Path, time.Since(start))

			var buf bytes.Buffer
			if err := ctx.Encode(&buf, compressed, format); err != nil {
				fail(fmt.Errorf("encode %q image: %w", format, err))
				return
			}

			ctx.cfg.logf("[PNGCompressor]: Replace storage image (StackID=%v Disk=%v Path=%v)", stack.ID, img.Disk, img.Path)
			start = time.Now()
			replaced, err := img.Replace(ctx, &buf, storage)
			if err != nil {
				fail(fmt.Errorf("replace image %q (%s): %w", img.Path, img.Disk, err))
				return
			}
			ctx.cfg.logf("[PNGCompressor]: Storage image replaced (StackID=%v Disk=%v Path=%v Duration=%v)", stack.ID, img.Disk, img.Path, time.Since(start))

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
func (svc *PostProcessor) Process(ctx context.Context, stack Stack, pipe ProcessingPipeline, opts ...ProcessorOption) (Stack, error) {
	return pipe.Process(ctx, stack, svc.encoder, svc.storage, opts...)
}

// PostProcessorOption is an option for PostProcessor.Run.
type PostProcessorOption func(*postProcessorConfig)

type postProcessorConfig struct {
	logger      Printer
	workers     int
	onProcessed []func(Stack, *Gallery)
}

// ProcessorLogger returns a PostProcessorOption that provides the post-processor
// with a logger.
func ProcessorLogger(logger Printer) PostProcessorOption {
	return func(cfg *postProcessorConfig) {
		cfg.logger = logger
	}
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

	cfg.log("Logging enabled.")

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

	cfg.logf("Post-processor running with %d worker(s).", cfg.workers)

	var wg sync.WaitGroup
	wg.Add(cfg.workers)

	for i := 0; i < cfg.workers; i++ {
		go func() {
			defer wg.Done()
			for job := range queue {
				cfg.logf("Received processing job (GalleryID=%v StackID=%v)", job.galleryID, job.stackID)

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

				cfg.logf("Processing stack (ID=%v)", stack.ID)
				start := time.Now()

				processed, err := svc.Process(ctx, stack, pipe, WithDebugger(cfg.logger))
				if err != nil {
					fail(fmt.Errorf("ProcessingPipeline failed: %w", err))
					continue
				}

				cfg.logf("Processing done (StackID=%v Duration=%v)", stack.ID, time.Since(start))

				if err := svc.galleries.Use(ctx, g.ID, func(gal *Gallery) error {
					g = gal
					if err := g.Update(processed.ID, func(Stack) Stack { return processed }); err != nil {
						return fmt.Errorf("update stack: %w [id=%v]", err, processed.ID)
					}
					return nil
				}); err != nil {
					fail(fmt.Errorf("update gallery: %w", err))
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
			id, _, _ := evt.Aggregate()
			switch data := evt.Data().(type) {
			case ImageUploadedData:
				enqueue(ctx, queue, id, data.Stack.ID)
			case ImageReplacedData:
				enqueue(ctx, queue, id, data.Stack.ID)
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

func (cfg postProcessorConfig) log(v ...interface{}) {
	if cfg.logger != nil {
		cfg.logger.Print(v...)
	}
}

func (cfg postProcessorConfig) logf(format string, v ...interface{}) {
	if cfg.logger != nil {
		cfg.logger.Print(fmt.Sprintf(format, v...))
	}
}

func (cfg processorConfig) log(v ...interface{}) {
	if cfg.logger != nil {
		cfg.logger.Print(v...)
	}
}

func (cfg processorConfig) logf(format string, v ...interface{}) {
	if cfg.logger != nil {
		cfg.logger.Print(fmt.Sprintf(format, v...))
	}
}
