package nav_test

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/modernice/goes/aggregate"
	"github.com/modernice/nice-cms/static/nav"
)

func TestNav_MarshalJSON(t *testing.T) {
	n := nav.New(uuid.New())

	b, err := json.Marshal(n)
	if err != nil {
		t.Fatalf("json.Marshal failed with %q", err)
	}

	var unmarshaled nav.Nav
	if err := json.Unmarshal(b, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal failed with %q", err)
	}

	if !cmp.Equal(n, &unmarshaled, cmpopts.IgnoreUnexported(aggregate.Base{})) {
		t.Fatalf("invalid unmarshal.\n\n%s", cmp.Diff(n, &unmarshaled))
	}
}
