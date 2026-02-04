package kubelet

import (
	"fmt"
	"miniku/pkg/runtime"
	"miniku/pkg/store"
	"miniku/pkg/types"
	"time"
)

const POLL_INTERVAL_MS = 5000

const MAX_RETRY_COUNT = 3

const BASE_DELAY_MS time.Duration = 1000
const MAX_DELAY_MS time.Duration = 60_000

type Kubelet struct {
	store   store.Store
	runtime runtime.Runtime
}

func New(store store.Store, runtime runtime.Runtime) Kubelet {
	return Kubelet{
		store,
		runtime,
	}
}

// infinite loop with time.Sleep
// list all pods from store
// for each, compare deired vs actual
// take action to converge
func (k *Kubelet) Run() {
	for {
		for _, pod := range k.store.List() {
			k.reconcilePod(pod)
		}

		// polling
		time.Sleep(time.Millisecond * POLL_INTERVAL_MS)
	}

}

func (k *Kubelet) reconcilePod(pod types.Pod) error {
	podStatus := pod.Status
	containerState, err := k.runtime.GetStatus(pod.ContainerID)
	if err != nil {
		//TODO handle each err properly
		// e.g. container not found should reconcile into re-creating pod
		return err
	}

	var updatedPod types.Pod
	switch {
	// pending but container not existing yet
	case podStatus == types.PodStatusPending && containerState == nil:
		// not time to retry yet
		if time.Now().Before(pod.NextRetryAt) {
			return nil
		}
		updatedPod, err = k.createAndRun(pod)
		if err != nil {
			// TODO panic?
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

	k.store.Put(updatedPod)
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
	delay := min(MAX_DELAY_MS, BASE_DELAY_MS*time.Duration(1<<pod.RetryCount))
	return time.Now().Add(delay)
}
