package main

import (
	"miniku/pkg/api"
	"miniku/pkg/controller"
	"miniku/pkg/kubelet"
	"miniku/pkg/runtime"
	"miniku/pkg/store"
	"miniku/pkg/types"
	"net/http"
)

func main() {
	podStore := store.NewMemStore[types.Pod]()
	rsStore := store.NewMemStore[types.ReplicaSet]()

	// kubelet: reconciles pods -> containers
	rt := &runtime.DockerCLIRuntime{}
	kubelet := kubelet.New(podStore, rt)
	go kubelet.Run()

	// controller: reconciles replicasets -> pods
	rsController := controller.New(podStore, rsStore)
	go rsController.Run()

	// API Server
	srv := &api.Server{
		PodStore: podStore,
		RSStore:  rsStore,
	}
	http.ListenAndServe(":8080", srv.Routes())
}
