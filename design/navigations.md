# Navigations

## Introduction

Navigations are typically hard-coded into the code of a website. In order for
website owners to be able to change navigation trees, there needs to be a
unified way for both developers and website owners to manage navigations.

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
update those navigation trees if they need to.
