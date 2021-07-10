package nav

// Item types
const (
	Label      = ItemType("label")
	StaticLink = ItemType("static_link")
)

// ItemType is an Item type.
type ItemType string

// Item is a navigation item.
type Item struct {
	ID      string   `json:"id"`
	Type    ItemType `json:"type"`
	Initial bool     `json:"initial"`

	Paths  map[string]string `json:"localePaths"`
	Labels map[string]string `json:"localeLabels"`

	Tree *Tree `json:"tree"`
}

// ItemOption is an option for an Item.
type ItemOption func(*Item)

// LocalePath returns an ItemOption that adds a localized path to an Item.
func LocalePath(locale, path string) ItemOption {
	return func(i *Item) {
		if i.Type != Label {
			i.Paths[locale] = path
		}
	}
}

// LocaleLabel returns an ItemOption that adds a localized label to an Item.
func LocaleLabel(locale, label string) ItemOption {
	return func(i *Item) {
		i.Labels[locale] = label
	}
}

// SubTree returns an ItemOption that adds a subtree to an Item.
func SubTree(items ...Item) ItemOption {
	return func(i *Item) {
		i.Tree = NewTree(items...)
	}
}

// Initial returns an ItemOption that marks an Item as "initial". An initial
// Item cannot be removed from a Nav.
func Initial() ItemOption {
	return func(i *Item) {
		i.Initial = true
	}
}

// NewItem returns an Item with the given ID and ItemType.
func NewItem(id string, typ ItemType, opts ...ItemOption) Item {
	item := Item{
		ID:     id,
		Type:   typ,
		Paths:  make(map[string]string),
		Labels: make(map[string]string),
	}
	for _, opt := range opts {
		opt(&item)
	}
	return item
}

// NewLabel returns an Item of type Label with the given default label.
func NewLabel(id, label string, opts ...ItemOption) Item {
	opts = append([]ItemOption{LocaleLabel("", label)}, opts...)
	return NewItem(id, Label, opts...)
}

// NewStaticLink returns an Item of type StaticLink with the given default path and label.
func NewStaticLink(id, path, label string, opts ...ItemOption) Item {
	opts = append([]ItemOption{
		LocalePath("", path),
		LocaleLabel("", label),
	}, opts...)
	return NewItem(id, StaticLink, opts...)
}

// Path returns the path for the given locale or the default path.
func (i Item) Path(locale string) string {
	if path, ok := i.Paths[locale]; ok {
		return path
	}
	return i.Paths[""]
}

// Label returns the label for the given locale or the default label.
func (i Item) Label(locale string) string {
	if t, ok := i.Labels[locale]; ok {
		return t
	}
	return i.Labels[""]
}

func (i *Item) ensureTree() {
	if i.Tree == nil {
		i.Tree = NewTree()
	}
}
