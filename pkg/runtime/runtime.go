/// For the sake of simplicity our runtime only supports the following func.
//
// - Create/Start a container from an image with a command
// - Stop/Kill a container
// - Remove a container
// - Get status of a container
// - List containers

package runtime

import (
	"miniku/pkg/types"
)

type Runtime interface {
	Run(pod types.PodSpec) (containerID string, err error)
	Stop(containerID string) error
	Remove(containerID string) error
	GetStatus(containerID string) (*types.ContainerStatus, error)
}
