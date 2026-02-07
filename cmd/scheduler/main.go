package main

import (
	"flag"
	"log"

	"miniku/pkg/client"
	"miniku/pkg/scheduler"
)

func main() {
	apiServer := flag.String("api-server", "http://localhost:8080", "API server URL")
	flag.Parse()

	c := client.New(*apiServer)

	log.Printf("scheduler: connecting to API server at %s", *apiServer)
	sched := scheduler.New(c)
	sched.Run()
}
