# Pages

## Create page

```go
package example

p, err := page.Create("foo")
```

### Create page with fields

```go
package example

p, err := page.Create(
	"contact",

	field.NewText("contactPhone", "+1 234 567 890", text.Localize("+49 123 456 789", "de", "ch", "at")),
	field.NewToggle("showPhone", false, toggle.Localize(true, "en", "de", "ch", "at")),

	field.NewMeta("meta", meta.Data{
		Title: "Contact us",
		Description: "Schedule a call and stuff.",
	}, meta.Localize(meta.Data{
		Title: "Kontakt aufnehmen",
		Description: "Nimm Kontakt zu uns auf.",
	}, "de", "ch", "at")),
)

p, err := page.Create(
	"pricing",

	field.NewMoney("basic", money.New(2995, "USD"), money.Localize(money.New(2495, "EUR"), "de", "ch", "at")),
	field.NewMoney("plus", money.New(5995, "USD"), money.Localize(money.New(4995, "EUR"), "de", "ch", "at")),
	field.NewMoney("premium", money.New(9999, "USD"), money.Localize(money.New(8495, "EUR"), "de", "ch", "at")),
)
```

## Fields

```go
package example

text := field.NewText("name", "Default")
num := field.NewInteger("name", 500)
float := field.NewFloat("name", 10.38)
money := field.NewMoney("name", money.New(1995, "USD"))
meta := field.NewMeta("name", meta.Data{
	Title: "Title",
	Description: "Description",
})
toggle := field.NewToggle("name", true)
```

### Field value

```go
package example

text := field.NewText("name", "Default", text.Localize("Hallo", "de"))
localized := text.Value("en") // "Default"
localized := text.Value("de") // "Hallo"

num := field.NewInteger("name", 100, integer.Localize(200, "de"))
localized := num.Value("en") // 100
localized := num.Value("de") // 200
```
