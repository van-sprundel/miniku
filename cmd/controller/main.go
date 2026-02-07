package main

import (
	"flag"
	"log"

	"miniku/pkg/client"
	"miniku/pkg/controller"
)

func main() {
	apiServer := flag.String("api-server", "http://localhost:8080", "API server URL")
	flag.Parse()

	c := client.New(*apiServer)

	log.Printf("controller: connecting to API server at %s", *apiServer)

	nodeCtrl := controller.NewNodeController(c)
	go nodeCtrl.Run()

	rsCtrl := controller.New(c)
	rsCtrl.Run()
}
