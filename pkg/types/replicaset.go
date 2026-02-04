package types

type ReplicaSet struct {
	Name         string            `json:"name"`
	DesiredCount uint              `json:"desiredCount"`
	CurrentCount uint              `json:"currentCount"`
	Selector     map[string]string `json:"selector"`
	Template     PodSpec           `json:"template"`
}
