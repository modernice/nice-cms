package media

//go:generate mkdir -p ../../gen/media/v1
//go:generate protoc -I ../../ -I. --go_out=module=github.com/modernice/nice-cms/internal/proto/gen/media/v1:../../gen/media/v1 --go-grpc_out=module=github.com/modernice/nice-cms/internal/proto/gen/media/v1:../../gen/media/v1 media.proto
