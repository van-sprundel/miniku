package store

import (
	"fmt"
	"testing"
)

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
	store := NewMemStore[testItem]()
	// pre-populate
	for i := range 1000 {
		store.Put(fmt.Sprintf("item-%d", i), testItem{Name: "test", Value: i})
	}

	for i := 0; b.Loop(); i++ {
		store.Delete(fmt.Sprintf("item-%d", i%1000))
		store.Put(fmt.Sprintf("item-%d", i%1000), testItem{Name: "test", Value: i})
	}
}

func BenchmarkMemStoreConcurrentReads(b *testing.B) {
	store := NewMemStore[testItem]()
	for i := range 1000 {
		store.Put(fmt.Sprintf("item-%d", i), testItem{Name: fmt.Sprintf("test-%d", i), Value: i})
	}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			store.Get(fmt.Sprintf("item-%d", i%1000))
			i++
		}
	})
}

func BenchmarkMemStoreConcurrentWrites(b *testing.B) {
	store := NewMemStore[testItem]()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			store.Put(fmt.Sprintf("item-%d", i%1000), testItem{Name: "test", Value: i})
			i++
		}
	})
}

func BenchmarkMemStoreConcurrentMixed(b *testing.B) {
	store := NewMemStore[testItem]()
	for i := range 1000 {
		store.Put(fmt.Sprintf("item-%d", i), testItem{Name: fmt.Sprintf("test-%d", i), Value: i})
	}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%10 == 0 {
				// 10% writes
				store.Put(fmt.Sprintf("item-%d", i%1000), testItem{Name: "test", Value: i})
			} else {
				// 90% reads
				store.Get(fmt.Sprintf("item-%d", i%1000))
			}
			i++
		}
	})
}

func BenchmarkMemStoreConcurrentList(b *testing.B) {
	sizes := []int{100, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			store := NewMemStore[testItem]()
			for i := range size {
				store.Put(fmt.Sprintf("item-%d", i), testItem{Name: fmt.Sprintf("test-%d", i), Value: i})
			}

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_ = store.List()
				}
			})
		})
	}
}
