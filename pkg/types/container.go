package types

type ContainerState struct {
	Status   ContainerStatus
	ExitCode int // if exited
}

type ContainerStatus string

const (
	ContainerStatusRunning ContainerStatus = "Running"
	ContainerStatusExited  ContainerStatus = "Exited"
	ContainerStatusUnknown ContainerStatus = "Unknown"
)
