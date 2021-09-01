# Page

A `Page` is a web page of a website. A page is identified by a unique name and
is composed of fields of different types.

## Create a page

```go
package example

func createPage() {
	p := page.New(uuid.New())

	err := p.Create("home")
}
```

## Add fields to a page

```go
package example

func addFields() {
	var p *page.Page

	err := p.AddFields(
		field.NewText(
			"headline", // field id
			"This is the default headline of the home page.", // default value
			text.Localize("Das ist die Standardüberschrift.", "de", "ch", "at"), // default value for specific languages
		),
		field.NewToggle(
			"headerVisible",
			true,
			toggle.Localize(false, "de", "ch", "at"),
		),
		field.NewMoney(
			"price",
			money.USD(50000),
			money.Localize(money.EUR(35000), "de", "ch", "at"),
		),
	)
}
```

## Update field values

```go
package example

func updateField() {
	var p *page.Page
	var cfg page.FieldConfig // field provides utilities to parse and format field values

	err := p.UpdateField(cfg, "headline", "New headline") // update for all languages
	err := p.UpdateField(cfg, "headline", "Neue Überschrift", "de", "ch", "at") // update for specific languages

	err := p.UpdateField(cfg, "headerVisible", true)
	err := p.UpdateField(cfg, "headerVisible", false, "de", "ch", "at")
}
```

## Remove a field

```go
package example

func removeField() {
	var p *page.Page

	err := p.RemoveField("headline")
}
```
