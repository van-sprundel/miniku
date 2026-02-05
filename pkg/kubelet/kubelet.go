package kubelet

import (
	"fmt"
	"log"
	"miniku/pkg/runtime"
	"miniku/pkg/store"
	"miniku/pkg/types"
	"time"
)

const POLL_INTERVAL = time.Millisecond * 5000

const MAX_RETRY_COUNT = 3

const BASE_DELAY time.Duration = 1000
const MAX_DELAY time.Duration = 60_000

type Kubelet struct {
	name      string
	podStore  store.PodStore
	nodeStore store.NodeStore
	runtime   runtime.Runtime
}

func New(podStore store.PodStore, nodeStore store.NodeStore, runtime runtime.Runtime, name string) Kubelet {
	return Kubelet{
		name,
		podStore,
		nodeStore,
		runtime,
	}
}

// rediscover existing containers on startup and link them to pods.
// orphaned containers (no matching pod) are stopped and removed.
//
// NOTE: this doesn't work right now because we're running the store in-memory so state will be lost
func (k *Kubelet) Sync() {
	containers, err := k.runtime.List()
	if err != nil {
		log.Printf("sync: failed to list containers: %v", err)
		return
	}

	for _, container := range containers {
		pod, exists := k.podStore.Get(container.Name)
		if !exists {
			k.removeContainer(container.Name, container.ID)
			continue
		}

		// only manage containers for pods assigned to this node
		if pod.Spec.NodeName != k.name {
			continue
		}

		if pod.ContainerID == "" || pod.ContainerID != container.ID {
			log.Printf("sync: linking container %s to pod %s", container.ID, pod.Spec.Name)
			pod.ContainerID = container.ID
			pod.Status = types.PodStatusRunning
			k.podStore.Put(pod.Spec.Name, pod)
		}
	}
}

// cleanupOrphanedContainers stops and removes containers whose pods
// no longer exist in the store (e.g. deleted via API or scaled down).
func (k *Kubelet) cleanupOrphanedContainers() {
	containers, err := k.runtime.List()
	if err != nil {
		log.Printf("kubelet: failed to list containers for cleanup: %v", err)
		return
	}

	for _, container := range containers {
		if _, exists := k.podStore.Get(container.Name); !exists {
			k.removeContainer(container.Name, container.ID)
		}
	}
}

func (k *Kubelet) removeContainer(name string, id string) {
	log.Printf("kubelet: removing orphan container %s (%s)", name, id)
	if err := k.runtime.Stop(id); err != nil {
		log.Printf("kubelet: failed to stop container %s: %v", id, err)
	}
	if err := k.runtime.Remove(id); err != nil {
		log.Printf("kubelet: failed to remove container %s: %v", id, err)
	}
}

func (k *Kubelet) Run() {
	k.Sync()

	for {
		for _, pod := range k.podStore.List() {
			// only reconcile pods assigned to this node
			if pod.Spec.NodeName != k.name {
				continue
			}
			if err := k.reconcilePod(pod); err != nil {
				log.Printf("kubelet: failed to reconcile pod %s: %v", pod.Spec.Name, err)
			}
		}

		k.cleanupOrphanedContainers()

		// polling
		k.updateHeartbeat()
		time.Sleep(POLL_INTERVAL)
	}
}

func (k *Kubelet) reconcilePod(pod types.Pod) error {
	podStatus := pod.Status

	var containerState *types.ContainerState
	if pod.ContainerID != "" {
		var err error
		containerState, err = k.runtime.GetStatus(pod.ContainerID)
		if err != nil {
			// container not found
			containerState = nil
		}
	}

	var updatedPod types.Pod
	var err error
	switch {
	// pending but container not existing yet
	case podStatus == types.PodStatusPending && containerState == nil:
		// not time to retry yet
		if time.Now().Before(pod.NextRetryAt) {
			return nil
		}
		updatedPod, err = k.createAndRun(pod)
		if err != nil {
			return err
		}

	// pending and created but should be running
	case podStatus == types.PodStatusPending && containerState.Status == types.ContainerStatusRunning:
		updatedPod = k.updatePodStatus(pod, types.PodStatusRunning)

	// happy flow, return
	case podStatus == types.PodStatusRunning && containerState != nil && containerState.Status == types.ContainerStatusRunning:
		return nil

	// expected running but not running, mark as failed
	case podStatus == types.PodStatusRunning && containerState != nil && containerState.Status != types.ContainerStatusRunning:
		updatedPod = k.updatePodStatus(pod, types.PodStatusFailed)

	// expected running but contianer missing, handle (reconcile)
	case podStatus == types.PodStatusRunning && containerState == nil:
		updatedPod = k.handleMissingContainer(pod)

	default:
		return fmt.Errorf("unhandled state")
	}

	k.podStore.Put(updatedPod.Spec.Name, updatedPod)
	return nil
}

// this function should backoff + retry when runtime.Run() fails.
func (k *Kubelet) createAndRun(pod types.Pod) (types.Pod, error) {

	cID, err := k.runtime.Run(pod.Spec)
	if err != nil {
		if pod.RetryCount == MAX_RETRY_COUNT {
			pod.Status = types.PodStatusFailed
			return pod, nil
		}

		pod.RetryCount++
		pod.NextRetryAt = calculateNextRetry(pod)
		return pod, nil
	}

	pod.ContainerID = cID
	pod.RetryCount = 0
	pod.Status = types.PodStatusRunning
	return pod, nil
}

func (k *Kubelet) updatePodStatus(pod types.Pod, status types.PodStatus) types.Pod {
	pod.Status = status
	return pod
}

// for this impl we're just going to recreate the missing container
// normally this would be based on some strategy
//
// due to my lazyness I'm going to mark the pod as pending so it can be retried
func (k *Kubelet) handleMissingContainer(pod types.Pod) types.Pod {
	pod.Status = types.PodStatusPending
	pod.ContainerID = "" // clear stale cID
	return pod
}

func calculateNextRetry(pod types.Pod) time.Time {
	delay := min(MAX_DELAY, BASE_DELAY*time.Duration(1<<pod.RetryCount))
	return time.Now().Add(delay)
}

func (k *Kubelet) updateHeartbeat() {
	node, ok := k.nodeStore.Get(k.name)
	if ok {
		node.LastHeartbeat = time.Now()
		k.nodeStore.Put(k.name, node)
	}
}
