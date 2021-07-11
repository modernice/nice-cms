# Page server

```go
package example

p, _ := page.Create("baz")

setup, err := pageserver.New(
	pageserver.Create("foo", field.NewText("foo", "Foo")),
	pageserver.Create("bar", field.NewText("foo", "Foo")),
	pageserver.Add(p),
)
```
