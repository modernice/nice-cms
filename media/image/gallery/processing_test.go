package gallery_test

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/modernice/goes/aggregate/repository"
	"github.com/modernice/goes/event/eventbus/chanbus"
	"github.com/modernice/goes/event/eventstore"
	"github.com/modernice/goes/event/eventstore/memstore"
	"github.com/modernice/nice-cms/internal/imggen"
	"github.com/modernice/nice-cms/media"
	"github.com/modernice/nice-cms/media/image"
	"github.com/modernice/nice-cms/media/image/gallery"
)

func TestProcessingPipeline_Process(t *testing.T) {
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))
	enc := image.NewEncoder()
	g := gallery.New(uuid.New())
	g.Create("foo")

	pipe := gallery.ProcessingPipeline{
		gallery.Resizer{
			"thumb": {Width: 240},
			"small": {Width: 640},
		},
		gallery.Resizer{
			"medium": {Width: 1280},
			"large":  {Width: 1920},
		},
	}

	_, buf := imggen.ColoredRectangle(800, 600, color.RGBA{100, 100, 100, 0xff})
	filesize := len(buf.Bytes())

	if _, err := g.Upload(context.Background(), storage, buf, exampleName, exampleDisk, examplePath); err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	stack := gallery.Stack{
		ID: uuid.New(),
		Images: []gallery.Image{
			{
				Image:    media.NewImage(800, 600, exampleName, exampleDisk, examplePath, filesize),
				Original: true,
			},
		},
	}

	processed, err := pipe.Process(context.Background(), stack, enc, storage)
	if err != nil {
		t.Fatalf("ProcessingPipeline failed to process Stack: %v", err)
	}

	if processed.ID != stack.ID {
		t.Fatalf("processed Stack should keep UUID %q; has %q", stack.ID, processed.ID)
	}

	if len(processed.Images) != 5 {
		t.Fatalf("processed Stack should contain 5 images; contains %d", len(processed.Images))
	}

	if !reflect.DeepEqual(processed.Images[0], stack.Images[0]) {
		t.Fatalf("first image of processed Stack should be the original")
	}

	sizeTests := []struct {
		size      string
		wantWidth int
	}{
		{
			size:      "",
			wantWidth: 800,
		},
		{
			size:      "thumb",
			wantWidth: 240,
		},
		{
			size:      "small",
			wantWidth: 640,
		},
		{
			size:      "medium",
			wantWidth: 1280,
		},
		{
			size:      "large",
			wantWidth: 1920,
		},
	}

	for _, tt := range sizeTests {
		t.Run(tt.size, func(t *testing.T) {
			var found bool
			var foundImage gallery.Image
			for _, img := range processed.Images {
				if img.Size == tt.size {
					found = true
					foundImage = img
					break
				}
			}

			if !found {
				t.Fatalf("Stack should contain image with %q size", tt.size)
			}

			if foundImage.Width != tt.wantWidth {
				t.Fatalf("Image in ImageRepository should have width of %d; has %d", tt.wantWidth, foundImage.Width)
			}

			wantPath := examplePath
			if tt.size != "" {
				pathWithoutExt := strings.TrimSuffix(examplePath, filepath.Ext(examplePath))
				wantPath = fmt.Sprintf("%s_%s.png", pathWithoutExt, tt.size)
			}

			if foundImage.Path != wantPath {
				t.Fatalf("Image path should be %q; is %q", wantPath, foundImage.Path)
			}
		})
	}
}

func TestProcessingPipeline_Process_illegalStackIDUpdate(t *testing.T) {
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))

	pipe := gallery.ProcessingPipeline{
		gallery.ProcessorFunc(func(ctx *gallery.ProcessorContext) error {
			return ctx.Update(func(s gallery.Stack) gallery.Stack {
				s.ID = uuid.New()
				return s
			})
		}),
	}

	_, buf := imggen.ColoredRectangle(800, 600, color.RGBA{100, 100, 100, 0xff})
	filesize := len(buf.Bytes())

	stack := gallery.Stack{
		ID: uuid.New(),
		Images: []gallery.Image{
			{
				Image:    media.NewImage(800, 600, exampleName, exampleDisk, examplePath, filesize),
				Original: true,
			},
		},
	}

	enc := image.NewEncoder()

	_, err := pipe.Process(context.Background(), stack, enc, storage)
	if !errors.Is(err, gallery.ErrStackCorrupted) {
		t.Fatalf("ProcessingPipeline should fail with %q when trying to update the UUID of a Stack; got %q", gallery.ErrStackCorrupted, err)
	}
}

func TestPostProcessor_Process(t *testing.T) {
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))
	enc := image.NewEncoder()
	estore := memstore.New()
	aggregates := repository.New(estore)
	galleries := gallery.GoesRepository(aggregates)

	svc := gallery.NewPostProcessor(enc, storage, galleries)

	g := gallery.New(uuid.New())
	g.Create("foo")

	_, buf := imggen.ColoredRectangle(800, 600, color.RGBA{100, 100, 100, 0xff})

	stack, err := g.Upload(context.Background(), storage, buf, exampleName, exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	pipe := gallery.ProcessingPipeline{
		gallery.Resizer{
			"small":  {Width: 640},
			"medium": {Width: 1280},
			"large":  {Width: 1920},
		},
	}

	processed, err := svc.Process(context.Background(), stack, pipe)
	if err != nil {
		t.Fatalf("processing failed: %v", err)
	}

	want, err := pipe.Process(context.Background(), stack, enc, storage)
	if err != nil {
		t.Fatalf("ProcessorPipeline failed: %v", err)
	}

	if !reflect.DeepEqual(want, processed) {
		t.Fatalf("Process returned wrong Stack.\n\nwant=%#v\n\ngot=%#v", want, processed)
	}
}

func TestPostProcessor_Run(t *testing.T) {
	enc := image.NewEncoder()
	storage := media.NewStorage(media.ConfigureDisk(exampleDisk, media.MemoryDisk()))
	ebus := chanbus.New()
	estore := eventstore.WithBus(memstore.New(), ebus)
	aggregates := repository.New(estore)
	galleries := gallery.GoesRepository(aggregates)
	svc := gallery.NewPostProcessor(enc, storage, galleries)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pipe := gallery.ProcessingPipeline{
		gallery.Resizer{
			"small":  {Width: 640},
			"medium": {Width: 1280},
			"large":  {Width: 1920},
		},
	}

	processedStack := make(chan gallery.Stack)

	errs, err := svc.Run(ctx, ebus, pipe, gallery.OnProcessed(func(s gallery.Stack, _ *gallery.Gallery) {
		processedStack <- s
	}))

	if err != nil {
		t.Fatalf("Run failed with %q", err)
	}

	g := gallery.New(uuid.New())
	g.Create("foo")

	_, buf := imggen.ColoredRectangle(800, 600, color.RGBA{100, 100, 100, 0xff})

	uploaded, err := g.Upload(context.Background(), storage, buf, exampleName, exampleDisk, examplePath)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	if err := galleries.Save(ctx, g); err != nil {
		t.Fatalf("failed to save Gallery: %v", err)
	}

	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()

	var stack gallery.Stack
	select {
	case <-timer.C:
		t.Fatal("timed out")
	case err := <-errs:
		t.Fatal(err)
	case stack = <-processedStack:
	}

	want, err := pipe.Process(context.Background(), uploaded, enc, storage)
	if err != nil {
		t.Fatalf("ProcessingPipeline failed: %v", err)
	}

	if !reflect.DeepEqual(want, stack) {
		t.Fatalf("PostProcessor's processed Stack is wrong.\n\nwant=%v\n\ngot=%v", want, stack)
	}
}
