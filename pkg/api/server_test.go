package api

import (
	"encoding/json"
	"miniku/pkg/store"
	"miniku/pkg/types"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer() (*Server, store.PodStore, store.ReplicaSetStore, store.NodeStore) {
	podStore := store.NewMemStore[types.Pod]()
	rsStore := store.NewMemStore[types.ReplicaSet]()
	nodeStore := store.NewMemStore[types.Node]()
	srv := &Server{PodStore: podStore, RSStore: rsStore, NodeStore: nodeStore}
	return srv, podStore, rsStore, nodeStore
}

func TestRoutes(t *testing.T) {
	srv, podStore, _, _ := newTestServer()
	podStore.Put("test", types.Pod{Spec: types.PodSpec{Name: "test", Image: "nginx"}})

	ts := httptest.NewServer(srv.Routes())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/pods")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /pods: got status %d, want 200", resp.StatusCode)
	}

	var pods []types.Pod
	if err := json.NewDecoder(resp.Body).Decode(&pods); err != nil {
		t.Fatal(err)
	}
	if len(pods) != 1 {
		t.Errorf("got %d pods, want 1", len(pods))
	}
}

func TestGetPod(t *testing.T) {
	tests := []struct {
		name       string
		setupPods  []types.Pod
		getPodName string
		wantStatus int
	}{
		{
			name:       "existing pod returns 200",
			setupPods:  []types.Pod{{Spec: types.PodSpec{Name: "foo", Image: "nginx"}}},
			getPodName: "foo",
			wantStatus: http.StatusOK,
		},
		{
			name:       "nonexistent pod returns 404",
			setupPods:  []types.Pod{},
			getPodName: "missing",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, podStore, _, _ := newTestServer()
			for _, p := range tt.setupPods {
				podStore.Put(p.Spec.Name, p)
			}

			req := httptest.NewRequest("GET", "/pods/"+tt.getPodName, nil)
			req.SetPathValue("name", tt.getPodName)
			rec := httptest.NewRecorder()

			srv.handleGetPod(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestListPods(t *testing.T) {
	tests := []struct {
		name      string
		setupPods []types.Pod
		wantCount int
	}{
		{
			name:      "empty store returns empty array",
			setupPods: []types.Pod{},
			wantCount: 0,
		},
		{
			name: "returns all pods",
			setupPods: []types.Pod{
				{Spec: types.PodSpec{Name: "pod1", Image: "nginx"}},
				{Spec: types.PodSpec{Name: "pod2", Image: "redis"}},
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, podStore, _, _ := newTestServer()
			for _, p := range tt.setupPods {
				podStore.Put(p.Spec.Name, p)
			}

			req := httptest.NewRequest("GET", "/pods", nil)
			rec := httptest.NewRecorder()

			srv.handleListPods(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("got status %d, want 200", rec.Code)
			}

			var pods []types.Pod
			if err := json.NewDecoder(rec.Body).Decode(&pods); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if len(pods) != tt.wantCount {
				t.Errorf("got %d pods, want %d", len(pods), tt.wantCount)
			}
		})
	}
}

func TestCreatePod(t *testing.T) {
	srv, podStore, _, _ := newTestServer()

	body := `{"spec":{"name":"test","image":"nginx"}}`
	req := httptest.NewRequest("POST", "/pods", strings.NewReader(body))
	rec := httptest.NewRecorder()

	srv.handleCreatePod(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("got status %d, want 201", rec.Code)
	}

	pods := podStore.List()
	if len(pods) != 1 {
		t.Fatalf("expected 1 pod in store, got %d", len(pods))
	}
	if pods[0].Spec.Name != "test" {
		t.Errorf("got pod name %q, want %q", pods[0].Spec.Name, "test")
	}
	if pods[0].Status != types.PodStatusPending {
		t.Errorf("got status %q, want %q", pods[0].Status, types.PodStatusPending)
	}
}

func TestCreatePodWithStatus(t *testing.T) {
	srv, podStore, _, _ := newTestServer()

	body := `{"spec":{"name":"test","image":"nginx"},"status":"Running"}`
	req := httptest.NewRequest("POST", "/pods", strings.NewReader(body))
	rec := httptest.NewRecorder()

	srv.handleCreatePod(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("got status %d, want 201", rec.Code)
	}

	pod, ok := podStore.Get("test")
	if !ok {
		t.Fatal("pod not found in store")
	}
	if pod.Status != types.PodStatusRunning {
		t.Errorf("got status %q, want %q", pod.Status, types.PodStatusRunning)
	}
}

func TestUpdatePod(t *testing.T) {
	srv, podStore, _, _ := newTestServer()
	podStore.Put("test", types.Pod{
		Spec:   types.PodSpec{Name: "test", Image: "nginx"},
		Status: types.PodStatusPending,
	})

	body := `{"spec":{"name":"test","image":"nginx","node_name":"node-1"},"status":"Running"}`
	req := httptest.NewRequest("PUT", "/pods/test", strings.NewReader(body))
	req.SetPathValue("name", "test")
	rec := httptest.NewRecorder()

	srv.handleUpdatePod(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want 200", rec.Code)
	}

	pod, ok := podStore.Get("test")
	if !ok {
		t.Fatal("pod not found in store")
	}
	if pod.Spec.NodeName != "node-1" {
		t.Errorf("got node %q, want %q", pod.Spec.NodeName, "node-1")
	}
	if pod.Status != types.PodStatusRunning {
		t.Errorf("got status %q, want %q", pod.Status, types.PodStatusRunning)
	}
}

func TestDeleteThenGet(t *testing.T) {
	srv, podStore, _, _ := newTestServer()
	podStore.Put("victim", types.Pod{Spec: types.PodSpec{Name: "victim", Image: "nginx"}})

	// delete the pod
	delReq := httptest.NewRequest("DELETE", "/pods/victim", nil)
	delReq.SetPathValue("name", "victim")
	delRec := httptest.NewRecorder()
	srv.handleDeletePod(delRec, delReq)

	if delRec.Code != http.StatusNoContent {
		t.Errorf("delete: got status %d, want 204", delRec.Code)
	}

	// try to get it
	getReq := httptest.NewRequest("GET", "/pods/victim", nil)
	getReq.SetPathValue("name", "victim")
	getRec := httptest.NewRecorder()
	srv.handleGetPod(getRec, getReq)

	if getRec.Code != http.StatusNotFound {
		t.Errorf("get after delete: got status %d, want 404", getRec.Code)
	}
}

func TestGetReplicaSet(t *testing.T) {
	tests := []struct {
		name       string
		setupRS    []types.ReplicaSet
		getRSName  string
		wantStatus int
	}{
		{
			name:       "existing replicaset returns 200",
			setupRS:    []types.ReplicaSet{{Name: "nginx-rs", DesiredCount: 3}},
			getRSName:  "nginx-rs",
			wantStatus: http.StatusOK,
		},
		{
			name:       "nonexistent replicaset returns 404",
			setupRS:    []types.ReplicaSet{},
			getRSName:  "missing",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, _, rsStore, _ := newTestServer()
			for _, rs := range tt.setupRS {
				rsStore.Put(rs.Name, rs)
			}

			req := httptest.NewRequest("GET", "/replicasets/"+tt.getRSName, nil)
			req.SetPathValue("name", tt.getRSName)
			rec := httptest.NewRecorder()

			srv.handleGetReplicaSet(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestCreateReplicaSet(t *testing.T) {
	srv, _, rsStore, _ := newTestServer()

	body := `{"name":"nginx-rs","desiredCount":3,"selector":{"app":"nginx"},"template":{"image":"nginx:latest"}}`
	req := httptest.NewRequest("POST", "/replicasets", strings.NewReader(body))
	rec := httptest.NewRecorder()

	srv.handleCreateReplicaSet(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("got status %d, want 201", rec.Code)
	}

	rsList := rsStore.List()
	if len(rsList) != 1 {
		t.Fatalf("expected 1 replicaset in store, got %d", len(rsList))
	}
	if rsList[0].Name != "nginx-rs" {
		t.Errorf("got rs name %q, want %q", rsList[0].Name, "nginx-rs")
	}
	if rsList[0].DesiredCount != 3 {
		t.Errorf("got desired count %d, want %d", rsList[0].DesiredCount, 3)
	}
}

func TestUpdateReplicaSet(t *testing.T) {
	srv, _, rsStore, _ := newTestServer()
	rsStore.Put("nginx-rs", types.ReplicaSet{Name: "nginx-rs", DesiredCount: 3})

	body := `{"name":"nginx-rs","desiredCount":5}`
	req := httptest.NewRequest("PUT", "/replicasets/nginx-rs", strings.NewReader(body))
	req.SetPathValue("name", "nginx-rs")
	rec := httptest.NewRecorder()

	srv.handleUpdateReplicaSet(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want 200", rec.Code)
	}

	rs, ok := rsStore.Get("nginx-rs")
	if !ok {
		t.Fatal("replicaset not found")
	}
	if rs.DesiredCount != 5 {
		t.Errorf("got desired count %d, want 5", rs.DesiredCount)
	}
}

func TestListReplicaSets(t *testing.T) {
	srv, _, rsStore, _ := newTestServer()
	rsStore.Put("rs1", types.ReplicaSet{Name: "rs1", DesiredCount: 2})
	rsStore.Put("rs2", types.ReplicaSet{Name: "rs2", DesiredCount: 5})

	req := httptest.NewRequest("GET", "/replicasets", nil)
	rec := httptest.NewRecorder()

	srv.handleListReplicaSets(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want 200", rec.Code)
	}

	var rsList []types.ReplicaSet
	if err := json.NewDecoder(rec.Body).Decode(&rsList); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(rsList) != 2 {
		t.Errorf("got %d replicasets, want 2", len(rsList))
	}
}

func TestGetNode(t *testing.T) {
	tests := []struct {
		name       string
		setupNodes []types.Node
		getNode    string
		wantStatus int
	}{
		{
			name:       "existing node returns 200",
			setupNodes: []types.Node{{Name: "node-1", Status: types.NodeStateReady}},
			getNode:    "node-1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "nonexistent node returns 404",
			setupNodes: []types.Node{},
			getNode:    "missing",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, _, _, nodeStore := newTestServer()
			for _, n := range tt.setupNodes {
				nodeStore.Put(n.Name, n)
			}

			req := httptest.NewRequest("GET", "/nodes/"+tt.getNode, nil)
			req.SetPathValue("name", tt.getNode)
			rec := httptest.NewRecorder()

			srv.handleGetNode(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestListNodes(t *testing.T) {
	srv, _, _, nodeStore := newTestServer()
	nodeStore.Put("node-1", types.Node{Name: "node-1", Status: types.NodeStateReady})
	nodeStore.Put("node-2", types.Node{Name: "node-2", Status: types.NodeStateReady})

	req := httptest.NewRequest("GET", "/nodes", nil)
	rec := httptest.NewRecorder()

	srv.handleListNodes(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want 200", rec.Code)
	}

	var nodes []types.Node
	if err := json.NewDecoder(rec.Body).Decode(&nodes); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(nodes) != 2 {
		t.Errorf("got %d nodes, want 2", len(nodes))
	}
}

func TestCreateNode(t *testing.T) {
	srv, _, _, nodeStore := newTestServer()

	body := `{"name":"node-1","status":"Ready"}`
	req := httptest.NewRequest("POST", "/nodes", strings.NewReader(body))
	rec := httptest.NewRecorder()

	srv.handleCreateNode(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("got status %d, want 201", rec.Code)
	}

	node, ok := nodeStore.Get("node-1")
	if !ok {
		t.Fatal("node not found in store")
	}
	if node.Name != "node-1" {
		t.Errorf("got name %q, want %q", node.Name, "node-1")
	}
}

func TestUpdateNode(t *testing.T) {
	srv, _, _, nodeStore := newTestServer()
	nodeStore.Put("node-1", types.Node{Name: "node-1", Status: types.NodeStateReady})

	body := `{"name":"node-1","status":"NotReady"}`
	req := httptest.NewRequest("PUT", "/nodes/node-1", strings.NewReader(body))
	req.SetPathValue("name", "node-1")
	rec := httptest.NewRecorder()

	srv.handleUpdateNode(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want 200", rec.Code)
	}

	node, ok := nodeStore.Get("node-1")
	if !ok {
		t.Fatal("node not found")
	}
	if node.Status != types.NodeStateNotReady {
		t.Errorf("got status %q, want %q", node.Status, types.NodeStateNotReady)
	}
}

func TestDeleteReplicaSet(t *testing.T) {
	srv, _, rsStore, _ := newTestServer()
	rsStore.Put("nginx-rs", types.ReplicaSet{Name: "nginx-rs", DesiredCount: 3})

	req := httptest.NewRequest("DELETE", "/replicasets/nginx-rs", nil)
	req.SetPathValue("name", "nginx-rs")
	rec := httptest.NewRecorder()

	srv.handleDeleteReplicaSet(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("got status %d, want 204", rec.Code)
	}

	_, ok := rsStore.Get("nginx-rs")
	if ok {
		t.Error("replicaset should have been deleted")
	}
}

func TestDeleteNode(t *testing.T) {
	srv, _, _, nodeStore := newTestServer()
	nodeStore.Put("node-1", types.Node{Name: "node-1", Status: types.NodeStateReady})

	req := httptest.NewRequest("DELETE", "/nodes/node-1", nil)
	req.SetPathValue("name", "node-1")
	rec := httptest.NewRecorder()

	srv.handleDeleteNode(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("got status %d, want 204", rec.Code)
	}

	_, ok := nodeStore.Get("node-1")
	if ok {
		t.Error("node should have been deleted")
	}
}
