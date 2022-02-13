package nav_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/modernice/goes/test"
	"github.com/modernice/nice-cms/static/nav"
)

func TestCreate_emptyName(t *testing.T) {
	names := []string{"", " "}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			tree, err := nav.Create(name)
			if !errors.Is(err, nav.ErrEmptyName) {
				t.Fatalf("New should fail with %q; got %q", nav.ErrEmptyName, err)
			}

			if tree.Name != "" {
				t.Fatalf("ID should be %q; is %q", "", tree.Name)
			}
		})
	}
}

func TestCreate(t *testing.T) {
	name := "foo"
	tree, err := nav.Create(name)
	if err != nil {
		t.Fatalf("New failed with %q", err)
	}

	if tree.Name != name {
		t.Fatalf("ID should be %q; is %q", name, tree.Name)
	}

	test.Change(t, tree, nav.Created, test.EventData(nav.CreatedData{Name: name}))
}

func TestCreate_withItems(t *testing.T) {
	name := "foo"
	items := []nav.Item{
		nav.NewLabel("foo", "Foo"),
		nav.NewStaticLink("contact", "/contact", "Contact"),
	}
	tree, err := nav.Create(name, items...)
	if err != nil {
		t.Fatalf("New failed with %q", err)
	}

	if tree.Name != name {
		t.Fatalf("ID should be %q; is %q", name, tree.Name)
	}

	label, err := tree.Item("foo")
	if err != nil {
		t.Fatalf("Item(%q) failed with %q", "foo", err)
	}

	link, err := tree.Item("contact")
	if err != nil {
		t.Fatalf("Item(%q) failed with %q", "contact", err)
	}

	test.Change(t, tree, nav.Created, test.EventData(nav.CreatedData{Name: name}))
	test.Change(t, tree, nav.ItemsAdded, test.EventData(nav.ItemsAddedData{
		Items: initialItems(label, link),
		Index: 0,
	}))
}

func TestNav_Append(t *testing.T) {
	items := []nav.Item{
		nav.NewLabel("foo", "Foo"),
		nav.NewLabel("bar", "Bar"),
		nav.NewStaticLink("baz", "/baz", "Baz"),
	}

	tree, err := nav.Create("foo", items[0])
	if err != nil {
		t.Fatalf("New failed with %q", err)
	}

	if err := tree.Append(items[1:]...); err != nil {
		t.Fatalf("Append failed with %q", err)
	}

	if !tree.HasItem("foo", "bar", "baz") {
		t.Fatalf("Tree should have %v items", []string{"foo", "bar", "baz"})
	}

	test.Change(t, tree, nav.ItemsAdded, test.EventData(nav.ItemsAddedData{
		Items: initialItems(items[:1]...),
		Index: 0,
	}))

	test.Change(t, tree, nav.ItemsAdded, test.EventData(nav.ItemsAddedData{
		Items: items[1:],
		Index: 1,
	}))
}

func TestNav_Insert(t *testing.T) {
	items := []nav.Item{
		nav.NewLabel("foo", "Foo"),
		nav.NewLabel("bar", "Bar"),
		nav.NewStaticLink("baz", "/baz", "Baz"),
		nav.NewStaticLink("foobar", "/foobar", "Foobar"),
	}

	tree, err := nav.Create("foo", items[:2]...)
	if err != nil {
		t.Fatalf("New failed with %q", err)
	}

	if err := tree.Insert(1, items[2:]...); err != nil {
		t.Fatalf("Insert failed with %q", err)
	}

	want := []nav.Item{
		initialItem(items[0]),
		items[2],
		items[3],
		initialItem(items[1]),
	}

	if !reflect.DeepEqual(tree.Items, want) {
		t.Fatalf("Items should be %v; is %v", want, tree.Items)
	}

	test.Change(t, tree, nav.ItemsAdded, test.EventData(nav.ItemsAddedData{
		Items: initialItems(items[:2]...),
		Index: 0,
	}))

	test.Change(t, tree, nav.ItemsAdded, test.EventData(nav.ItemsAddedData{
		Items: items[2:],
		Index: 1,
	}))
}

func TestNav_Insert_duplicate(t *testing.T) {
	tree, _ := nav.Create(
		"foo",
		nav.NewLabel("foo", "Foo"),
		nav.NewLabel("bar", "Bar"),
		nav.NewLabel("baz", "Baz"),
	)

	if err := tree.Insert(2, nav.NewLabel("baz", "Baz")); !errors.Is(err, nav.ErrDuplicateItem) {
		t.Fatalf("Insert should fail with %q; failed with %q", nav.ErrDuplicateItem, err)
	}
}

func TestNav_InsertAt(t *testing.T) {
	items := []nav.Item{
		nav.NewLabel("foo", "Foo"),
		nav.NewLabel("bar", "Bar", nav.SubTree(
			nav.NewLabel("baz", "Baz", nav.SubTree(
				nav.NewLabel("foo", "Foo"),
				nav.NewLabel("bar", "Baz"),
				nav.NewLabel("baz", "Baz"),
			)),
		)),
	}

	moreItems := []nav.Item{
		nav.NewLabel("qux", "Qux"),
		nav.NewLabel("quux", "Quux"),
	}

	navi, err := nav.Create("foo", items...)
	if err != nil {
		t.Fatalf("New failed with %q", err)
	}

	test.Change(t, navi, nav.ItemsAdded, test.EventData(nav.ItemsAddedData{
		Items: initialItems(items...),
		Index: 0,
	}))

	want := []nav.Item{
		items[1].Tree.Items[0].Tree.Items[0],
		moreItems[0], moreItems[1],
		items[1].Tree.Items[0].Tree.Items[1],
		items[1].Tree.Items[0].Tree.Items[2],
	}

	if err := navi.InsertAt("bar.baz", 1, moreItems...); err != nil {
		t.Fatalf("InsertAt failed with %q", err)
	}

	item, err := navi.Item("bar.baz")
	if err != nil {
		t.Fatalf("Item(%q) failed with %q", "bar.baz", err)
	}

	if !reflect.DeepEqual(item.Tree.Items, want) {
		t.Fatalf("item.Tree.Items should be\n\n%v\n\nis\n\n%v\n\n%s\n", want, item.Tree.Items, cmp.Diff(want, item.Tree.Items))
	}

	test.Change(t, navi, nav.ItemsAdded, test.EventData(nav.ItemsAddedData{
		Items: moreItems,
		Index: 1,
		Path:  "bar.baz",
	}))
}

func TestNav_InsertAt_duplicate(t *testing.T) {
	tree, _ := nav.Create("foo", nav.NewLabel("foo", "foo", nav.SubTree(
		nav.NewLabel("foo", "Foo"),
		nav.NewLabel("bar", "Bar"),
		nav.NewLabel("baz", "Baz"),
	)))

	if err := tree.InsertAt("foo", 2, nav.NewLabel("baz", "Baz")); !errors.Is(err, nav.ErrDuplicateItem) {
		t.Fatalf("InsertAt should fail with %q; failed with %q", nav.ErrDuplicateItem, err)
	}
}

func TestNav_Remove(t *testing.T) {
	tree, _ := nav.Create("foo")

	items := []nav.Item{
		nav.NewLabel("foo", "Foo"),
		nav.NewLabel("bar", "Bar"),
		nav.NewLabel("baz", "Baz"),
		nav.NewLabel("qux", "Qux"),
	}

	tree.Append(items...)

	if err := tree.Remove("bar", "baz"); err != nil {
		t.Fatalf("Remove failed with %q", err)
	}

	if _, err := tree.Item("bar"); err == nil {
		t.Fatalf("%q Item not removed", "bar")
	}

	if _, err := tree.Item("baz"); err == nil {
		t.Fatalf("%q Item not removed", "baz")
	}

	if _, err := tree.Item("foo"); err != nil {
		t.Fatalf("Item(%q) failed with %q", "foo", err)
	}

	if _, err := tree.Item("qux"); err != nil {
		t.Fatalf("Item(%q) failed with %q", "qux", err)
	}

	test.Change(t, tree, nav.ItemsRemoved, test.EventData(nav.ItemsRemovedData{
		Items: []string{"bar", "baz"},
	}))
}

func TestNav_Remove_nested(t *testing.T) {
	tree, _ := nav.Create("foo")

	items := []nav.Item{
		nav.NewLabel("foo", "Foo"),
		nav.NewLabel("bar", "Bar", nav.SubTree(
			nav.NewLabel("foo", "Foo"),
			nav.NewLabel("bar", "Bar", nav.SubTree(
				nav.NewLabel("foo", "Foo"),
				nav.NewLabel("bar", "Bar"),
			)),
		)),
		nav.NewLabel("baz", "Baz", nav.SubTree(
			nav.NewLabel("foo", "Foo"),
			nav.NewLabel("bar", "Bar"),
		)),
	}

	tree.Append(items...)

	if err := tree.Remove("bar.bar.foo", "baz"); err != nil {
		t.Fatalf("Remove failed with %q", err)
	}

	if _, err := tree.Item("baz"); err == nil {
		t.Fatalf("%q Item not removed", "baz")
	}

	if _, err := tree.Item("bar.bar.foo"); err == nil {
		t.Fatalf("%q Item not removed", "bar.bar.foo")
	}

	if _, err := tree.Item("bar.bar"); err != nil {
		t.Fatalf("%q Item should not be removed", "bar.bar")
	}

	test.Change(t, tree, nav.ItemsRemoved, test.EventData(nav.ItemsRemovedData{
		Items: []string{"bar.bar.foo", "baz"},
	}))
}

func TestNav_Remove_initialItem(t *testing.T) {
	items := []nav.Item{
		nav.NewLabel("foo", "Foo"),
		nav.NewLabel("bar", "Bar"),
		nav.NewLabel("baz", "Baz"),
		nav.NewLabel("qux", "Qux"),
	}

	tree, _ := nav.Create("foo", items[:2]...)
	tree.Append(items[2:]...)

	if err := tree.Remove("qux", "bar"); !errors.Is(err, nav.ErrInitialItem) {
		t.Fatalf("Remove should fail with %q; got %q", nav.ErrInitialItem, err)
	}

	test.NoChange[any](t, tree, nav.ItemsRemoved)
}

func TestNav_Sort(t *testing.T) {
	items := []nav.Item{
		nav.NewLabel("foo", "Foo"),
		nav.NewLabel("bar", "Bar"),
		nav.NewLabel("baz", "Baz"),
		nav.NewLabel("qux", "Qux"),
	}

	tests := []struct {
		sort             []string
		want             []string
		wantEventSorting []string
		noChange         bool
	}{
		{
			want:     []string{"foo", "bar", "baz", "qux"},
			noChange: true,
		},
		{
			sort:     []string{},
			want:     []string{"foo", "bar", "baz", "qux"},
			noChange: true,
		},
		{
			sort:     []string{"foo", "bar", "baz", "qux"},
			want:     []string{"foo", "bar", "baz", "qux"},
			noChange: true,
		},
		{
			sort:             []string{"bar", "baz", "qux", "foo"},
			want:             []string{"bar", "baz", "qux", "foo"},
			wantEventSorting: []string{"bar", "baz", "qux", "foo"},
		},
		{
			sort:             []string{"foo", "baz"},
			want:             []string{"foo", "baz", "bar", "qux"},
			wantEventSorting: []string{"foo", "baz"},
		},
		{
			sort:             []string{"abc", "foo", "xyz", "baz", "xxx", "qux"},
			want:             []string{"foo", "baz", "qux", "bar"},
			wantEventSorting: []string{"foo", "baz", "qux"},
		},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.sort, ","), func(t *testing.T) {
			tree, _ := nav.Create("foo", items...)
			tree.Sort(tt.sort)

			for i, id := range tt.want {
				item := tree.Items[i]
				if item.ID != id {
					t.Fatalf("tree.Items[%d] should be %q; is %q", i, id, item.ID)
				}
			}

			if !tt.noChange {
				test.Change(t, tree, nav.Sorted, test.EventData(nav.SortedData{Sorting: tt.wantEventSorting}))
			}
		})
	}
}

func TestNav_SortAt(t *testing.T) {
	items := []nav.Item{
		nav.NewLabel("foo", "Foo"),
		nav.NewLabel("bar", "Bar", nav.SubTree(
			nav.NewLabel("baz", "Baz", nav.SubTree(
				nav.NewLabel("foo", "Foo"),
				nav.NewLabel("bar", "Bar"),
				nav.NewLabel("baz", "Baz"),
				nav.NewLabel("qux", "Qux"),
			)),
		)),
	}

	tests := []struct {
		sort             []string
		want             []string
		wantEventSorting []string
		noChange         bool
	}{
		{
			want:     []string{"foo", "bar", "baz", "qux"},
			noChange: true,
		},
		{
			sort:     []string{},
			want:     []string{"foo", "bar", "baz", "qux"},
			noChange: true,
		},
		{
			sort:     []string{"foo", "bar", "baz", "qux"},
			want:     []string{"foo", "bar", "baz", "qux"},
			noChange: true,
		},
		{
			sort:             []string{"bar", "baz", "qux", "foo"},
			want:             []string{"bar", "baz", "qux", "foo"},
			wantEventSorting: []string{"bar", "baz", "qux", "foo"},
		},
		{
			sort:             []string{"foo", "baz"},
			want:             []string{"foo", "baz", "bar", "qux"},
			wantEventSorting: []string{"foo", "baz"},
		},
		{
			sort:             []string{"abc", "foo", "xyz", "baz", "xxx", "qux"},
			want:             []string{"foo", "baz", "qux", "bar"},
			wantEventSorting: []string{"foo", "baz", "qux"},
		},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.sort, ","), func(t *testing.T) {
			tree, _ := nav.Create("foo", items...)
			tree.SortAt("bar.baz", tt.sort)

			item, err := tree.Item("bar.baz")
			if err != nil {
				t.Fatalf("Item(%q) failed with %q", "bar.baz", err)
			}

			for i, id := range tt.want {
				titem := item.Tree.Items[i]
				if titem.ID != id {
					t.Fatalf("item.Tree.Items[%d] should be %q; is %q", i, id, titem.ID)
				}
			}

			if !tt.noChange {
				test.Change(t, tree, nav.Sorted, test.EventData(nav.SortedData{
					Sorting: tt.wantEventSorting,
					Path:    "bar.baz",
				}))
			}
		})
	}
}

func TestNav_Item(t *testing.T) {
	foo := nav.NewLabel("foo", "Foo")
	bar := nav.NewLabel("bar", "Bar", nav.SubTree(
		nav.NewLabel("foo", "foo"),
		nav.NewLabel("bar", "bar", nav.SubTree(
			nav.NewLabel("foo", "Foo"),
			nav.NewLabel("bar", "Bar"),
		)),
		nav.NewLabel("baz", "Baz"),
	))

	navi, err := nav.Create("foo", foo, bar)
	if err != nil {
		t.Fatalf("Create failed with %q", err)
	}

	tests := []struct {
		path         string
		wantNotFound bool
	}{
		{path: "", wantNotFound: true},
		{path: "foo"},
		{path: "foo.bar", wantNotFound: true},
		{path: "bar"},
		{path: "bar.baz"},
		{path: "bar.foo"},
		{path: "bar.bar"},
		{path: "bar.bar.foo"},
		{path: "bar.bar.bar"},
		{path: "bar.bar.baz", wantNotFound: true},
		{path: "bar.bar.foo.foo", wantNotFound: true},
		{path: "baz", wantNotFound: true},
		{path: "foo.", wantNotFound: true},
		{path: "baz.bar.", wantNotFound: true},
		{path: ".baz", wantNotFound: true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			item, err := navi.Item(tt.path)

			if tt.wantNotFound {
				if !errors.Is(err, nav.ErrItemNotFound) {
					t.Fatalf("Item(%q) should fail with %q; got %q", tt.path, nav.ErrItemNotFound, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Item(%q) failed with %q", tt.path, err)
			}

			ids := strings.Split(tt.path, ".")
			last := ids[len(ids)-1]

			if last != item.ID {
				t.Fatalf("ItemID should be %q; is %q", last, item.ID)
			}
		})
	}
}

func initialItems(items ...nav.Item) []nav.Item {
	out := make([]nav.Item, len(items))
	for i, item := range items {
		out[i] = initialItem(item)
	}
	return out
}

func initialItem(item nav.Item) nav.Item {
	item.Initial = true
	return item
}
