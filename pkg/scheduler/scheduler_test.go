package scheduler

import (
	"miniku/pkg/testutil"
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
			env := testutil.NewTestEnv()
			defer env.Close()

			env.PodStore.Put(tt.pod.Spec.Name, tt.pod)
			for _, node := range tt.nodes {
				env.NodeStore.Put(node.Name, node)
			}

			sched := New(env.Client)
			err := sched.scheduleOne(tt.pod)

			pod, _ := env.PodStore.Get(tt.pod.Spec.Name)

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
			env := testutil.NewTestEnv()
			defer env.Close()

			for _, node := range tt.nodes {
				env.NodeStore.Put(node.Name, node)
			}

			sched := New(env.Client)
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
			env := testutil.NewTestEnv()
			defer env.Close()

			for _, node := range tt.nodes {
				env.NodeStore.Put(node.Name, node)
			}

			sched := New(env.Client)
			_, ok := sched.pickNode()

			if ok != tt.expectOk {
				t.Errorf("expected ok=%v, got %v", tt.expectOk, ok)
			}
		})
	}
}

func TestPickNodeRoundRobin(t *testing.T) {
	env := testutil.NewTestEnv()
	defer env.Close()

	env.NodeStore.Put("node-a", types.Node{Name: "node-a", Status: types.NodeStateReady})
	env.NodeStore.Put("node-b", types.Node{Name: "node-b", Status: types.NodeStateReady})
	env.NodeStore.Put("node-c", types.Node{Name: "node-c", Status: types.NodeStateReady})

	sched := New(env.Client)

	// pick 6 nodes, should round-robin through a, b, c twice
	seen := make([]string, 6)
	for i := range 6 {
		node, ok := sched.pickNode()
		if !ok {
			t.Fatal("expected node to be picked")
		}
		seen[i] = node.Name
	}

	// sorted order: node-a, node-b, node-c
	expected := []string{"node-a", "node-b", "node-c", "node-a", "node-b", "node-c"}
	for i, name := range expected {
		if seen[i] != name {
			t.Errorf("pick %d: got %s, want %s", i, seen[i], name)
		}
	}
}

func TestSchedulerSkipsAlreadyScheduledPods(t *testing.T) {
	env := testutil.NewTestEnv()
	defer env.Close()

	// pod alr assigned to node-1
	pod := types.Pod{
		Spec:   types.PodSpec{Name: "test-pod", NodeName: "node-1"},
		Status: types.PodStatusPending,
	}
	env.PodStore.Put(pod.Spec.Name, pod)

	env.NodeStore.Put("node-2", types.Node{Name: "node-2", Status: types.NodeStateReady})

	sched := New(env.Client)

	// manually call what `Run()` does for one iteration
	pods, err := sched.client.ListPods()
	if err != nil {
		t.Fatalf("ListPods failed: %v", err)
	}
	for _, p := range pods {
		if p.Spec.NodeName == "" {
			if err := sched.scheduleOne(p); err != nil {
				t.Fatalf("scheduleOne failed: %v", err)
			}
		}
	}

	// pod should still be on node-1
	result, _ := env.PodStore.Get("test-pod")
	if result.Spec.NodeName != "node-1" {
		t.Errorf("expected pod to stay on node-1, got %s", result.Spec.NodeName)
	}
}
