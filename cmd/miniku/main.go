package main

import (
	"miniku/pkg/api"
	"miniku/pkg/kubelet"
	"miniku/pkg/runtime"
	"miniku/pkg/store"
	"miniku/pkg/types"
	"net/http"
)

func main() {
	podStore := store.NewMemStore[types.Pod]()
	runtime := &runtime.DockerCLIRuntime{}
	kubelet := kubelet.New(podStore, runtime)
	go kubelet.Run()

	srv := &api.Server{Store: podStore}
	http.ListenAndServe(":8080", srv.Routes())
}
