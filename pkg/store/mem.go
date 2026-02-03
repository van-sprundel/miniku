package store

import (
	"miniku/pkg/types"
	"sync"
)

type MemStore struct {
	mu   sync.RWMutex
	pods map[string]types.Pod
}

func NewMemStore() *MemStore {
	return &MemStore{pods: make(map[string]types.Pod)}
}

func (m *MemStore) List() []types.Pod {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]types.Pod, 0, len(m.pods))
	for _, p := range m.pods {
		out = append(out, p)
	}
	return out
}

func (m *MemStore) Get(name string) (types.Pod, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.pods[name]
	return p, ok
}

func (m *MemStore) Put(pod types.Pod) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pods[pod.Spec.Name] = pod
}

func (m *MemStore) Delete(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.pods, name)
}
