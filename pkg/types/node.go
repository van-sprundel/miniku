package types

type Node struct {
	Name   string    `json:"name"`
	Status NodeState `json:"status"`
}

type NodeState string

const (
	NodeStateNotReady NodeState = "NotReady"
	NodeStateReady    NodeState = "Ready"
)
