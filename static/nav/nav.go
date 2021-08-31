package nav

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/modernice/goes/aggregate"
	"github.com/modernice/goes/event"
	"github.com/modernice/nice-cms/internal/unique"
)

// Aggregate is the name of the Nav aggregate.
const Aggregate = "cms.static.nav"

var (
	// ErrEmptyName is returned when trying to create a Nav with an empty name.
	ErrEmptyName = errors.New("empty name")

	// ErrInitialItem is returned when trying to remove an initial Item from a Nav.
	ErrInitialItem = errors.New("initial item")

	// ErrItemNotFound is returned when an Item cannot be found within the Tree of a Nav.
	ErrItemNotFound = errors.New("item not found")

	// ErrDuplicateItem is returned when trying to add an Item with an already
	// used ID to a Tree.
	ErrDuplicateItem = errors.New("duplicate item")
)

// Repository is the repository for navigations.
type Repository interface {
	// Save saves a Nav.
	Save(context.Context, *Nav) error

	// Fetch returns the Nav with the given UUID.
	Fetch(context.Context, uuid.UUID) (*Nav, error)

	// Use fetches the Nav with the given UUID, calls the provided function with
	// the Nav as the argument and then saves the Nav. If the provided function
	// returns a non-nil error, the Nav is not saved and that error is returned.
	Use(context.Context, uuid.UUID, func(*Nav) error) error

	// Delete deletes the given Nav.
	Delete(context.Context, *Nav) error
}

// Nav is a navigation.
type Nav struct {
	*aggregate.Base
	*Tree

	Name string
}

// Tree is an Item tree.
type Tree struct {
	Items []Item `json:"items"`
}

// NewTree returns a Tree of the given Items.
func NewTree(items ...Item) *Tree {
	return &Tree{Items: items}
}

// Create creates a Nav with the provided name and adds the given Items to it.
// The provided Items are added as initial items and cannot be removed from the
// Nav.
//
// Create should typically not be called manually, but rather by the command
// handler because the command handler validates the uniqueness of the provided
// name through a *Lookup:
//
//	var bus command.Bus
//	cmd := nav.CreateCmd(name, items...)
//	bus.Dispatch(context.TODO(), cmd)
func Create(name string, items ...Item) (*Nav, error) {
	return CreateWithID(uuid.New(), name, items...)
}

// CreateWithID does the same as Create, but accepts a custom UUID.
func CreateWithID(id uuid.UUID, name string, items ...Item) (*Nav, error) {
	nav := New(id)

	if err := nav.Create(name); err != nil {
		return nav, err
	}

	if len(items) > 0 {
		if err := nav.initial(items...); err != nil {
			return nav, err
		}
	}

	return nav, nil
}

// New returns an uncreated Nav n. Call n.Create with the name of the Nav to
// create it.
func New(id uuid.UUID) *Nav {
	return &Nav{
		Base: aggregate.New(Aggregate, id),
		Tree: NewTree(),
	}
}

// HasItem returns whether the Nav has the given items. HasItem accepts
// dot-seperated paths to reference nested items:
//
//	nav, _ := Create("example", NewLabel("foo", "Foo"), NewLabel("bar", "Bar", SubTree(
//		NewLabel("baz", "Baz"),
//	)))
//	nav.HasItem("foo", "bar", "bar.baz")
func (nav *Nav) HasItem(items ...string) bool {
	if len(items) == 0 {
		return true
	}

	for _, path := range items {
		if _, err := nav.Item(path); err != nil {
			return false
		}
	}

	return true
}

// Create creates the navigation by giving it a name.
func (nav *Nav) Create(name string) error {
	if name = strings.TrimSpace(name); name == "" {
		return ErrEmptyName
	}

	aggregate.NextEvent(nav, Created, CreatedData{Name: name})

	return nil
}

func (nav *Nav) createTree(evt event.Event) {
	data := evt.Data().(CreatedData)
	nav.Name = data.Name
}

// Prepend prepends Items at the root level of the navigation.
func (nav *Nav) Prepend(items ...Item) error {
	return nav.Insert(0, items...)
}

// PrependAt prepends Items at the given path.
func (nav *Nav) PrependAt(path string, items ...Item) error {
	return nav.InsertAt(path, 0, items...)
}

// Append appends Items at the root level of the navigation.
func (nav *Nav) Append(items ...Item) error {
	return nav.Insert(len(nav.Items), items...)
}

// AppendAt appends Items at the given path.
func (nav *Nav) AppendAt(path string, items ...Item) error {
	item, err := nav.Item(path)
	if err != nil || item.Tree == nil {
		return err
	}
	return nav.InsertAt(path, len(item.Tree.Items), items...)
}

// Insert inserts Items at the given index at the root level of the navigation.
func (nav *Nav) Insert(index int, items ...Item) error {
	if index < 0 {
		return fmt.Errorf("negative index %d", index)
	}

	if err := nav.Tree.checkDuplicates(items...); err != nil {
		return err
	}

	if index >= len(nav.Items) {
		index = len(nav.Items)
	}

	aggregate.NextEvent(nav, ItemsAdded, ItemsAddedData{
		Items: items,
		Index: index,
	})

	return nil
}

// InsertAt inserts Items at the given index of the Tree at path.
func (nav *Nav) InsertAt(path string, index int, items ...Item) error {
	if path == "" {
		return nav.Insert(index, items...)
	}

	if index < 0 {
		return fmt.Errorf("negative index %d", index)
	}

	item, err := nav.Item(path)
	if err != nil {
		return err
	}

	var childItems int
	if item.Tree != nil {
		childItems = len(item.Tree.Items)

		if err := item.Tree.checkDuplicates(items...); err != nil {
			return err
		}
	}

	if index >= childItems {
		index = childItems
	}

	aggregate.NextEvent(nav, ItemsAdded, ItemsAddedData{
		Items: items,
		Index: index,
		Path:  path,
	})

	return nil
}

func (nav *Nav) initial(items ...Item) error {
	initial := make([]Item, len(items))
	copy(initial, items)
	for i, item := range initial {
		initial[i] = deepInitial(item)
	}
	return nav.Insert(0, initial...)
}

func deepInitial(item Item) Item {
	item.Initial = true
	if item.Tree == nil {
		return item
	}
	for i, itm := range item.Tree.Items {
		item.Tree.Items[i] = deepInitial(itm)
	}
	return item
}

func (nav *Nav) addItems(evt event.Event) {
	data := evt.Data().(ItemsAddedData)

	if data.Path == "" {
		nav.addRootItems(data)
		return
	}

	item, err := nav.Item(data.Path)
	if err != nil {
		return
	}
	item.ensureTree()

	item.Tree.insert(data.Index, data.Items...)

	nav.replace(data.Path, item)
}

func (nav *Nav) addRootItems(data ItemsAddedData) {
	nav.insert(data.Index, data.Items...)
}

func (nav *Nav) replace(path string, replacement Item) {
	ids := strings.Split(path, ".")
	if len(ids) == 0 {
		return
	}
	if len(ids) == 1 {
		nav.Tree.replace(ids[0], replacement)
		return
	}

	parent, err := nav.Item(parentPath(path))
	if err != nil {
		return
	}
	parent.ensureTree()

	parent.Tree.replace(ids[len(ids)-1], replacement)
}

func parentPath(path string) string {
	ids := strings.Split(path, ".")
	if len(ids) == 0 {
		return ""
	}
	if len(ids) == 1 {
		return ids[1]
	}
	return strings.Join(ids[:len(ids)-1], ".")
}

// Remove removes Items from Nav. Remove accepts dot-seperated paths to
// reference nested Items:
//
//	nav.Remove("foo.bar.baz")
func (nav *Nav) Remove(items ...string) error {
	paths := make([]string, 0, len(items))

	for _, path := range items {
		item, err := nav.Item(path)
		if err != nil {
			continue
		}

		if item.Initial {
			return fmt.Errorf("cannot remove initial item: %w", ErrInitialItem)
		}

		paths = append(paths, path)
	}

	aggregate.NextEvent(nav, ItemsRemoved, ItemsRemovedData{Items: paths})

	return nil
}

func (nav *Nav) removeItems(evt event.Event) {
	data := evt.Data().(ItemsRemovedData)
	nav.remove(data.Items...)
}

func (t *Tree) remove(items ...string) {
	for _, path := range items {
		ids := strings.Split(path, ".")
		if len(ids) == 0 {
			continue
		}

		if len(ids) == 1 {
			t.removeItem(ids[0])
			continue
		}

		parentPath := strings.Join(ids[:len(ids)-1], ".")
		parent, err := t.Item(parentPath)
		if err != nil {
			continue
		}

		item, err := t.Item(path)
		if err != nil {
			continue
		}

		if parent.Tree != nil {
			parent.Tree.removeItem(item.ID)
		}
	}
}

func (t *Tree) removeItem(id string) {
	for i, item := range t.Items {
		if item.ID == id {
			t.Items = append(t.Items[:i], t.Items[i+1:]...)
			return
		}
	}
}

// Sort sorts the root level of the navigation. Items are sorted by the sorting
// of the Item IDs in sorting:
//
//	nav.Sort([]string{"bar", "baz", "foo"})
func (nav *Nav) Sort(sorting []string) {
	nav.SortAt("", sorting)
}

// SortAt sorts the tree at the given path. Items are sorted by the sorting
// of the Item IDs in sorting:
//
//	nav.SortAt("foo.bar", []string{"bar", "baz", "foo"})
func (nav *Nav) SortAt(path string, sorting []string) {
	tree := nav.Tree

	if path != "" {
		item, err := nav.Item(path)
		if err != nil || item.Tree == nil {
			return
		}
		tree = item.Tree
	}

	ids := make([]string, 0, len(sorting))
	for _, id := range unique.Strings(sorting...) {
		if _, err := tree.Item(id); err != nil {
			continue
		}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return
	}

	items := tree.copyItems()
	if sort.SliceIsSorted(items, nav.lessFunc(items, ids)) {
		return
	}

	aggregate.NextEvent(nav, Sorted, SortedData{
		Sorting: ids,
		Path:    path,
	})
}

func (nav *Nav) sort(evt event.Event) {
	data := evt.Data().(SortedData)

	if data.Path == "" {
		sorted := nav.copyItems()
		sort.SliceStable(sorted, nav.lessFunc(sorted, data.Sorting))
		nav.Items = sorted
		return
	}

	parent, err := nav.Item(parentPath(data.Path))
	if err != nil || parent.Tree == nil {
		return
	}

	item, err := nav.Item(data.Path)
	if err != nil || item.Tree == nil {
		return
	}

	sorted := item.Tree.copyItems()
	sort.SliceStable(sorted, item.Tree.lessFunc(sorted, data.Sorting))
	item.Tree.Items = sorted

	parent.Tree.replace(item.ID, item)
}

// ApplyEvent applies aggregate events.
func (nav *Nav) ApplyEvent(evt event.Event) {
	switch evt.Name() {
	case Created:
		nav.createTree(evt)
	case ItemsAdded:
		nav.addItems(evt)
	case ItemsRemoved:
		nav.removeItems(evt)
	case Sorted:
		nav.sort(evt)
	}
}

// Item returns the Item at the given path, or ErrItemNotFound.
func (t *Tree) Item(path string) (Item, error) {
	ids := strings.Split(path, ".")
	if len(ids) == 0 {
		return Item{}, ErrItemNotFound
	}

	item, err := t.item(ids[0])
	if err != nil {
		return item, err
	}

	if len(ids) == 1 {
		return item, nil
	}

	if item.Tree == nil {
		return Item{}, ErrItemNotFound
	}

	return item.Tree.Item(strings.Join(ids[1:], "."))
}

func (t *Tree) item(id string) (Item, error) {
	for _, item := range t.Items {
		if item.ID == id {
			return item, nil
		}
	}
	return Item{}, ErrItemNotFound
}

func (t *Tree) insert(index int, items ...Item) {
	add := append(items, t.Items[index:]...)
	t.Items = append(t.Items[:index], add...)
}

func (t *Tree) replace(id string, replacement Item) {
	for i, item := range t.Items {
		if item.ID == id {
			t.Items[i] = replacement
			return
		}
	}
}

func (t *Tree) copyItems() []Item {
	out := make([]Item, len(t.Items))
	copy(out, t.Items)
	return out
}

func (t *Tree) lessFunc(items []Item, sorting []string) func(i, j int) bool {
	return func(i, j int) bool {
		left := items[i]
		right := items[j]
		leftIndex, rightIndex := -1, -1

		for i, id := range sorting {
			if left.ID == id {
				leftIndex = i
				break
			}
			if right.ID == id {
				rightIndex = i
				break
			}
		}

		if leftIndex > -1 {
			if rightIndex > -1 && leftIndex < rightIndex {
				return true
			}
			return true
		}

		if rightIndex > -1 {
			if leftIndex > -1 && rightIndex < leftIndex {
				return false
			}
			return false
		}

		for _, item := range t.Items {
			if item.ID == left.ID {
				return true
			}
			if item.ID == right.ID {
				return false
			}
		}

		return true
	}
}

func (t *Tree) checkDuplicates(items ...Item) error {
	for _, item := range items {
		if _, err := t.Item(item.ID); err == nil {
			return fmt.Errorf("%q: %w", item.ID, ErrDuplicateItem)
		}
	}
	return nil
}

type goesRepository struct {
	repo aggregate.Repository
}

// GoesRepository returns a Repository that uses the provided aggregate
// repository under the hood.
func GoesRepository(repo aggregate.Repository) Repository {
	return &goesRepository{repo}
}

func (r *goesRepository) Save(ctx context.Context, n *Nav) error {
	return r.repo.Save(ctx, n)
}

func (r *goesRepository) Fetch(ctx context.Context, id uuid.UUID) (*Nav, error) {
	n := New(id)
	if err := r.repo.Fetch(ctx, n); err != nil {
		return n, fmt.Errorf("goes: %w", err)
	}
	return n, nil
}

func (r *goesRepository) Use(ctx context.Context, id uuid.UUID, fn func(*Nav) error) error {
	n, err := r.Fetch(ctx, id)
	if err != nil {
		return fmt.Errorf("fetch nav: %w", err)
	}
	if err := fn(n); err != nil {
		return err
	}
	if err := r.Save(ctx, n); err != nil {
		return fmt.Errorf("save nav: %w", err)
	}
	return nil
}

func (r *goesRepository) Delete(ctx context.Context, n *Nav) error {
	return r.repo.Delete(ctx, n)
}

type jsonNav struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Items []Item    `json:"items"`
}

func (n *Nav) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonNav{
		ID:    n.ID,
		Name:  n.Name,
		Items: n.Items,
	})
}

func (n *Nav) UnmarshalJSON(b []byte) error {
	var jn jsonNav
	if err := json.Unmarshal(b, &jn); err != nil {
		return err
	}
	nav := New(jn.ID)
	nav.Name = jn.Name
	nav.Items = jn.Items
	*n = *nav
	return nil
}
