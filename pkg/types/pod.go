package types

type PodSpec struct {
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	Command []string          `json:"command,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

type PodStatus string

const (
	StatusPending PodStatus = "Pending"
	StatusRunning PodStatus = "Running"
	StatusFailed  PodStatus = "Failed"
	StatusUnknown PodStatus = "Unknown"
)

type Pod struct {
	Spec        PodSpec   `json:"spec"`
	Status      PodStatus `json:"status"`
	ContainerID string    `json:"containerId,omitempty"`
	Message     string    `json:"message,omitempty"`
}

func NewPod(Spec PodSpec) Pod {
	return Pod{Spec: Spec, Status: StatusPending}
}
