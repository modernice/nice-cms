package image_test

import (
	"image/color"
	"testing"

	"github.com/modernice/cms/internal/imggen"
	"github.com/modernice/cms/media/image"
)

func TestResizer_Resize(t *testing.T) {
	resizer := image.Resizer{
		"thumb":  {240, 120},
		"small":  {640, 320},
		"medium": {1280, 600},
		"large":  {1920, 1200},
	}

	orgWidth := 800
	orgHeight := 600
	rect, _ := imggen.ColoredRectangle(orgWidth, orgHeight, color.RGBA{100, 100, 100, 0xff})

	resized := resizer.Resize(rect)

	for dimensionName, dimension := range resizer {
		var found bool
		for resizedDimensionName, resizedImage := range resized {
			if resizedDimensionName != dimensionName {
				continue
			}

			found = true

			width := resizedImage.Bounds().Dx()
			height := resizedImage.Bounds().Dy()

			if width != dimension.Width {
				t.Fatalf("resized image %q should have width of %d; has %d", dimensionName, dimension.Width, width)
			}

			if height != dimension.Height {
				t.Fatalf("resized image %q should have height of %d; has %d", dimensionName, dimension.Height, height)
			}

			break
		}

		if !found {
			t.Fatalf("expected resized image with %q dimensions", dimensionName)
		}
	}
}
