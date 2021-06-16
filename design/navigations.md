# Navigations

## Introduction

Navigations are typically hard-coded into the code of a website. In order for
website owners to be able to edit navigation trees, there needs to be a unified
way for both developers and website owners to manage navigations.

## Design

A tree is a recursive list, where each list item may itself be its own list.

A navigation item may be of one of the following types:

- link to static page
  - without subtree
  - with subtree
- link to blog post (without subtree)
- label only (with subtree)

A navigation tree must be identified by a **unique name** (string).

Developers need the ability to **guard navigations** from being deleted by
website owners to ensure availability of navigation trees that are required for
a website to work.

On the frontend, developers must be able to query navigation trees by their name
to

- build the view for website users.
- build the management for website owners.

Developers do not have to code data structures to implement navigation trees,
because navigation tree logic doesn't need to be customizable.

## Initial navigations

When a developer builds a website, they most likely want to hard-code the
initial navigation trees. Forcing developers to create navigations through an
admin UI would reduce development experience, so developers must be able to
hard-code the initial navigations through code, while allowing website owners to
update those navigation trees within an admin UI if they need to.

```go
package example

func NewNavigations(blog *blog.Blog) static.Navigations {
  return static.Navigations{
    nav.NewTree("main", func(nav *nav.Tree) nav.Items {
      return nav.Items{
        nav.NewStaticLink("contact", func(l *nav.StaticLink) string {
          switch l.Locale {
          case "de":
            return "Kontakt"
          default:
            return "Contact"
          }
        }, nav.WithFieldOptions(field.Localized())),

        nav.NewStaticLink("about", func(l *nav.StaticLink) string {
          return "About"
        }, nav.WithSubNav(nav.NewSubTree(func(nav *nav.Tree) nav.Items {
          return nav.Items{
            nav.NewStaticLink("legal", func(l *nav.StaticLink) string {
              switch l.Locale {
              case "de":
                return "Impressum"
              default:
                return "Legal"
              }
            }),
          }
        }, nav.WithFieldOptions(field.Localized()))),

        nav.NewStaticLink("pricing", func(l *nav.StaticLink) string {
          switch l.Locale {
          case "de":
            return "Preise"
          default:
            return "Pricing"
          }
        }, nav.WithFieldOptions(field.Localized())),

        nav.NewBlogLink(blog, "post-id", func(l *nav.BlogLink) string {
          return l.Post.Field("shortTitle").Localize(l.Locale)
        }),

        nav.NewLabel(func(l *nav.Label) string {
          switch l.Locale {
          case "de":
            return "Nicht klickbar!"
          default:
            return "Unclickable!"
          }
        }, nav.WithFieldOptions(field.Localized())),
      }
    }),

    nav.NewTree("footer", func(nav *nav.Tree) nav.Items {
      return nav.Items{
        nav.NewStaticLink("legal", func(l *nav.StaticLink) string {
          switch l.Locale {
          case "de":
            return "Impressum"
          default:
            return "Legal"
          }
        }, nav.WithFieldOptions(field.Localized())),
      }
    })
  }
}
```

```go
package example

func NewPageServer(pages static.Pages, navs static.Navigations) http.Handler {
  return static.NewHTTPServer(pages, static.WithNavigations(navs))
}
```

## Frontend tooling

```ts
import { connect, StaticLink, NavigationTree } from '@nice-cms/static'

const static = await connect('http://localhost:8000')

const mainNav = await static.navigations.fetch('main')

console.log('Fetched "main" navigation.\n')

printItems(mainNav)

function printItems(nav: NavigationTree, indent = 0) {
  const spaces = new Array(indent).fill(' ')

  for (const item of mainNav.items) {
    console.log(spaces)
    console.log(`${spaces}Navigation item: ${item.label}`)
    if (item.link) {
      console.log(`${spaces}Item links to: ${item.link.path} (${item.link.type})`)
    }
    if (item.tree) {
      console.log(`${spaces}Item has sub-tree:`)
      printItems(item.tree, indent+4)
    }
  }
}
```
