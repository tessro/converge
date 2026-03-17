# converge

A CLI for continuous alignment between humans and agents.

Converge lets you define your exported API surface in **plan files** using
plain Go, then verifies that generated implementations match exactly. Humans
write the spec; agents write the code; converge keeps them in sync.

## Install

```
go install github.com/tessro/converge/cmd/converge@latest
```

## How it works

A converge project splits each package into two files:

- **`_plan.go`** (`//go:build plan`) — the spec. Defines types, function
  signatures, and `converge.Imagine()` descriptions of intended behavior.
- **`_gen.go`** (`//go:build !plan`) — the implementation. Must export the
  exact same types and signatures as the plan.

```
mypackage/
├── mypackage_plan.go    # what it should do
└── mypackage_gen.go     # what it does
```

The two files are mutually exclusive via build tags. `converge check` extracts
the exported API surface from each build and diffs them — any mismatch is an
error.

## Example

Here's a todo list package built in this style (see `example/todo/`):

**todo_plan.go** — the spec:

```go
//go:build plan

package todo

import "github.com/tessro/converge"

type Item struct {
    ID    int
    Title string
    Done  bool
}

type Store struct {
    items  []Item
    nextID int
}

func NewStore() *Store {
    converge.Imagine("return a new Store with an empty item list and nextID starting at 1")
    return nil
}

func (s *Store) Add(title string) Item {
    converge.Imagine("assign the next ID, append a new Item to the store, increment nextID, and return the item")
    return Item{}
}

func (s *Store) Complete(id int) error {
    converge.Imagine("find the item by ID and set Done to true; return an error if not found")
    return nil
}
```

**todo_gen.go** — the implementation:

```go
//go:build !plan

package todo

import "fmt"

type Item struct {
    ID    int
    Title string
    Done  bool
}

type Store struct {
    items  []Item
    nextID int
}

func NewStore() *Store {
    return &Store{nextID: 1}
}

func (s *Store) Add(title string) Item {
    item := Item{ID: s.nextID, Title: title}
    s.nextID++
    s.items = append(s.items, item)
    return item
}

func (s *Store) Complete(id int) error {
    for i := range s.items {
        if s.items[i].ID == id {
            s.items[i].Done = true
            return nil
        }
    }
    return fmt.Errorf("item %d not found", id)
}
```

Running `converge check` verifies alignment:

```
$ converge check ./example/todo
ok — 1 package checked, all exports match
```

If the implementation drifts — say a function is missing or a signature
changes — converge reports exactly what's wrong:

```
=== package todo (./example/todo) ===

  MISSING (in plan, not yet implemented):

    func (s *Store) Complete(id int) error
        todo_plan.go:28
        "find the item by ID and set Done to true; return an error if not found"

FAIL — 1 of 1 package has issues
```

## CLI

### `converge check [dir]`

Compare exported signatures between plan and impl builds. Walks the directory
tree, finds packages with `_plan.go` files, and diffs the API surfaces.

Exits 0 if they match, 1 if they differ.

### `converge lint [-fix] [dir]`

Check that converge conventions are followed:

- `converge` package is only imported in `*_plan.go` files
- Plan files have `//go:build plan`
- File naming is consistent

Pass `-fix` to auto-add missing build tags.

## The `converge` package

The library provides a single function:

```go
import "github.com/tessro/converge"

converge.Imagine("description of intended behavior")
```

`Imagine` is a no-op. It exists so that plan files compile and so that
`converge check` can extract the description and include it in its output.
The description is the spec — it tells the code generator what to build.

## File conventions

| Suffix | Build tag | Purpose |
|---|---|---|
| `_plan.go` | `//go:build plan` | API spec with Imagine descriptions |
| `_gen.go` | `//go:build !plan` | Generated implementation |

Both files define the complete package — types, functions, everything. They
are fully divergent; there are no shared files. `converge check` ensures the
exported surface stays in sync.

## License

MIT
