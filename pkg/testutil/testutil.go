package testutil

import (
	"miniku/pkg/api"
	"miniku/pkg/client"
	"miniku/pkg/store"
	"miniku/pkg/types"
	"net/http/httptest"
)

type TestEnv struct {
	Server    *httptest.Server
	Client    *client.Client
	PodStore  store.PodStore
	RSStore   store.ReplicaSetStore
	NodeStore store.NodeStore
}

func NewTestEnv() *TestEnv {
	podStore := store.NewMemStore[types.Pod]()
	rsStore := store.NewMemStore[types.ReplicaSet]()
	nodeStore := store.NewMemStore[types.Node]()

	srv := &api.Server{PodStore: podStore, RSStore: rsStore, NodeStore: nodeStore}
	ts := httptest.NewServer(srv.Routes())

	c := client.New(ts.URL)

	return &TestEnv{
		Server:    ts,
		Client:    c,
		PodStore:  podStore,
		RSStore:   rsStore,
		NodeStore: nodeStore,
	}
}

func (e *TestEnv) Close() {
	e.Server.Close()
}
