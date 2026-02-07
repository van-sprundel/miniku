package client

import (
	"miniku/pkg/api"
	"miniku/pkg/store"
	"miniku/pkg/types"
	"net/http/httptest"
	"testing"
)

func setup() (*Client, store.PodStore, store.ReplicaSetStore, store.NodeStore, *httptest.Server) {
	podStore := store.NewMemStore[types.Pod]()
	rsStore := store.NewMemStore[types.ReplicaSet]()
	nodeStore := store.NewMemStore[types.Node]()

	srv := &api.Server{PodStore: podStore, RSStore: rsStore, NodeStore: nodeStore}
	ts := httptest.NewServer(srv.Routes())

	c := New(ts.URL)
	return c, podStore, rsStore, nodeStore, ts
}

func TestPodRoundTrip(t *testing.T) {
	c, podStore, _, _, ts := setup()
	defer ts.Close()

	// create
	pod := types.Pod{
		Spec:   types.PodSpec{Name: "test", Image: "nginx"},
		Status: types.PodStatusPending,
	}
	if err := c.CreatePod(pod); err != nil {
		t.Fatalf("CreatePod: %v", err)
	}

	// verify in store
	if len(podStore.List()) != 1 {
		t.Fatalf("expected 1 pod in store")
	}

	// get
	got, found, err := c.GetPod("test")
	if err != nil {
		t.Fatalf("GetPod: %v", err)
	}
	if !found {
		t.Fatal("expected pod to be found")
	}
	if got.Spec.Name != "test" {
		t.Errorf("got name %q, want %q", got.Spec.Name, "test")
	}

	// list
	pods, err := c.ListPods()
	if err != nil {
		t.Fatalf("ListPods: %v", err)
	}
	if len(pods) != 1 {
		t.Errorf("got %d pods, want 1", len(pods))
	}

	// update
	got.Spec.NodeName = "node-1"
	got.Status = types.PodStatusRunning
	if err := c.UpdatePod("test", got); err != nil {
		t.Fatalf("UpdatePod: %v", err)
	}

	updated, _, _ := c.GetPod("test")
	if updated.Spec.NodeName != "node-1" {
		t.Errorf("got node %q, want %q", updated.Spec.NodeName, "node-1")
	}
	if updated.Status != types.PodStatusRunning {
		t.Errorf("got status %q, want %q", updated.Status, types.PodStatusRunning)
	}

	// delete
	if err := c.DeletePod("test"); err != nil {
		t.Fatalf("DeletePod: %v", err)
	}

	_, found, _ = c.GetPod("test")
	if found {
		t.Error("expected pod to be deleted")
	}
}

func TestReplicaSetRoundTrip(t *testing.T) {
	c, _, rsStore, _, ts := setup()
	defer ts.Close()

	rs := types.ReplicaSet{
		Name:         "web",
		DesiredCount: 3,
		Selector:     map[string]string{"app": "web"},
		Template:     types.PodSpec{Image: "nginx"},
	}
	if err := c.CreateReplicaSet(rs); err != nil {
		t.Fatalf("CreateReplicaSet: %v", err)
	}

	if len(rsStore.List()) != 1 {
		t.Fatalf("expected 1 RS in store")
	}

	got, found, err := c.GetReplicaSet("web")
	if err != nil {
		t.Fatalf("GetReplicaSet: %v", err)
	}
	if !found {
		t.Fatal("expected RS to be found")
	}
	if got.DesiredCount != 3 {
		t.Errorf("got desired %d, want 3", got.DesiredCount)
	}

	rsList, err := c.ListReplicaSets()
	if err != nil {
		t.Fatalf("ListReplicaSets: %v", err)
	}
	if len(rsList) != 1 {
		t.Errorf("got %d RSs, want 1", len(rsList))
	}

	got.DesiredCount = 5
	if err := c.UpdateReplicaSet("web", got); err != nil {
		t.Fatalf("UpdateReplicaSet: %v", err)
	}

	updated, _, _ := c.GetReplicaSet("web")
	if updated.DesiredCount != 5 {
		t.Errorf("got desired %d, want 5", updated.DesiredCount)
	}

	if err := c.DeleteReplicaSet("web"); err != nil {
		t.Fatalf("DeleteReplicaSet: %v", err)
	}

	_, found, _ = c.GetReplicaSet("web")
	if found {
		t.Error("expected RS to be deleted")
	}
}

func TestNodeRoundTrip(t *testing.T) {
	c, _, _, nodeStore, ts := setup()
	defer ts.Close()

	node := types.Node{Name: "node-1", Status: types.NodeStateReady}
	if err := c.CreateNode(node); err != nil {
		t.Fatalf("CreateNode: %v", err)
	}

	if len(nodeStore.List()) != 1 {
		t.Fatalf("expected 1 node in store")
	}

	got, found, err := c.GetNode("node-1")
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	if !found {
		t.Fatal("expected node to be found")
	}
	if got.Status != types.NodeStateReady {
		t.Errorf("got status %q, want %q", got.Status, types.NodeStateReady)
	}

	nodes, err := c.ListNodes()
	if err != nil {
		t.Fatalf("ListNodes: %v", err)
	}
	if len(nodes) != 1 {
		t.Errorf("got %d nodes, want 1", len(nodes))
	}

	got.Status = types.NodeStateNotReady
	if err := c.UpdateNode("node-1", got); err != nil {
		t.Fatalf("UpdateNode: %v", err)
	}

	updated, _, _ := c.GetNode("node-1")
	if updated.Status != types.NodeStateNotReady {
		t.Errorf("got status %q, want %q", updated.Status, types.NodeStateNotReady)
	}

	if err := c.DeleteNode("node-1"); err != nil {
		t.Fatalf("DeleteNode: %v", err)
	}

	_, found, _ = c.GetNode("node-1")
	if found {
		t.Error("expected node to be deleted")
	}
}

func TestGetNotFound(t *testing.T) {
	c, _, _, _, ts := setup()
	defer ts.Close()

	_, found, err := c.GetPod("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Error("expected not found")
	}

	_, found, err = c.GetReplicaSet("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Error("expected not found")
	}

	_, found, err = c.GetNode("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Error("expected not found")
	}
}
