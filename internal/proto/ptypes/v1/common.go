package ptypes

import (
	"github.com/google/uuid"
	protocommon "github.com/modernice/nice-cms/internal/proto/gen/common/v1"
)

// UUIDProto encodes a UUID.
func UUIDProto(id uuid.UUID) *protocommon.UUID {
	return &protocommon.UUID{Bytes: id[:]}
}

// UUID deoodes a UUID.
func UUID(id *protocommon.UUID) uuid.UUID {
	var b [16]byte
	copy(b[:], id.GetBytes())
	return uuid.UUID(b)
}
