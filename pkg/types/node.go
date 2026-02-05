package types

import "time"

type Node struct {
	Name          string    `json:"name"`
	Status        NodeState `json:"status"`
	LastHeartbeat time.Time `json:"time"`
}

type NodeState string

const (
	NodeStateNotReady NodeState = "NotReady"
	NodeStateReady    NodeState = "Ready"
)
