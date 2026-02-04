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
			st := store.NewMemStore[types.Pod]()
			for _, p := range tt.setupPods {
				st.Put(p.Spec.Name, p)
			}
			srv := &Server{Store: st}

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
			st := store.NewMemStore[types.Pod]()
			for _, p := range tt.setupPods {
				st.Put(p.Spec.Name, p)
			}
			srv := &Server{Store: st}

			req := httptest.NewRequest("GET", "/pods", nil)
			rec := httptest.NewRecorder()

			srv.handleListPods(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("got status %d, want 200", rec.Code)
			}

			var pods []types.Pod
			json.NewDecoder(rec.Body).Decode(&pods)
			if len(pods) != tt.wantCount {
				t.Errorf("got %d pods, want %d", len(pods), tt.wantCount)
			}
		})
	}
}

func TestCreatePod(t *testing.T) {
	st := store.NewMemStore[types.Pod]()
	srv := &Server{Store: st}

	body := `{"name":"test","image":"nginx"}`
	req := httptest.NewRequest("POST", "/pods", strings.NewReader(body))
	rec := httptest.NewRecorder()

	srv.handleCreatePod(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("got status %d, want 201", rec.Code)
	}

	pods := st.List()
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

func TestDeleteThenGet(t *testing.T) {
	st := store.NewMemStore[types.Pod]()
	st.Put("victim", types.Pod{Spec: types.PodSpec{Name: "victim", Image: "nginx"}})
	srv := &Server{Store: st}

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
