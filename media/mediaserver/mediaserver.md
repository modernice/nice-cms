# Media server

```go
var commands command.Bus
var galleries gallery.Repository

srv := mediaserver.New(commands, mediaserver.WithRoutes(routes.Gallery{
	Disabled: []string{routes.UploadGalleryImage},
}), mediaserver.WithRoutes(routes.Document{
	Disabled: []string{routes.UploadDocument},
}))
```
