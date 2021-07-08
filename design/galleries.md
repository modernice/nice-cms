# Galleries

Galleries provides additional features around image management and should be the
main entrypoint for managing image assets.

## Requirements

- automatically resize uploaded images
- rename images
- replace images

## Design

### Setup image resizing

Image resizing should be simple to set up:

```go
package example

func NewGalleryService(
	stacks gallery.StackRepository,
	imageService *image.Service,
) *gallery.Service {
	return gallery.NewService(
		stacks,
		imageService,
		gallery.WithResizer(gallery.Dimensions{
			"thumb": media.Dimension{240, 0},
			"small": media.Dimension{640, 0},
			"medium": media.Dimension{1280, 0},
			"large": media.Dimension{1920, 0},
		}),
	)
}
```

```go
package example

func NewMediaServer(galleries *gallery.Service) http.Handler {
	return media.NewHTTPServer(
		media.WithGalleryService(galleries),
	)
}
```

Resizing should happen asynchronously in the background.

### Upload an image

**Go:**

```go
package example

func UploadImage(img io.Reader, svc *gallery.Service) {
	gallery, err := svc.Gallery("main")
	if err != nil {
		panic(fmt.Errorf(`get "main" gallery: %w`, err))
	}

	stack, processed, err := gallery.Upload(
		context.TODO(), img, "name", "disk", "path",
		gallery.WithAltDescription("alt description"),
		gallery.WithTags("foo", "bar", "baz"),
	)
	if err != nil {
		panic(fmt.Errorf("upload image: %w", err))
	}

	// stack.ID != uuid.Nil

	for _, img := range stack.Images {
		if img.Original {
			continue
		}

		if img.Pending {
			log.Printf("Image %q pending processing...\n", img.Path)
		}
	}
	
	select {
	case <-time.After(time.Minute):
		panic("timed out after 1m")
	case err := <-processed:
		if err != nil {
			panic(fmt.Errorf("processing failed: %w", err))
		}

		stack, err = gallery.Fetch(context.TODO(), stack.ID)
		if err != nil {
			panic(fmt.Errorf("fetch stack: %w", err))
		}
	}
}
```

**JS:**

```ts
import { connect } from '@nice-cms/media'

const media = await connect('http://localhost:8000')
const gallery = await media.galleries.fetch('main')

let image: Image

const { stack, onProcessed } = await gallery.upload(image, 'name', 'disk', 'path')

for (const img of stack.images) {
	if (img.pending) {
		console.log(`Image "${img.path}" is pending processing...`)
		continue
	}

	console.log(`Image "${img.path}" is ready.`)
}

onProcessed(stack => {
	for (const img of stack.images) {
		if (!img.original) {
			console.log(`Image "${img.path}" processed.`)
		}
	}
})
```

### Delete an image (stack)

**Go:**

```go
package example

func DeleteImage(stack *gallery.Stack, svc *gallery.Service) {
	gallery, err := svc.Gallery("main")
	if err != nil {
		panic(fmt.Errorf(`get "main" gallery: %w`, err))
	}

	if err := gallery.Delete(context.TODO(), stack); err != nil {
		panic(fmt.Errorf("delete stack: %w", err))
	}
}
```

**JS:**

```ts
import { connect } from '@nice-cms/media'

const media = await connect('http://localhost:8000')
const gallery = await media.galleries.fetch('main')
const stack = gallery.stack('{StackID}')

await gallery.delete(stack.id)

// stack.deleted === true
```

### Add/remove tags

**Go:**

```go
package example

func AddTags(stackID uuid.UUID, gallery *gallery.Gallery) {
	gallery.Tag(context.TODO(), stackID)
}
```

**JS:**

```ts
import { connect } from '@nice-cms/media'

const media = await connect('http://localhost:8000')
const gallery = await media.galleries.fetch('main')
let stack = gallery.stack('{StackID}')

await gallery.tag(stack.ID, 'foo', 'bar', 'baz')
await gallery.untag(stack.ID, 'bar')

// stack.tags === ['foo', 'baz']
```
