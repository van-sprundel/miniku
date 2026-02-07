package store

import (
	"path/filepath"
	"testing"

	bolt "go.etcd.io/bbolt"
)

type testItem struct {
	Name  string
	Value int
}

// storeFactory creates a fresh store for testing.
type storeFactory func(t *testing.T) Store[testItem]

func memFactory(t *testing.T) Store[testItem] {
	t.Helper()
	return NewMemStore[testItem]()
}

func boltFactory(t *testing.T) Store[testItem] {
	t.Helper()
	db, err := bolt.Open(filepath.Join(t.TempDir(), "test.db"), 0o600, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewBoltStore[testItem](db, "test")
}

func runStoreTests(t *testing.T, name string, factory storeFactory) {
	t.Run(name, func(t *testing.T) {
		t.Run("PutAndGet", func(t *testing.T) {
			s := factory(t)
			s.Put("a", testItem{Name: "a", Value: 1})

			got, ok := s.Get("a")
			if !ok {
				t.Fatal("expected item to be found")
			}
			if got.Name != "a" || got.Value != 1 {
				t.Errorf("got %+v, want {a 1}", got)
			}
		})

		t.Run("GetMissing", func(t *testing.T) {
			s := factory(t)
			_, ok := s.Get("missing")
			if ok {
				t.Error("expected not found")
			}
		})

		t.Run("PutOverwrite", func(t *testing.T) {
			s := factory(t)
			s.Put("a", testItem{Name: "a", Value: 1})
			s.Put("a", testItem{Name: "a", Value: 2})

			got, ok := s.Get("a")
			if !ok {
				t.Fatal("expected item to be found")
			}
			if got.Value != 2 {
				t.Errorf("got Value=%d, want 2", got.Value)
			}
		})

		t.Run("Delete", func(t *testing.T) {
			s := factory(t)
			s.Put("a", testItem{Name: "a", Value: 1})
			s.Delete("a")

			_, ok := s.Get("a")
			if ok {
				t.Error("expected item to be deleted")
			}
		})

		t.Run("DeleteMissing", func(t *testing.T) {
			s := factory(t)
			s.Delete("nope") // should not panic
		})

		t.Run("ListEmpty", func(t *testing.T) {
			s := factory(t)
			items := s.List()
			if len(items) != 0 {
				t.Errorf("expected empty list, got %d items", len(items))
			}
		})

		t.Run("ListMultiple", func(t *testing.T) {
			s := factory(t)
			s.Put("a", testItem{Name: "a", Value: 1})
			s.Put("b", testItem{Name: "b", Value: 2})
			s.Put("c", testItem{Name: "c", Value: 3})

			items := s.List()
			if len(items) != 3 {
				t.Fatalf("expected 3 items, got %d", len(items))
			}

			seen := map[string]bool{}
			for _, item := range items {
				seen[item.Name] = true
			}
			for _, name := range []string{"a", "b", "c"} {
				if !seen[name] {
					t.Errorf("missing item %q in list", name)
				}
			}
		})

		t.Run("ListAfterDelete", func(t *testing.T) {
			s := factory(t)
			s.Put("a", testItem{Name: "a", Value: 1})
			s.Put("b", testItem{Name: "b", Value: 2})
			s.Delete("a")

			items := s.List()
			if len(items) != 1 {
				t.Fatalf("expected 1 item, got %d", len(items))
			}
			if items[0].Name != "b" {
				t.Errorf("expected item b, got %s", items[0].Name)
			}
		})
	})
}

func TestMemStore(t *testing.T) {
	runStoreTests(t, "MemStore", memFactory)
}

func TestBoltStore(t *testing.T) {
	runStoreTests(t, "BoltStore", boltFactory)
}
