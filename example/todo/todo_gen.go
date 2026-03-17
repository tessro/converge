//go:build !plan

// Package todo is a simple in-memory todo list demonstrating the
// converge plan/gen pattern.
package todo

import "fmt"

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
	return &Store{nextID: 1}
}

// Add creates a new todo item and returns it.
func (s *Store) Add(title string) Item {
	item := Item{ID: s.nextID, Title: title}
	s.nextID++
	s.items = append(s.items, item)
	return item
}

// List returns all todo items.
func (s *Store) List() []Item {
	out := make([]Item, len(s.items))
	copy(out, s.items)
	return out
}

// Get returns a single item by ID.
func (s *Store) Get(id int) (Item, error) {
	for _, item := range s.items {
		if item.ID == id {
			return item, nil
		}
	}
	return Item{}, fmt.Errorf("item %d not found", id)
}

// Complete marks an item as done.
func (s *Store) Complete(id int) error {
	for i := range s.items {
		if s.items[i].ID == id {
			s.items[i].Done = true
			return nil
		}
	}
	return fmt.Errorf("item %d not found", id)
}

// Delete removes an item from the store.
func (s *Store) Delete(id int) error {
	for i, item := range s.items {
		if item.ID == id {
			s.items = append(s.items[:i], s.items[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("item %d not found", id)
}
