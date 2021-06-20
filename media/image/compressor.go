package image

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
)

// PNGCompressor compresses images using a png.Encoder.
//
//	var img image.Image
//	c := PNGCompressor(png.BestCompression)
//	compressed := c.Compress(img)
type PNGCompressor png.CompressionLevel

// CompressionLevel returns the png.CompressionLevel.
func (comp PNGCompressor) CompressionLevel() png.CompressionLevel {
	return png.CompressionLevel(comp)
}

// Compress compresses the provided Image and returns the compressed Image.
// Compressor uses a png.Encoder to compress the Image with
// comp.CompressionLevel() as the compression level.
func (comp PNGCompressor) Compress(img image.Image) (image.Image, error) {
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.CompressionLevel(comp)}

	if err := enc.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encode as png with compression level %d: %w", comp.CompressionLevel(), err)
	}

	img, _, err := image.Decode(&buf)
	if err != nil {
		return nil, fmt.Errorf("decode compressed image: %w", err)
	}

	return img, nil
}
