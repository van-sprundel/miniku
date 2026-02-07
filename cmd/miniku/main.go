package main

import (
	"log"
	"net/http"

	bolt "go.etcd.io/bbolt"

	"miniku/pkg/api"
	"miniku/pkg/controller"
	"miniku/pkg/kubelet"
	"miniku/pkg/runtime"
	"miniku/pkg/scheduler"
	"miniku/pkg/store"
	"miniku/pkg/types"
)

func main() {
	db, err := bolt.Open("miniku.db", 0600, nil)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("failed to close database: %v", err)
		}
	}()

	podStore := store.NewBoltStore[types.Pod](db, "pods")
	rsStore := store.NewBoltStore[types.ReplicaSet](db, "replicasets")
	nodeStore := store.NewBoltStore[types.Node](db, "nodes")

	nodeStore.Put("node-1", types.Node{Name: "node-1", Status: types.NodeStateReady})
	nodeStore.Put("node-2", types.Node{Name: "node-2", Status: types.NodeStateReady})

	// assign pods to nodes
	sched := scheduler.New(podStore, nodeStore)
	go sched.Run()

	// reconcile pods -> containers (one per node)
	rt := &runtime.DockerCLIRuntime{}
	kubelet1 := kubelet.New(podStore, nodeStore, rt, "node-1")
	kubelet2 := kubelet.New(podStore, nodeStore, rt, "node-2")
	go kubelet1.Run()
	go kubelet2.Run()

	// reconcile replicasets -> pods
	rsController := controller.New(podStore, rsStore)
	go rsController.Run()

	// mark nodes NotReady if heartbeat is stale
	nodeController := controller.NewNodeController(nodeStore)
	go nodeController.Run()

	// API Server
	srv := &api.Server{
		PodStore: podStore,
		RSStore:  rsStore,
	}
	if err := http.ListenAndServe(":8080", srv.Routes()); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
