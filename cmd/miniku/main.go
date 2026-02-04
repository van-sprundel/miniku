package main

import (
	"miniku/pkg/api"
	"miniku/pkg/kubelet"
	"miniku/pkg/runtime"
	"miniku/pkg/store"
	"net/http"
)

func main() {
	store := store.NewMemStore()
	runtime := &runtime.DockerCLIRuntime{}
	kubelet := kubelet.New(store, runtime)
	go kubelet.Run()

	srv := &api.Server{Store: store}
	http.ListenAndServe(":8080", srv.Routes())
}
