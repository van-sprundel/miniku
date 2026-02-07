package integration

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"miniku/pkg/controller"
	"miniku/pkg/kubelet"
	"miniku/pkg/runtime"
	"miniku/pkg/scheduler"
	"miniku/pkg/store"
	"miniku/pkg/types"
)

type mockRuntime struct {
	mu         sync.Mutex
	containers map[string]*mockContainer
	nextID     atomic.Uint64
}

type mockContainer struct {
	id     string
	name   string
	status types.ContainerStatus
}

func newMockRuntime() *mockRuntime {
	return &mockRuntime{
		containers: make(map[string]*mockContainer),
	}
}

func (r *mockRuntime) Run(spec types.PodSpec) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := fmt.Sprintf("ctr-%d", r.nextID.Add(1))
	r.containers[id] = &mockContainer{
		id:     id,
		name:   spec.Name,
		status: types.ContainerStatusRunning,
	}
	return id, nil
}

func (r *mockRuntime) Stop(containerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	c, ok := r.containers[containerID]
	if !ok {
		return fmt.Errorf("container %s not found", containerID)
	}
	c.status = types.ContainerStatusExited
	return nil
}

func (r *mockRuntime) Remove(containerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.containers, containerID)
	return nil
}

func (r *mockRuntime) GetStatus(containerID string) (*types.ContainerState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	c, ok := r.containers[containerID]
	if !ok {
		return nil, fmt.Errorf("container %s not found", containerID)
	}
	return &types.ContainerState{Status: c.status}, nil
}

func (r *mockRuntime) List() ([]runtime.ContainerInfo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var out []runtime.ContainerInfo
	for _, c := range r.containers {
		out = append(out, runtime.ContainerInfo{ID: c.id, Name: c.name})
	}
	return out, nil
}

func (r *mockRuntime) crashContainer(containerID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if c, ok := r.containers[containerID]; ok {
		c.status = types.ContainerStatusExited
	}
}

// func (r *mockRuntime) removeContainer(containerID string) {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()
//
// 	delete(r.containers, containerID)
// }

func (r *mockRuntime) containerCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.containers)
}

type cluster struct {
	podStore  store.PodStore
	rsStore   store.ReplicaSetStore
	nodeStore store.NodeStore
	rt        *mockRuntime
}

const testPollInterval = 50 * time.Millisecond

func newCluster() *cluster {
	podStore := store.NewMemStore[types.Pod]()
	rsStore := store.NewMemStore[types.ReplicaSet]()
	nodeStore := store.NewMemStore[types.Node]()
	rt := newMockRuntime()

	nodeStore.Put("node-1", types.Node{Name: "node-1", Status: types.NodeStateReady, LastHeartbeat: time.Now()})
	nodeStore.Put("node-2", types.Node{Name: "node-2", Status: types.NodeStateReady, LastHeartbeat: time.Now()})

	sched := scheduler.New(podStore, nodeStore)
	sched.PollInterval = testPollInterval
	go sched.Run()

	k1 := kubelet.New(podStore, nodeStore, rt, "node-1")
	k1.PollInterval = testPollInterval
	go k1.Run()

	k2 := kubelet.New(podStore, nodeStore, rt, "node-2")
	k2.PollInterval = testPollInterval
	go k2.Run()

	rsCtrl := controller.New(podStore, rsStore)
	rsCtrl.PollInterval = testPollInterval
	go rsCtrl.Run()

	nodeCtrl := controller.NewNodeController(nodeStore)
	nodeCtrl.PollInterval = testPollInterval
	go nodeCtrl.Run()

	return &cluster{
		podStore:  podStore,
		rsStore:   rsStore,
		nodeStore: nodeStore,
		rt:        rt,
	}
}

// poll a condition until it returns true or the timeout expires.
func waitFor(t *testing.T, timeout time.Duration, desc string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for: %s", desc)
}

func TestReplicaSetCreatesPods(t *testing.T) {
	c := newCluster()

	c.rsStore.Put("web", types.ReplicaSet{
		Name:         "web",
		DesiredCount: 3,
		Selector:     map[string]string{"app": "web"},
		Template:     types.PodSpec{Image: "nginx"},
	})

	waitFor(t, 5*time.Second, "3 pods running", func() bool {
		running := 0
		for _, pod := range c.podStore.List() {
			if pod.Status == types.PodStatusRunning {
				running++
			}
		}
		return running == 3
	})

	if got := c.rt.containerCount(); got != 3 {
		t.Errorf("expected 3 containers, got %d", got)
	}

	// verify pods are spread across nodes
	nodes := map[string]int{}
	for _, pod := range c.podStore.List() {
		nodes[pod.Spec.NodeName]++
	}
	if len(nodes) < 2 {
		t.Errorf("expected pods on at least 2 nodes, got %v", nodes)
	}
}

func TestScaleUp(t *testing.T) {
	c := newCluster()

	c.rsStore.Put("web", types.ReplicaSet{
		Name:         "web",
		DesiredCount: 2,
		Selector:     map[string]string{"app": "web"},
		Template:     types.PodSpec{Image: "nginx"},
	})

	waitFor(t, 5*time.Second, "2 pods running", func() bool {
		running := 0
		for _, pod := range c.podStore.List() {
			if pod.Status == types.PodStatusRunning {
				running++
			}
		}
		return running == 2
	})

	// scale up to 5
	rs, _ := c.rsStore.Get("web")
	rs.DesiredCount = 5
	c.rsStore.Put("web", rs)

	waitFor(t, 5*time.Second, "5 pods running", func() bool {
		running := 0
		for _, pod := range c.podStore.List() {
			if pod.Status == types.PodStatusRunning {
				running++
			}
		}
		return running == 5
	})

	if got := c.rt.containerCount(); got != 5 {
		t.Errorf("expected 5 containers, got %d", got)
	}
}

func TestScaleDown(t *testing.T) {
	c := newCluster()

	c.rsStore.Put("web", types.ReplicaSet{
		Name:         "web",
		DesiredCount: 4,
		Selector:     map[string]string{"app": "web"},
		Template:     types.PodSpec{Image: "nginx"},
	})

	waitFor(t, 5*time.Second, "4 pods running", func() bool {
		running := 0
		for _, pod := range c.podStore.List() {
			if pod.Status == types.PodStatusRunning {
				running++
			}
		}
		return running == 4
	})

	// scale down to 1
	rs, _ := c.rsStore.Get("web")
	rs.DesiredCount = 1
	c.rsStore.Put("web", rs)

	waitFor(t, 5*time.Second, "1 pod remaining", func() bool {
		return len(c.podStore.List()) == 1
	})

	waitFor(t, 5*time.Second, "1 container remaining", func() bool {
		return c.rt.containerCount() == 1
	})
}

func TestContainerCrashRecovery(t *testing.T) {
	c := newCluster()

	c.rsStore.Put("web", types.ReplicaSet{
		Name:         "web",
		DesiredCount: 1,
		Selector:     map[string]string{"app": "web"},
		Template:     types.PodSpec{Image: "nginx"},
	})

	waitFor(t, 5*time.Second, "1 pod running", func() bool {
		for _, pod := range c.podStore.List() {
			if pod.Status == types.PodStatusRunning {
				return true
			}
		}
		return false
	})

	// crash the container
	pods := c.podStore.List()
	crashedID := pods[0].ContainerID
	c.rt.crashContainer(crashedID)

	// the kubelet should detect the crash (pod → Failed),
	// then the RS controller creates a replacement pod,
	// which the scheduler assigns and the kubelet starts.
	waitFor(t, 10*time.Second, "replacement pod running", func() bool {
		for _, pod := range c.podStore.List() {
			if pod.Status == types.PodStatusRunning && pod.ContainerID != crashedID {
				return true
			}
		}
		return false
	})
}

func TestDeletePodCleansUpContainer(t *testing.T) {
	c := newCluster()

	// create a single pod directly (not via RS)
	c.podStore.Put("solo", types.Pod{
		Spec:   types.PodSpec{Name: "solo", Image: "nginx"},
		Status: types.PodStatusPending,
	})

	waitFor(t, 5*time.Second, "pod running", func() bool {
		pod, ok := c.podStore.Get("solo")
		return ok && pod.Status == types.PodStatusRunning
	})

	if got := c.rt.containerCount(); got != 1 {
		t.Fatalf("expected 1 container before delete, got %d", got)
	}

	// delete the pod
	c.podStore.Delete("solo")

	// kubelet should clean up the orphaned container
	waitFor(t, 5*time.Second, "container cleaned up", func() bool {
		return c.rt.containerCount() == 0
	})
}

func TestMultipleReplicaSets(t *testing.T) {
	c := newCluster()

	c.rsStore.Put("web", types.ReplicaSet{
		Name:         "web",
		DesiredCount: 2,
		Selector:     map[string]string{"app": "web"},
		Template:     types.PodSpec{Image: "nginx"},
	})
	c.rsStore.Put("api", types.ReplicaSet{
		Name:         "api",
		DesiredCount: 3,
		Selector:     map[string]string{"app": "api"},
		Template:     types.PodSpec{Image: "node"},
	})

	waitFor(t, 5*time.Second, "5 total pods running", func() bool {
		running := 0
		for _, pod := range c.podStore.List() {
			if pod.Status == types.PodStatusRunning {
				running++
			}
		}
		return running == 5
	})

	// verify label separation
	webPods, apiPods := 0, 0
	for _, pod := range c.podStore.List() {
		if pod.Spec.Labels["app"] == "web" {
			webPods++
		}
		if pod.Spec.Labels["app"] == "api" {
			apiPods++
		}
	}
	if webPods != 2 {
		t.Errorf("expected 2 web pods, got %d", webPods)
	}
	if apiPods != 3 {
		t.Errorf("expected 3 api pods, got %d", apiPods)
	}

	if got := c.rt.containerCount(); got != 5 {
		t.Errorf("expected 5 containers, got %d", got)
	}
}

func TestDeleteReplicaSetCleansUpPods(t *testing.T) {
	c := newCluster()

	c.rsStore.Put("web", types.ReplicaSet{
		Name:         "web",
		DesiredCount: 3,
		Selector:     map[string]string{"app": "web"},
		Template:     types.PodSpec{Image: "nginx"},
	})

	waitFor(t, 5*time.Second, "3 pods running", func() bool {
		running := 0
		for _, pod := range c.podStore.List() {
			if pod.Status == types.PodStatusRunning {
				running++
			}
		}
		return running == 3
	})

	// delete the RS and scale to 0
	rs, _ := c.rsStore.Get("web")
	rs.DesiredCount = 0
	c.rsStore.Put("web", rs)

	waitFor(t, 5*time.Second, "all pods removed", func() bool {
		return len(c.podStore.List()) == 0
	})

	waitFor(t, 5*time.Second, "all containers removed", func() bool {
		return c.rt.containerCount() == 0
	})
}
