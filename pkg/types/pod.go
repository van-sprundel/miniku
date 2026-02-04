package types

import (
	"time"
)

type PodSpec struct {
	Name     string            `json:"name"`
	Image    string            `json:"image"`
	NodeName string            `json:"node_name"`
	Command  []string          `json:"command,omitempty"`
	Env      map[string]string `json:"env,omitempty"`
	Labels   map[string]string `json:"labels,omitempty"`
}

type PodStatus string

const (
	PodStatusPending PodStatus = "Pending"
	PodStatusRunning PodStatus = "Running"
	PodStatusFailed  PodStatus = "Failed"
	PodStatusUnknown PodStatus = "Unknown"
)

type Pod struct {
	Spec        PodSpec   `json:"spec"`
	Status      PodStatus `json:"status"`
	ContainerID string    `json:"containerId,omitempty"`
	Message     string    `json:"message,omitempty"`
	RetryCount  uint8     `json:"retry_count"`
	NextRetryAt time.Time `json:"next_retry_at"`
}

func NewPod(Spec PodSpec) Pod {
	return Pod{Spec: Spec, Status: PodStatusPending}
}
