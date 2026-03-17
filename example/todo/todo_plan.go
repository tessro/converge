//go:build plan

// Package todo is a simple in-memory todo list demonstrating the
// converge plan/gen pattern.
package todo

import "github.com/tessro/converge"

// Item represents a single todo item.
type Item struct {
	ID    int
	Title string
	Done  bool
}

// Store holds todo items in memory.
type Store struct {
	items  []Item
	nextID int
}

// NewStore returns an empty Store ready for use.
func NewStore() *Store {
	converge.Imagine("return a new Store with an empty item list and nextID starting at 1")
	return nil
}

// Add creates a new todo item and returns it.
func (s *Store) Add(title string) Item {
	converge.Imagine("assign the next ID, append a new Item to the store, increment nextID, and return the item")
	return Item{}
}

// List returns all todo items.
func (s *Store) List() []Item {
	converge.Imagine("return a copy of the items slice so callers cannot mutate internal state")
	return nil
}

// Get returns a single item by ID.
func (s *Store) Get(id int) (Item, error) {
	converge.Imagine("find the item with the given ID and return it; return an error if not found")
	return Item{}, nil
}

// Complete marks an item as done.
func (s *Store) Complete(id int) error {
	converge.Imagine("find the item by ID and set Done to true; return an error if not found")
	return nil
}

// Delete removes an item from the store.
func (s *Store) Delete(id int) error {
	converge.Imagine("remove the item with the given ID; return an error if not found")
	return nil
}
