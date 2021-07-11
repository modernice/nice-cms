package nav_test

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/modernice/cms/static/nav"
)

func TestPage_MarshalJSON(t *testing.T) {
	n := nav.New(uuid.New())

	b, err := json.Marshal(n)
	if err != nil {
		t.Fatalf("json.Marshal failed with %q", err)
	}

	var unmarshaled nav.Nav
	if err := json.Unmarshal(b, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal failed with %q", err)
	}

	if !cmp.Equal(n, &unmarshaled) {
		t.Fatalf("invalid unmarshal.\n\n%s", cmp.Diff(n, &unmarshaled))
	}
}
