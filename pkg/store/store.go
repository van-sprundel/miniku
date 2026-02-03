package store

import "miniku/pkg/types"

type Store interface {
	List() []types.Pod
	Get(name string) (types.Pod, bool)
	Put(pod types.Pod)
	Delete(name string)
}
