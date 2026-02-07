package controller

import (
	"log"
	"miniku/pkg/client"
	"miniku/pkg/types"
	"time"
)

const NODE_HEARTBEAT_THRESHOLD = 15 * time.Second

type NodeController struct {
	client       *client.Client
	PollInterval time.Duration
}

func NewNodeController(client *client.Client) *NodeController {
	return &NodeController{
		client:       client,
		PollInterval: 5 * time.Second,
	}
}

func (c *NodeController) Run() {
	for {
		nodes, err := c.client.ListNodes()
		if err != nil {
			log.Printf("node controller: failed to list nodes: %v", err)
			time.Sleep(c.PollInterval)
			continue
		}

		for _, node := range nodes {
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
	if err := c.client.UpdateNode(node.Name, node); err != nil {
		log.Printf("node controller: failed to update node %s: %v", node.Name, err)
	}
}
