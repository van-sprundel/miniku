/// Main API architecture:
// POST /pods
// GET /pods
// GET /pods/{name}
// DEL /pods/{name}

package api

import (
	"encoding/json"
	"miniku/pkg/store"
	"miniku/pkg/types"
	"net/http"
)

type Server struct {
	Store store.Store
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /pods", s.handleListPods)
	mux.HandleFunc("POST /pods", s.handleCreatePod)
	mux.HandleFunc("GET /pods/{name}", s.handleGetPod)
	mux.HandleFunc("DELETE /pods/{name}", s.handleDeletePod)
	return mux
}

func (s *Server) handleListPods(w http.ResponseWriter, r *http.Request) {
	pods := s.Store.List()
	encoder := json.NewEncoder(w)
	encoder.Encode(pods)
}
func (s *Server) handleCreatePod(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	w.Header().Set("Content-Type", "application/json")

	var podSpec types.PodSpec
	err := decoder.Decode(&podSpec)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	pod := types.NewPod(podSpec)
	s.Store.Put(pod)

	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleGetPod(w http.ResponseWriter, r *http.Request) {
	nameString := r.PathValue("name")
	w.Header().Set("Content-Type", "application/json")

	pod, ok := s.Store.Get(nameString)
	if !ok {
		http.Error(w, "pod not found", http.StatusNotFound)
		return
	}

	encoder := json.NewEncoder(w)
	encoder.Encode(pod)
}
func (s *Server) handleDeletePod(w http.ResponseWriter, r *http.Request) {
	nameString := r.PathValue("name")
	s.Store.Delete(nameString)

	w.WriteHeader(http.StatusNoContent)
}
