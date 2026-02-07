package controller

import (
	"miniku/pkg/store"
	"miniku/pkg/types"
	"time"
)

const NODE_HEARTBEAT_THRESHOLD = 15 * time.Second

type NodeController struct {
	nodeStore    store.NodeStore
	PollInterval time.Duration
}

func NewNodeController(nodeStore store.NodeStore) *NodeController {
	return &NodeController{
		nodeStore:    nodeStore,
		PollInterval: 5 * time.Second,
	}
}

func (c *NodeController) Run() {
	for {
		for _, node := range c.nodeStore.List() {
			c.reconcile(node)
		}
		time.Sleep(c.PollInterval)
	}
}

func (c *NodeController) reconcile(node types.Node) {
	if time.Since(node.LastHeartbeat) > NODE_HEARTBEAT_THRESHOLD {
		node.Status = types.NodeStateNotReady
	} else {
		node.Status = types.NodeStateReady
	}
	c.nodeStore.Put(node.Name, node)
}
