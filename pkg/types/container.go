package types

type ContainerState struct {
	Status   ContainerStatus
	ExitCode int // if exited
}

type ContainerStatus string

const (
	ContainerStatusRunning     ContainerStatus = "Running"
	ContainerStatusTerminating ContainerStatus = "Terminating"
	ContainerStatusExited      ContainerStatus = "Exited"
	ContainerStatusUnknown     ContainerStatus = "Unknown"
)
