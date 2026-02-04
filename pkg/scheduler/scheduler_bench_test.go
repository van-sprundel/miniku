package scheduler

import (
	"fmt"
	"miniku/pkg/store"
	"miniku/pkg/types"
	"testing"
)

func BenchmarkPickNode(b *testing.B) {
	sizes := []int{3, 10, 100}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("nodes=%d", size), func(b *testing.B) {
			podStore := store.NewMemStore[types.Pod]()
			nodeStore := store.NewMemStore[types.Node]()

			for i := range size {
				nodeStore.Put(fmt.Sprintf("node-%d", i), types.Node{
					Name:   fmt.Sprintf("node-%d", i),
					Status: types.NodeStateReady,
				})
			}

			sched := New(podStore, nodeStore)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sched.pickNode()
			}
		})
	}
}

func BenchmarkScheduleOne(b *testing.B) {
	podStore := store.NewMemStore[types.Pod]()
	nodeStore := store.NewMemStore[types.Node]()

	nodeStore.Put("node-1", types.Node{Name: "node-1", Status: types.NodeStateReady})
	nodeStore.Put("node-2", types.Node{Name: "node-2", Status: types.NodeStateReady})

	sched := New(podStore, nodeStore)

	for i := 0; b.Loop(); i++ {
		pod := types.Pod{
			Spec:   types.PodSpec{Name: fmt.Sprintf("pod-%d", i)},
			Status: types.PodStatusPending,
		}
		podStore.Put(pod.Spec.Name, pod)
		_ = sched.scheduleOne(pod)
	}
}

func BenchmarkGetAvailableNodes(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("nodes=%d", size), func(b *testing.B) {
			podStore := store.NewMemStore[types.Pod]()
			nodeStore := store.NewMemStore[types.Node]()

			// half ready, half not ready
			for i := range size {
				status := types.NodeStateReady
				if i%2 == 0 {
					status = types.NodeStateNotReady
				}
				nodeStore.Put(fmt.Sprintf("node-%d", i), types.Node{
					Name:   fmt.Sprintf("node-%d", i),
					Status: status,
				})
			}

			sched := New(podStore, nodeStore)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = sched.getAvailableNodes()
			}
		})
	}
}
