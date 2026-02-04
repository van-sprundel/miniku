package scheduler

import (
	"miniku/pkg/store"
	"miniku/pkg/types"
	"testing"
)

func TestScheduleOne(t *testing.T) {
	tests := []struct {
		name           string
		pod            types.Pod
		nodes          []types.Node
		expectAssigned bool
	}{
		{
			name: "assigns pod to available node",
			pod: types.Pod{
				Spec:   types.PodSpec{Name: "test-pod"},
				Status: types.PodStatusPending,
			},
			nodes: []types.Node{
				{Name: "node-1", Status: types.NodeStateReady},
			},
			expectAssigned: true,
		},
		{
			name: "no assignment when no nodes",
			pod: types.Pod{
				Spec:   types.PodSpec{Name: "test-pod"},
				Status: types.PodStatusPending,
			},
			nodes:          []types.Node{},
			expectAssigned: false,
		},
		{
			name: "no assignment when all nodes NotReady",
			pod: types.Pod{
				Spec:   types.PodSpec{Name: "test-pod"},
				Status: types.PodStatusPending,
			},
			nodes: []types.Node{
				{Name: "node-1", Status: types.NodeStateNotReady},
				{Name: "node-2", Status: types.NodeStateNotReady},
			},
			expectAssigned: false,
		},
		{
			name: "assigns to Ready node when mixed",
			pod: types.Pod{
				Spec:   types.PodSpec{Name: "test-pod"},
				Status: types.PodStatusPending,
			},
			nodes: []types.Node{
				{Name: "node-1", Status: types.NodeStateNotReady},
				{Name: "node-2", Status: types.NodeStateReady},
			},
			expectAssigned: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			podStore := store.NewMemStore[types.Pod]()
			nodeStore := store.NewMemStore[types.Node]()

			podStore.Put(tt.pod.Spec.Name, tt.pod)
			for _, node := range tt.nodes {
				nodeStore.Put(node.Name, node)
			}

			sched := New(podStore, nodeStore)
			err := sched.scheduleOne(tt.pod)

			pod, _ := podStore.Get(tt.pod.Spec.Name)

			if tt.expectAssigned {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if pod.Spec.NodeName == "" {
					t.Errorf("expected pod to be assigned to a node")
				}
			} else {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if pod.Spec.NodeName != "" {
					t.Errorf("expected pod to not be assigned, got %s", pod.Spec.NodeName)
				}
			}
		})
	}
}

func TestGetAvailableNodes(t *testing.T) {
	tests := []struct {
		name          string
		nodes         []types.Node
		expectedCount int
	}{
		{
			name:          "no nodes",
			nodes:         []types.Node{},
			expectedCount: 0,
		},
		{
			name: "all ready",
			nodes: []types.Node{
				{Name: "node-1", Status: types.NodeStateReady},
				{Name: "node-2", Status: types.NodeStateReady},
			},
			expectedCount: 2,
		},
		{
			name: "all not ready",
			nodes: []types.Node{
				{Name: "node-1", Status: types.NodeStateNotReady},
				{Name: "node-2", Status: types.NodeStateNotReady},
			},
			expectedCount: 0,
		},
		{
			name: "mixed",
			nodes: []types.Node{
				{Name: "node-1", Status: types.NodeStateReady},
				{Name: "node-2", Status: types.NodeStateNotReady},
				{Name: "node-3", Status: types.NodeStateReady},
			},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			podStore := store.NewMemStore[types.Pod]()
			nodeStore := store.NewMemStore[types.Node]()

			for _, node := range tt.nodes {
				nodeStore.Put(node.Name, node)
			}

			sched := New(podStore, nodeStore)
			available := sched.getAvailableNodes()

			if len(available) != tt.expectedCount {
				t.Errorf("expected %d available nodes, got %d", tt.expectedCount, len(available))
			}
		})
	}
}

func TestPickNode(t *testing.T) {
	tests := []struct {
		name     string
		nodes    []types.Node
		expectOk bool
	}{
		{
			name:     "no nodes returns false",
			nodes:    []types.Node{},
			expectOk: false,
		},
		{
			name: "with nodes returns true",
			nodes: []types.Node{
				{Name: "node-1", Status: types.NodeStateReady},
			},
			expectOk: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			podStore := store.NewMemStore[types.Pod]()
			nodeStore := store.NewMemStore[types.Node]()

			for _, node := range tt.nodes {
				nodeStore.Put(node.Name, node)
			}

			sched := New(podStore, nodeStore)
			_, ok := sched.pickNode()

			if ok != tt.expectOk {
				t.Errorf("expected ok=%v, got %v", tt.expectOk, ok)
			}
		})
	}
}

func TestSchedulerSkipsAlreadyScheduledPods(t *testing.T) {
	podStore := store.NewMemStore[types.Pod]()
	nodeStore := store.NewMemStore[types.Node]()

	// pod alr assigned to node-1
	pod := types.Pod{
		Spec:   types.PodSpec{Name: "test-pod", NodeName: "node-1"},
		Status: types.PodStatusPending,
	}
	podStore.Put(pod.Spec.Name, pod)

	nodeStore.Put("node-2", types.Node{Name: "node-2", Status: types.NodeStateReady})

	sched := New(podStore, nodeStore)

	// manually call what `Run()` does for one iteration
	for _, p := range podStore.List() {
		if p.Spec.NodeName == "" {
			sched.scheduleOne(p)
		}
	}

	// pod should still be on node-1
	result, _ := podStore.Get("test-pod")
	if result.Spec.NodeName != "node-1" {
		t.Errorf("expected pod to stay on node-1, got %s", result.Spec.NodeName)
	}
}
