package media

// import (
// 	"context"
// 	"io"
// 	"time"
// )

// // VideoService provides uploads and retrieval of videos.
// type VideoService interface {
// 	// Upload uploads a video to the specified disk and path and returns the
// 	// uploaded Video.
// 	Upload(_ context.Context, _ io.Reader, name, disk, path string) (Video, error)

// 	// Delete deletes the video at the specified disk and path. Delete returns
// 	// ErrFileNotFound if the video does not exist.
// 	Delete(_ context.Context, disk, path string) error

// 	// Get returns
// 	Get(_ context.Context, disk, path string) (Video, error)
// }

// // Video is an uploaded video.
// type Video struct {
// 	File

// 	Width    int
// 	Height   int
// 	Duration time.Duration
// }
