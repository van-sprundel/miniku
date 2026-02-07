/// Main API architecture:
// Pods:
//   POST /pods
//   GET /pods
//   GET /pods/{name}
//   PUT /pods/{name}
//   DELETE /pods/{name}
//
// ReplicaSets:
//   POST /replicasets
//   GET /replicasets
//   GET /replicasets/{name}
//   PUT /replicasets/{name}
//   DELETE /replicasets/{name}
//
// Nodes:
//   POST /nodes
//   GET /nodes
//   GET /nodes/{name}
//   PUT /nodes/{name}
//   DELETE /nodes/{name}

package api

import (
	"encoding/json"
	"log"
	"miniku/pkg/store"
	"miniku/pkg/types"
	"net/http"
)

type Server struct {
	PodStore  store.PodStore
	RSStore   store.ReplicaSetStore
	NodeStore store.NodeStore
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /pods", s.handleListPods)
	mux.HandleFunc("POST /pods", s.handleCreatePod)
	mux.HandleFunc("GET /pods/{name}", s.handleGetPod)
	mux.HandleFunc("PUT /pods/{name}", s.handleUpdatePod)
	mux.HandleFunc("DELETE /pods/{name}", s.handleDeletePod)

	mux.HandleFunc("GET /replicasets", s.handleListReplicaSets)
	mux.HandleFunc("POST /replicasets", s.handleCreateReplicaSet)
	mux.HandleFunc("GET /replicasets/{name}", s.handleGetReplicaSet)
	mux.HandleFunc("PUT /replicasets/{name}", s.handleUpdateReplicaSet)
	mux.HandleFunc("DELETE /replicasets/{name}", s.handleDeleteReplicaSet)

	mux.HandleFunc("GET /nodes", s.handleListNodes)
	mux.HandleFunc("POST /nodes", s.handleCreateNode)
	mux.HandleFunc("GET /nodes/{name}", s.handleGetNode)
	mux.HandleFunc("PUT /nodes/{name}", s.handleUpdateNode)
	mux.HandleFunc("DELETE /nodes/{name}", s.handleDeleteNode)

	return mux
}

func (s *Server) handleListPods(w http.ResponseWriter, r *http.Request) {
	pods := s.PodStore.List()
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, pods)
}

func (s *Server) handleCreatePod(w http.ResponseWriter, r *http.Request) {
	var pod types.Pod
	if err := json.NewDecoder(r.Body).Decode(&pod); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if pod.Status == "" {
		pod.Status = types.PodStatusPending
	}

	s.PodStore.Put(pod.Spec.Name, pod)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, pod)
}

func (s *Server) handleGetPod(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	pod, ok := s.PodStore.Get(name)
	if !ok {
		http.Error(w, "pod not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, pod)
}

func (s *Server) handleUpdatePod(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	var pod types.Pod
	if err := json.NewDecoder(r.Body).Decode(&pod); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	s.PodStore.Put(name, pod)

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, pod)
}

func (s *Server) handleDeletePod(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	s.PodStore.Delete(name)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListReplicaSets(w http.ResponseWriter, r *http.Request) {
	replicaSets := s.RSStore.List()
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, replicaSets)
}

func (s *Server) handleCreateReplicaSet(w http.ResponseWriter, r *http.Request) {
	var rs types.ReplicaSet
	if err := json.NewDecoder(r.Body).Decode(&rs); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	s.RSStore.Put(rs.Name, rs)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, rs)
}

func (s *Server) handleGetReplicaSet(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	rs, ok := s.RSStore.Get(name)
	if !ok {
		http.Error(w, "replicaset not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, rs)
}

func (s *Server) handleUpdateReplicaSet(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	var rs types.ReplicaSet
	if err := json.NewDecoder(r.Body).Decode(&rs); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	s.RSStore.Put(name, rs)

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, rs)
}

func (s *Server) handleDeleteReplicaSet(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	s.RSStore.Delete(name)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListNodes(w http.ResponseWriter, r *http.Request) {
	nodes := s.NodeStore.List()
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, nodes)
}

func (s *Server) handleCreateNode(w http.ResponseWriter, r *http.Request) {
	var node types.Node
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	s.NodeStore.Put(node.Name, node)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, node)
}

func (s *Server) handleGetNode(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	node, ok := s.NodeStore.Get(name)
	if !ok {
		http.Error(w, "node not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, node)
}

func (s *Server) handleUpdateNode(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	var node types.Node
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	s.NodeStore.Put(name, node)

	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, node)
}

func (s *Server) handleDeleteNode(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	s.NodeStore.Delete(name)
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, v any) {
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("failed to encode response: %v", err)
	}
}
