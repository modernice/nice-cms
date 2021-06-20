package image_test

import (
	"bytes"
	stdimage "image"
	"image/color"
	"testing"

	"github.com/modernice/cms/internal/imggen"
	"github.com/modernice/cms/media/image"
)

func TestEncoder_Encode(t *testing.T) {
	enc := image.NewEncoder()

	tests := []string{"jpeg", "png"}

	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			img, _ := imggen.ColoredRectangle(800, 600, color.RGBA{100, 100, 100, 0xff})

			var buf bytes.Buffer

			err := enc.Encode(&buf, img, tt)
			if err != nil {
				t.Fatalf("Encoder.Encode failed with %q", err)
			}

			_, format, err := stdimage.Decode(&buf)
			if err != nil {
				t.Fatalf("image.Decode failed with %q", err)
			}

			if format != tt {
				t.Fatalf("decoded image should have format %q; has format %q", tt, format)
			}
		})
	}
}
