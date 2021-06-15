# Static web pages

Static page are the bare minimum pages that a website needs to be complete
(home, about, contact etc.). They have mostly non-generic designs that don't
allow for its content to be edited willy-nilly, because the content of such
pages is often tightly aligned with their design. This makes it hard to give
website owners the possibility to edit static content themselves.

Goal of this module is to provide developers with a framework to define static
pages in a way that enables them to provide content editing capabilities to
website owners while simultaneously not getting in the way of the developer
during the development process.

## Requirements

- simple & quick page structure definition
- various field types (text, number etc.)
- support for localization
- uneditable fields
- seamless integration into development & deployment process

## Design

Developers define static pages with fields for different kinds of content or
data, with functions returning the default values for each field:

```go
package example

func NewStaticPages() static.Pages {
	return static.Pages{
		static.NewPage("contact", static.WithField(
			field.NewText("contactEmail", func(f *field.Text) string {
				return "hello@example.com"
			}),

			field.NewLocalizedText("contactPhone", func(f *field.LocalizedText) string {
				switch f.Locale {
				case "en":
					return "+1 234 567 890"
				default:
					return "+49 234 567 890"
				}
			}),

			field.NewLocalizedMeta("meta", func(f *field.LocalizedMeta) meta.Data {
				switch f.Locale {
				case "de":
					return meta.Data{
						Title: "Kontakt",
						Description: "Nehme Kontakt zu uns auf.",
					}
				default:
					return meta.Data{
						Title: "Contact us",
						Description: "Let's get in touch.",
					}
				}
			}),
		)),

		static.NewPage("legal", static.WithField(
			field.NewToggle("showCompanyName", func(f *field.Toggle) bool {
				return true
			}),
		)),

		static.NewPage("pricing", static.WithField(
			field.NewMoney("basic", func(f *field.Money) money.Money {
				return money.USD(2999)
			}),
			field.NewMoney("plus", func(f *field.Money) money.Money {
				return money.USD(5999)
			}),
			field.NewMoney("premium", func(f *field.Money) money.Money {
				return money.USD(9999)
			}),
			field.NewMeta("meta", func(f *field.Meta) meta.Data {
				return meta.Data{
					Title: "Pricing",
					Description: "View our pricing options.",
				}
			}),
		)),
	)
}
```

The example above shows approximately how a developer would define static pages.
The toolkit needs to provide the most used field types, including:

- `TextField` & `LocalizedTextField`
- `Meta` & `LocalizedMeta`
- `Money` (& `LocalizedMoney` ?)
- `Toggle`
- 

### Page server

Creating an HTTP server for the pages should be as simple as:

```go
package example

func NewPageServer(pages static.Pages) http.Handler {
	return static.NewHTTPServer(pages)
}
```

```sh
GET  /pages/{PageName} # fetch a page
PATCH /pages/{PageName} # update multiple fields of a page
PUT /pages/{PageName}/{FieldName} # update a specific field
DELETE /pages/{PageName} # reset page to defaults (defined by developer)
```

## Frontend tooling

An npm package must provide access to static page management:

```ts
import { connect, MoneyField, formatMoney } from '@nice-cms/static'

const pages = await connect('http://localhost:8000')

const pricing = await pages.fetch<{
	base: MoneyField
	plus: MoneyField
	premium: MoneyField
}>('pricing')

console.log('Fetched prices:')
console.log(`Base ${formatMoney(pricing.base)}`)
console.log(`Plus ${formatMoney(pricing.plus)}`)
console.log(`Premium ${formatMoney(pricing.premium)}`)
```

### Schema definition (optional)

Because web developers work most of the time on the frontend inside a JS
ecosystem, it may be wise to let them define page schemas directly in
JavaScript:

```ts
import {
	connect,
	definePage,
	textField,
	localizedTextField,
	localizedMetaField,
} from '@nice-cms/static'

const schema = definePage(
	'contact',

	textField('contactEmail', 'hello@example.com'),

	localizedTextField('contactPhone', field => {
		switch (field.locale) {
		case 'de':
			return '+49 123 456 789'
		default:
			return '+1 234 567 890'
		}
	}),

	localizedMetaField('meta', field => {
		switch (field.locale) {
		case 'de':
			return {
				title: 'Kontakt',
				description: 'Nimm Kontakt zu uns auf.',
			}
		default:
			return {
				title: 'Contact',
				description: 'Get in touch.',
			}
		}
	}),
)

const pages = await connect('http://localhost:8000')

const pricing = await pages.fetch(schema)

console.log('Fetched prices:')
console.log(`Base ${formatMoney(pricing.base)}`)
console.log(`Plus ${formatMoney(pricing.plus)}`)
console.log(`Premium ${formatMoney(pricing.premium)}`)
```
