package main

import (
	"miniku/pkg/api"
	"miniku/pkg/controller"
	"miniku/pkg/kubelet"
	"miniku/pkg/runtime"
	"miniku/pkg/scheduler"
	"miniku/pkg/store"
	"miniku/pkg/types"
	"net/http"
)

func main() {
	podStore := store.NewMemStore[types.Pod]()
	rsStore := store.NewMemStore[types.ReplicaSet]()
	nodeStore := store.NewMemStore[types.Node]()

	nodeStore.Put("node-1", types.Node{Name: "node-1", Status: types.NodeStateReady})
	nodeStore.Put("node-2", types.Node{Name: "node-2", Status: types.NodeStateReady})

	// assign pods to nodes
	sched := scheduler.New(podStore, nodeStore)
	go sched.Run()

	// reconcile pods -> containers (one per node)
	rt := &runtime.DockerCLIRuntime{}
	kubelet1 := kubelet.New(podStore, rt, "node-1")
	kubelet2 := kubelet.New(podStore, rt, "node-2")
	go kubelet1.Run()
	go kubelet2.Run()

	// reconcile replicasets -> pods
	rsController := controller.New(podStore, rsStore)
	go rsController.Run()

	// API Server
	srv := &api.Server{
		PodStore: podStore,
		RSStore:  rsStore,
	}
	http.ListenAndServe(":8080", srv.Routes())
}
