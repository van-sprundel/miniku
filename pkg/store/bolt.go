package store

import (
	"encoding/json"
	"log"
	"sync"

	bolt "go.etcd.io/bbolt"
)

type BoltStore[T any] struct {
	mu     sync.RWMutex
	db     *bolt.DB
	bucket []byte
}

func NewBoltStore[T any](db *bolt.DB, bucket string) *BoltStore[T] {
	if err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucket))
		return err
	}); err != nil {
		log.Fatalf("bolt: failed to create bucket %q: %v", bucket, err)
	}

	return &BoltStore[T]{
		db:     db,
		bucket: []byte(bucket),
	}
}

func (s *BoltStore[T]) List() []T {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []T
	if err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(s.bucket)
		return b.ForEach(func(k, v []byte) error {
			var item T
			if err := json.Unmarshal(v, &item); err != nil {
				return err
			}
			out = append(out, item)
			return nil
		})
	}); err != nil {
		log.Printf("bolt: list %q: %v", s.bucket, err)
	}
	return out
}

func (s *BoltStore[T]) Get(name string) (T, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var item T
	var found bool
	if err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(s.bucket)
		v := b.Get([]byte(name))
		if v == nil {
			return nil
		}
		if err := json.Unmarshal(v, &item); err != nil {
			return err
		}
		found = true
		return nil
	}); err != nil {
		log.Printf("bolt: get %q/%s: %v", s.bucket, name, err)
	}
	return item, found
}

func (s *BoltStore[T]) Put(name string, t T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(s.bucket)
		data, err := json.Marshal(t)
		if err != nil {
			return err
		}
		return b.Put([]byte(name), data)
	}); err != nil {
		log.Printf("bolt: put %q/%s: %v", s.bucket, name, err)
	}
}

func (s *BoltStore[T]) Delete(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(s.bucket)
		return b.Delete([]byte(name))
	}); err != nil {
		log.Printf("bolt: delete %q/%s: %v", s.bucket, name, err)
	}
}
