package image

import (
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"sync"
)

var (
	// ErrUnknownFormat is returned when trying to encode an image whose format
	// is not registered in an Encoder.
	ErrUnknownFormat = errors.New("unknown format")
)

// FormatEncoder encodes images of a specific format (JPEG, PNG etc.).
type FormatEncoder interface {
	// Encode encodes the provided Image and writes the result into the
	// specified Writer.
	Encode(io.Writer, image.Image) error
}

// Func allows functions to be used as FormatEncoders.
type Func func(io.Writer, image.Image) error

// Encode returns fn(w, img).
func (fn Func) Encode(w io.Writer, img image.Image) error {
	return fn(w, img)
}

// Encoder is a multi-format image encoder.
type Encoder interface {
	// Encode encodes an image using the appropriate FormatEncoder for the
	// specified format.
	Encode(io.Writer, image.Image, string) error
}

type encoder struct {
	mux      sync.RWMutex
	encoders map[string]FormatEncoder
}

// EncoderOption is an Encoder option.
type EncoderOption func(*encoder)

// WithFormat returns an EncoderOption that registers a FormatEncoder for the
// given image format.
//
// Format must be the same as would be returned by image.Decode. Provide an
// empty string as format to set the default encoder for unknown formats.
func WithFormat(format string, enc FormatEncoder) EncoderOption {
	return func(e *encoder) {
		e.encoders[format] = enc
	}
}

// NewEncoder returns a new Encoder with default support for JPEGs, GIFs and
// PNGs. When provided an unknown image format, Encoder falls back to the PNG
// encoder.
func NewEncoder(opts ...EncoderOption) Encoder {
	return newEncoder(opts...)
}

func newEncoder(opts ...EncoderOption) *encoder {
	enc := encoder{
		encoders: map[string]FormatEncoder{
			"jpeg": Func(JPEGEncoder),
			"gif":  Func(GIFEncoder),
			"":     Func(PNGEncoder),
		},
	}
	for _, opt := range opts {
		opt(&enc)
	}
	return &enc
}

// JPEGEncoder encodes images using jpeg.Encode with maximum quality.
func JPEGEncoder(w io.Writer, img image.Image) error {
	return jpeg.Encode(w, img, &jpeg.Options{Quality: 100})
}

// GIFEncoder encodes images using gif.Encode with default options.
func GIFEncoder(w io.Writer, img image.Image) error {
	return gif.Encode(w, img, nil)
}

var pngEncoder = png.Encoder{CompressionLevel: png.BestCompression}

// PNGEncoder encodes images using a png.Encoder with png.BestCompression as the
// compression level.
func PNGEncoder(w io.Writer, img image.Image) error {
	return pngEncoder.Encode(w, img)
}

// Encode encodes the provided Image using the appropriate encoder for the
// specified format or the default encoder if the format has no configured
// FormatEncoder.
func (enc *encoder) Encode(w io.Writer, img image.Image, format string) error {
	if enc == nil {
		*enc = *newEncoder()
	}

	fenc, err := enc.get(format)
	if err != nil {
		return fmt.Errorf("get %q encoder: %w", format, err)
	}

	if err := fenc.Encode(w, img); err != nil {
		return fmt.Errorf("%q encoder: %w", format, err)
	}

	return nil
}

func (enc *encoder) get(format string) (FormatEncoder, error) {
	enc.mux.RLock()
	defer enc.mux.RUnlock()
	if fenc, ok := enc.encoders[format]; ok {
		return fenc, nil
	}
	if fenc, ok := enc.encoders[""]; ok {
		return fenc, nil
	}
	return nil, ErrUnknownFormat
}
