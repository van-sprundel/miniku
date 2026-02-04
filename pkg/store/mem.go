package store

import (
	"sync"
)

type MemStore[T any] struct {
	mu   sync.RWMutex
	data map[string]T
}

func NewMemStore[T any]() *MemStore[T] {
	return &MemStore[T]{
		data: make(map[string]T),
	}
}

func (m *MemStore[T]) List() []T {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]T, 0, len(m.data))
	for _, item := range m.data {
		out = append(out, item)
	}
	return out
}

func (m *MemStore[T]) Get(name string) (T, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	item, ok := m.data[name]
	return item, ok
}

func (m *MemStore[T]) Put(name string, t T) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[name] = t
}

func (m *MemStore[T]) Delete(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, name)
}
