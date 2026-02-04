package store

import "miniku/pkg/types"

type Store[T any] interface {
	List() []T
	Get(name string) (T, bool)
	Put(name string, t T)
	Delete(name string)
}

type PodStore = Store[types.Pod]
type ReplicaSetStore = Store[types.ReplicaSet]
