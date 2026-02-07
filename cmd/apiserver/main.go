package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	bolt "go.etcd.io/bbolt"

	"miniku/pkg/api"
	"miniku/pkg/store"
	"miniku/pkg/types"
)

func main() {
	port := flag.Int("port", 8080, "port to listen on")
	dbPath := flag.String("db", "miniku.db", "path to BoltDB file")
	flag.Parse()

	db, err := bolt.Open(*dbPath, 0600, nil)
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

	srv := &api.Server{
		PodStore:  podStore,
		RSStore:   rsStore,
		NodeStore: nodeStore,
	}

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("apiserver: listening on %s", addr)
	if err := http.ListenAndServe(addr, srv.Routes()); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
