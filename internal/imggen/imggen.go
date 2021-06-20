package imggen

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
)

// ColoredRectangle creates a rectangle with the given dimensions and color.
func ColoredRectangle(width, height int, color color.Color) (image.Image, *bytes.Buffer) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, color)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(fmt.Errorf("encode png: %w", err))
	}
	return img, &buf
}
