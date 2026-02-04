package types

type ReplicaSet struct {
	Name         string
	DesiredCount uint
	CurrentCount uint
	Selector     map[string]string // e.g., {"app": "nginx"}
	Template     PodSpec
}
