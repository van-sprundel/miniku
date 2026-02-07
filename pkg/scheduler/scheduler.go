package scheduler

import (
	"errors"
	"log"
	"miniku/pkg/store"
	"miniku/pkg/types"
	"time"
)

type Scheduler struct {
	podStore     store.PodStore
	nodeStore    store.NodeStore
	nextIndex    uint
	PollInterval time.Duration
}

func New(podStore store.PodStore, nodeStore store.NodeStore) *Scheduler {
	return &Scheduler{
		podStore:     podStore,
		nodeStore:    nodeStore,
		PollInterval: 5 * time.Second,
	}
}

func (s *Scheduler) Run() {
	for {
		for _, pod := range s.podStore.List() {
			// pod is unscheduled
			if pod.Spec.NodeName == "" {
				err := s.scheduleOne(pod)
				if err != nil {
					log.Println(err)
					//TODO what to do here?
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
	s.podStore.Put(pod.Spec.Name, pod)

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
	filteredNodes := []types.Node{}

	for _, v := range s.nodeStore.List() {
		if v.Status == types.NodeStateReady {
			filteredNodes = append(filteredNodes, v)
		}
	}

	return filteredNodes
}
