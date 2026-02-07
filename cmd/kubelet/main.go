package main

import (
	"flag"
	"log"

	"miniku/pkg/client"
	"miniku/pkg/kubelet"
	"miniku/pkg/runtime"
	"miniku/pkg/types"
)

func main() {
	apiServer := flag.String("api-server", "http://localhost:8080", "API server URL")
	name := flag.String("name", "", "node name (required)")
	rootDir := flag.String("root-dir", "/var/lib/miniku", "runtime root directory")
	flag.Parse()

	if *name == "" {
		log.Fatal("--name is required")
	}

	c := client.New(*apiServer)

	// register node
	if err := c.CreateNode(types.Node{
		Name:   *name,
		Status: types.NodeStateReady,
	}); err != nil {
		log.Fatalf("failed to register node: %v", err)
	}

	log.Printf("kubelet %s: registered with API server at %s", *name, *apiServer)

	rt, err := runtime.NewNamespaceRuntime(*rootDir)
	if err != nil {
		log.Fatalf("failed to create runtime: %v", err)
	}
	k := kubelet.New(c, rt, *name)
	k.Run()
}
