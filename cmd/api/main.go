package main

import (
	"miniku/pkg/api"
	"miniku/pkg/store"
	"net/http"
)

func main() {
	store := store.NewMemStore()
	srv := &api.Server{Store: store}
	http.ListenAndServe(":8080", srv.Routes())
}
