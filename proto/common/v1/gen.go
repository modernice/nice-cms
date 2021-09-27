package common

//go:generate mkdir -p ../../gen/common/v1
//go:generate protoc -I ../../ -I. --go_out=module=github.com/modernice/nice-cms/proto/gen/common/v1:../../gen/common/v1 --go-grpc_out=module=github.com/modernice/nice-cms/proto/gen/common/v1:../../gen/common/v1 common.proto
