/// Main API architecture:
// Pods:
//   POST /pods
//   GET /pods
//   GET /pods/{name}
//   DELETE /pods/{name}
//
// ReplicaSets:
//   POST /replicasets
//   GET /replicasets
//   GET /replicasets/{name}
//   DELETE /replicasets/{name}

package api

import (
	"encoding/json"
	"miniku/pkg/store"
	"miniku/pkg/types"
	"net/http"
)

type Server struct {
	PodStore store.PodStore
	RSStore  store.ReplicaSetStore
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /pods", s.handleListPods)
	mux.HandleFunc("POST /pods", s.handleCreatePod)
	mux.HandleFunc("GET /pods/{name}", s.handleGetPod)
	mux.HandleFunc("DELETE /pods/{name}", s.handleDeletePod)

	mux.HandleFunc("GET /replicasets", s.handleListReplicaSets)
	mux.HandleFunc("POST /replicasets", s.handleCreateReplicaSet)
	mux.HandleFunc("GET /replicasets/{name}", s.handleGetReplicaSet)
	mux.HandleFunc("DELETE /replicasets/{name}", s.handleDeleteReplicaSet)

	return mux
}

func (s *Server) handleListPods(w http.ResponseWriter, r *http.Request) {
	pods := s.PodStore.List()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pods)
}

func (s *Server) handleCreatePod(w http.ResponseWriter, r *http.Request) {
	var podSpec types.PodSpec
	if err := json.NewDecoder(r.Body).Decode(&podSpec); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	pod := types.NewPod(podSpec)
	s.PodStore.Put(pod.Spec.Name, pod)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(pod)
}

func (s *Server) handleGetPod(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	pod, ok := s.PodStore.Get(name)
	if !ok {
		http.Error(w, "pod not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pod)
}

func (s *Server) handleDeletePod(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	s.PodStore.Delete(name)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListReplicaSets(w http.ResponseWriter, r *http.Request) {
	replicaSets := s.RSStore.List()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(replicaSets)
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
	json.NewEncoder(w).Encode(rs)
}

func (s *Server) handleGetReplicaSet(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	rs, ok := s.RSStore.Get(name)
	if !ok {
		http.Error(w, "replicaset not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rs)
}

func (s *Server) handleDeleteReplicaSet(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	s.RSStore.Delete(name)
	w.WriteHeader(http.StatusNoContent)
}
