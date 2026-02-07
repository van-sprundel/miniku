package scheduler

import (
	"errors"
	"log"
	"miniku/pkg/client"
	"miniku/pkg/types"
	"sort"
	"time"
)

type Scheduler struct {
	client       *client.Client
	nextIndex    uint
	PollInterval time.Duration
}

func New(client *client.Client) *Scheduler {
	return &Scheduler{
		client:       client,
		PollInterval: 5 * time.Second,
	}
}

func (s *Scheduler) Run() {
	for {
		pods, err := s.client.ListPods()
		if err != nil {
			log.Printf("scheduler: failed to list pods: %v", err)
			time.Sleep(s.PollInterval)
			continue
		}

		for _, pod := range pods {
			// pod is unscheduled
			if pod.Spec.NodeName == "" {
				err := s.scheduleOne(pod)
				if err != nil {
					log.Println(err)
				}
			}
		}

		time.Sleep(s.PollInterval)
	}
}

// schedule a single pod
func (s *Scheduler) scheduleOne(pod types.Pod) error {
	node, ok := s.pickNode()
	if !ok {
		return errors.New("no node available for scheduling")
	}

	log.Printf("scheduler: assigning pod %s to node %s", pod.Spec.Name, node.Name)
	pod.Spec.NodeName = node.Name
	if err := s.client.UpdatePod(pod.Spec.Name, pod); err != nil {
		return err
	}

	return nil
}

// we'll pick round-robin as the strategy
func (s *Scheduler) pickNode() (types.Node, bool) {
	availableNodes := s.getAvailableNodes()
	if len(availableNodes) == 0 {
		return types.Node{}, false
	}

	node := availableNodes[s.nextIndex%uint(len(availableNodes))]
	s.nextIndex++
	return node, true
}

// filter Ready nodes
func (s *Scheduler) getAvailableNodes() []types.Node {
	nodes, err := s.client.ListNodes()
	if err != nil {
		log.Printf("scheduler: failed to list nodes: %v", err)
		return nil
	}

	filteredNodes := []types.Node{}
	for _, v := range nodes {
		if v.Status == types.NodeStateReady {
			filteredNodes = append(filteredNodes, v)
		}
	}

	// sort for deterministic round-robin
	sort.Slice(filteredNodes, func(i, j int) bool {
		return filteredNodes[i].Name < filteredNodes[j].Name
	})

	return filteredNodes
}
