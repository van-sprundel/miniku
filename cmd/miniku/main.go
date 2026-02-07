package main

import (
	"log"
	"net"
	"net/http"

	bolt "go.etcd.io/bbolt"

	"miniku/pkg/api"
	"miniku/pkg/client"
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

	// start API server on :8080 in a goroutine
	srv := &api.Server{
		PodStore:  podStore,
		RSStore:   rsStore,
		NodeStore: nodeStore,
	}

	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	go func() {
		if err := http.Serve(ln, srv.Routes()); err != nil {
			log.Fatalf("server failed: %v", err)
		}
	}()
	log.Println("apiserver: listening on :8080")

	// create client pointing at localhost:8080
	c := client.New("http://localhost:8080")

	// register nodes via client
	if err := c.CreateNode(types.Node{Name: "node-1", Status: types.NodeStateReady}); err != nil {
		log.Fatalf("failed to register node-1: %v", err)
	}
	if err := c.CreateNode(types.Node{Name: "node-2", Status: types.NodeStateReady}); err != nil {
		log.Fatalf("failed to register node-2: %v", err)
	}

	// assign pods to nodes
	sched := scheduler.New(c)
	go sched.Run()

	// reconcile pods -> containers (one per node)
	rt := &runtime.DockerCLIRuntime{}
	kubelet1 := kubelet.New(c, rt, "node-1")
	kubelet2 := kubelet.New(c, rt, "node-2")
	go kubelet1.Run()
	go kubelet2.Run()

	// reconcile replicasets -> pods
	rsController := controller.New(c)
	go rsController.Run()

	// mark nodes NotReady if heartbeat is stale
	nodeController := controller.NewNodeController(c)
	nodeController.Run()
}
