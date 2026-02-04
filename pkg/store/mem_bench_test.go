package store

import (
	"fmt"
	"testing"
)

type testItem struct {
	Name  string
	Value int
}

func BenchmarkMemStorePut(b *testing.B) {
	store := NewMemStore[testItem]()
	item := testItem{Name: "test", Value: 42}

	for i := 0; b.Loop(); i++ {
		store.Put(fmt.Sprintf("item-%d", i), item)
	}
}

func BenchmarkMemStoreGet(b *testing.B) {
	store := NewMemStore[testItem]()
	for i := range 1000 {
		store.Put(fmt.Sprintf("item-%d", i), testItem{Name: fmt.Sprintf("test-%d", i), Value: i})
	}

	for i := 0; b.Loop(); i++ {
		store.Get(fmt.Sprintf("item-%d", i%1000))
	}
}

func BenchmarkMemStoreList(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			store := NewMemStore[testItem]()
			for i := range size {
				store.Put(fmt.Sprintf("item-%d", i), testItem{Name: fmt.Sprintf("test-%d", i), Value: i})
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = store.List()
			}
		})
	}
}

func BenchmarkMemStoreDelete(b *testing.B) {
	for b.Loop() {
		b.StopTimer()
		store := NewMemStore[testItem]()
		store.Put("item", testItem{Name: "test", Value: 42})
		b.StartTimer()

		store.Delete("item")
	}
}
