package page_test

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/modernice/nice-cms/static/page"
)

func TestPage_MarshalJSON(t *testing.T) {
	p := page.New(uuid.New())

	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("json.Marshal failed with %q", err)
	}

	var unmarshaled page.Page
	if err := json.Unmarshal(b, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal failed with %q", err)
	}

	if !cmp.Equal(p, &unmarshaled) {
		t.Fatalf("invalid unmarshal.\n\n%s", cmp.Diff(p, &unmarshaled))
	}
}
