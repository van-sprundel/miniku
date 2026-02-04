/*
for each:
 1. Find all pods matching selector
 2. calc diff
 3. if too few, create pods
 4. if too many, delete pods
 5. and then update the current count
*/
package controller

import (
	"crypto/rand"
	"fmt"
	"miniku/pkg/store"
	"miniku/pkg/types"
	"time"
)

const POLL_INTERVAL_MS = 5000

type ReplicaSetController struct {
	podStore store.PodStore
	rsStore  store.ReplicaSetStore
}

func New(podStore store.PodStore, rsStore store.ReplicaSetStore) *ReplicaSetController {
	return &ReplicaSetController{
		podStore,
		rsStore,
	}
}

func (c *ReplicaSetController) Run() {
	for {
		for _, rs := range c.rsStore.List() {
			c.reconcile(rs)
		}

		time.Sleep(time.Millisecond * POLL_INTERVAL_MS)
	}

}
func (c *ReplicaSetController) reconcile(rs types.ReplicaSet) error {
	matchingPods := c.getMatchingPods(rs)
	current := uint(len(matchingPods))
	desired := rs.DesiredCount

	if current < desired {
		diff := desired - current
		for range diff {
			if err := c.createPod(rs); err != nil {
				return err
			}
		}
	}

	if current > desired {
		diff := current - desired
		for i := range diff {
			if err := c.deletePod(matchingPods[i]); err != nil {
				return err
			}
		}
	}

	// TODO while this is more robust than calculating (pods mightve silently failed), there's probably a more optimal way
	rs.CurrentCount = uint(len(c.getMatchingPods(rs)))
	c.rsStore.Put(rs.Name, rs)
	return nil
}
func (c *ReplicaSetController) getMatchingPods(rs types.ReplicaSet) []types.Pod {
	var result []types.Pod
	for _, pod := range c.podStore.List() {
		if matchesSelector(pod, rs.Selector) {
			result = append(result, pod)
		}
	}
	return result
}

// a better alternative would be to use uuid's for this
// but this is fine for a toy
func generatePodName(rsName string) string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%s-%x", rsName, b)
}

func (c *ReplicaSetController) createPod(rs types.ReplicaSet) error {
	pod := types.Pod{
		Spec: types.PodSpec{
			Name:    generatePodName(rs.Name),
			Image:   rs.Template.Image,
			Command: rs.Template.Command,
			Labels:  rs.Selector, // so it matches back to this RS
		},
		Status: types.PodStatusPending,
	}
	c.podStore.Put(pod.Spec.Name, pod)
	return nil
}
func (c *ReplicaSetController) deletePod(pod types.Pod) error {
	c.podStore.Delete(pod.Spec.Name)
	return nil
}

func matchesSelector(pod types.Pod, selector map[string]string) bool {
	for key, value := range selector {
		podValue, exists := pod.Spec.Labels[key]
		if !exists || podValue != value {
			return false
		}
	}
	return true
}
