/**
 * Runtime is there to "interact with the runtime (in this case docker)".
 * This mean the runtime should *not know anything* about the store and should only interact with said runtime.
 *
 *
 ```go
 package main

 import (
	"fmt"
	"miniku/pkg/runtime"
	"miniku/pkg/types"
 )

 func main() {
	r := &runtime.DockerCLIRuntime{}
	id, err := r.Run(types.PodSpec{Name: "test1", Image: "nginx"})
	fmt.Println(id, err)
 }
 ```
*/

package runtime

import (
	"miniku/pkg/types"
	"os/exec"
	"strings"
)

const DOCKER_FILTER_LABEL = "miniku=true"

// using the CLI to keep debugging easy for now. Can swap this lateron
type DockerCLIRuntime struct{}

func (r *DockerCLIRuntime) Run(pod types.PodSpec) (containerID string, err error) {
	args := []string{"run", "-d", "--label", DOCKER_FILTER_LABEL, "--name", pod.Name, pod.Image}
	args = append(args, pod.Command...)

	res, err := execCommand(args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(res)), nil
}

// `docker stop {id} {force && '--force'}`
func (r *DockerCLIRuntime) Stop(containerID string) error {
	_, err := execCommand("stop", containerID)
	return err
}

// `docker rm {id}`
func (r *DockerCLIRuntime) Remove(containerID string) error {
	_, err := execCommand("rm", containerID)
	return err
}

// `docker inspect {id}` -> JSON -> return status
func (r *DockerCLIRuntime) GetStatus(containerID string) (*types.ContainerState, error) {
	res, err := execCommand("inspect", "--format", "{{.State.Status}}", containerID)
	if err != nil {
		return nil, err
	}

	status := strings.TrimSpace(string(res))
	switch status {
	case "running":
		return &types.ContainerState{Status: types.ContainerStatusRunning}, nil
	case "exited":
		return &types.ContainerState{Status: types.ContainerStatusExited}, nil // TODO exitcode
	default:
		return &types.ContainerState{Status: types.ContainerStatusUnknown}, nil
	}

}

func execCommand(args ...string) ([]byte, error) {
	return exec.Command("docker", args...).Output()
}
