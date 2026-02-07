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
	"log"
	"miniku/pkg/client"
	"miniku/pkg/types"
	"time"
)

type ReplicaSetController struct {
	client       *client.Client
	PollInterval time.Duration
}

func New(client *client.Client) *ReplicaSetController {
	return &ReplicaSetController{
		client:       client,
		PollInterval: 5 * time.Second,
	}
}

func (c *ReplicaSetController) Run() {
	for {
		rsList, err := c.client.ListReplicaSets()
		if err != nil {
			log.Printf("controller: failed to list replicasets: %v", err)
			time.Sleep(c.PollInterval)
			continue
		}

		for _, rs := range rsList {
			if err := c.reconcile(rs); err != nil {
				log.Printf("controller: failed to reconcile %s: %v", rs.Name, err)
			}
		}

		time.Sleep(c.PollInterval)
	}
}
func (c *ReplicaSetController) reconcile(rs types.ReplicaSet) error {
	matchingPods, err := c.getMatchingPods(rs)
	if err != nil {
		return err
	}
	current := uint(len(matchingPods))
	desired := rs.DesiredCount

	if current < desired {
		diff := desired - current
		for range diff {
			if err := c.createPod(rs); err != nil {
				return err
			}
		}
		current += diff
	}

	if current > desired {
		diff := current - desired
		for i := range diff {
			if err := c.deletePod(matchingPods[i]); err != nil {
				return err
			}
		}
		current -= diff
	}

	rs.CurrentCount = current
	return c.client.UpdateReplicaSet(rs.Name, rs)
}
func (c *ReplicaSetController) getMatchingPods(rs types.ReplicaSet) ([]types.Pod, error) {
	pods, err := c.client.ListPods()
	if err != nil {
		return nil, err
	}

	var result []types.Pod
	for _, pod := range pods {
		if pod.Status == types.PodStatusFailed {
			continue
		}
		if matchesSelector(pod, rs.Selector) {
			result = append(result, pod)
		}
	}
	return result, nil
}

// a better alternative would be to use uuid's for this
// but this is fine for a toy
func generatePodName(rsName string) string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		log.Printf("warning: failed to generate random bytes: %v", err)
	}
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
	return c.client.CreatePod(pod)
}
func (c *ReplicaSetController) deletePod(pod types.Pod) error {
	return c.client.DeletePod(pod.Spec.Name)
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
