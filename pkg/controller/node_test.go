package controller

import (
	"miniku/pkg/store"
	"miniku/pkg/types"
	"testing"
	"time"
)

func TestNodeControllerReconcile(t *testing.T) {
	tests := []struct {
		name           string
		node           types.Node
		expectedStatus types.NodeState
	}{
		{
			name: "fresh heartbeat stays ready",
			node: types.Node{
				Name:          "node-1",
				Status:        types.NodeStateReady,
				LastHeartbeat: time.Now(),
			},
			expectedStatus: types.NodeStateReady,
		},
		{
			name: "stale heartbeat marked notready",
			node: types.Node{
				Name:          "node-1",
				Status:        types.NodeStateReady,
				LastHeartbeat: time.Now().Add(-30 * time.Second),
			},
			expectedStatus: types.NodeStateNotReady,
		},
		{
			name: "zero heartbeat marked notready",
			node: types.Node{
				Name:   "node-1",
				Status: types.NodeStateReady,
			},
			expectedStatus: types.NodeStateNotReady,
		},
		{
			name: "recovered node marked ready",
			node: types.Node{
				Name:          "node-1",
				Status:        types.NodeStateNotReady,
				LastHeartbeat: time.Now(),
			},
			expectedStatus: types.NodeStateReady,
		},
		{
			name: "heartbeat exactly at threshold",
			node: types.Node{
				Name:          "node-1",
				Status:        types.NodeStateReady,
				LastHeartbeat: time.Now().Add(-NODE_HEARTBEAT_THRESHOLD),
			},
			expectedStatus: types.NodeStateNotReady,
		},
		{
			name: "heartbeat just before threshold",
			node: types.Node{
				Name:          "node-1",
				Status:        types.NodeStateReady,
				LastHeartbeat: time.Now().Add(-NODE_HEARTBEAT_THRESHOLD + time.Second),
			},
			expectedStatus: types.NodeStateReady,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodeStore := store.NewMemStore[types.Node]()
			nodeStore.Put(tt.node.Name, tt.node)

			controller := NewNodeController(nodeStore)
			controller.reconcile(tt.node)

			updatedNode, found := nodeStore.Get(tt.node.Name)
			if !found {
				t.Fatal("node not found in store")
			}

			if updatedNode.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, updatedNode.Status)
			}
		})
	}
}
