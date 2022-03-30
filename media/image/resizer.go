package image

import (
	"image"
	"sync"

	"github.com/disintegration/imaging"
)

// Dimensions are image dimensions (width & height).
type Dimensions struct {
	Width  int
	Height int
}

// A Resizer resizes images into different Dimensions. Resizing is done in
// parallel for each Dimensions.
type Resizer map[string]Dimensions

// Resize resizes an Image into the configured Dimensions. Providing 0 as the
// Width or Height for Dimensions of a size preserves the aspect-ratio for that
// size.
//
//	var img image.Image
//
//	r := Resizer{
//		"small": {Width: 640, Height: 0},
//		"medium": {Width: 1280, Height: 0},
//		"large": {Width: 1920, Height: 0},
//	}
//
//	resized := r.Resize(img)
//	// resized["small"].Bounds().Dx() == 640
//	// resized["medium"].Bounds().Dx() == 1280
//	// resized["large"].Bounds().Dx() == 1920
func (r Resizer) Resize(img image.Image) map[string]image.Image {
	type result struct {
		name string
		img  *image.NRGBA
	}

	results := make(chan result)

	var wg sync.WaitGroup
	for name, d := range r {
		wg.Add(1)
		go func(name string, d Dimensions) {
			defer wg.Done()
			results <- result{
				name: name,
				img:  d.Resize(img),
			}
		}(name, d)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	out := make(map[string]image.Image)
	for res := range results {
		out[res.name] = res.img
	}

	return out
}

// Resize resizes an Image into the Dimensions.
func (dim Dimensions) Resize(img image.Image) *image.NRGBA {
	return imaging.Resize(img, dim.Width, dim.Height, imaging.Lanczos)
}
