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

	"github.com/google/uuid"
	"github.com/modernice/cms/internal/imggen"
	"github.com/modernice/cms/media"
	"github.com/modernice/cms/media/image"
	"github.com/modernice/cms/media/image/gallery"
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
