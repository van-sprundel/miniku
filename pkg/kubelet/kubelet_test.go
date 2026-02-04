package kubelet

import (
	"errors"
	"miniku/pkg/runtime"
	"miniku/pkg/store"
	"miniku/pkg/types"
	"slices"
	"testing"
	"time"
)

type mockRuntime struct {
	runFunc       func(types.PodSpec) (string, error)
	stopFunc      func(string) error
	removeFunc    func(string) error
	getStatusFunc func(string) (*types.ContainerState, error)
	listFunc      func() ([]runtime.ContainerInfo, error)
}

func (r *mockRuntime) Run(spec types.PodSpec) (string, error) {
	if r.runFunc != nil {
		return r.runFunc(spec)
	}
	return "1", nil
}

func (r *mockRuntime) Stop(containerID string) error {
	if r.stopFunc != nil {
		return r.stopFunc(containerID)
	}
	return nil
}

func (r *mockRuntime) Remove(containerID string) error {
	if r.removeFunc != nil {
		return r.removeFunc(containerID)
	}
	return nil
}

func (r *mockRuntime) GetStatus(containerID string) (*types.ContainerState, error) {
	if r.getStatusFunc != nil {
		return r.getStatusFunc(containerID)
	}
	return nil, nil
}

func (r *mockRuntime) List() ([]runtime.ContainerInfo, error) {
	if r.listFunc != nil {
		return r.listFunc()
	}
	return []runtime.ContainerInfo{}, nil
}

func TestCreateAndRun(t *testing.T) {
	tests := []struct {
		name                string
		pod                 types.Pod
		runFunc             func(types.PodSpec) (string, error)
		expectedStatus      types.PodStatus
		expectedRetry       uint8
		expectedContainerID string
		expectNextRetrySet  bool
	}{
		{
			name: "success",
			pod: types.Pod{
				Spec:   types.PodSpec{Name: "test-pod", Image: "nginx"},
				Status: types.PodStatusPending,
			},
			runFunc: func(spec types.PodSpec) (string, error) {
				return "container-123", nil
			},
			expectedStatus:      types.PodStatusRunning,
			expectedRetry:       0,
			expectedContainerID: "container-123",
			expectNextRetrySet:  false,
		},
		{
			name: "first failure",
			pod: types.Pod{
				Spec:       types.PodSpec{Name: "test-pod", Image: "nginx"},
				Status:     types.PodStatusPending,
				RetryCount: 0,
			},
			runFunc: func(spec types.PodSpec) (string, error) {
				return "", errors.New("container failed to start")
			},
			expectedStatus:      types.PodStatusPending,
			expectedRetry:       1,
			expectedContainerID: "",
			expectNextRetrySet:  true,
		},
		{
			name: "second failure",
			pod: types.Pod{
				Spec:       types.PodSpec{Name: "test-pod", Image: "nginx"},
				Status:     types.PodStatusPending,
				RetryCount: 1,
			},
			runFunc: func(spec types.PodSpec) (string, error) {
				return "", errors.New("container failed to start")
			},
			expectedStatus:      types.PodStatusPending,
			expectedRetry:       2,
			expectedContainerID: "",
			expectNextRetrySet:  true,
		},
		{
			name: "max retries exceeded",
			pod: types.Pod{
				Spec:       types.PodSpec{Name: "test-pod", Image: "nginx"},
				Status:     types.PodStatusPending,
				RetryCount: MAX_RETRY_COUNT,
			},
			runFunc: func(spec types.PodSpec) (string, error) {
				return "", errors.New("container failed to start")
			},
			expectedStatus:      types.PodStatusFailed,
			expectedRetry:       MAX_RETRY_COUNT,
			expectedContainerID: "",
			expectNextRetrySet:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRT := &mockRuntime{runFunc: tt.runFunc}
			memStore := store.NewMemStore[types.Pod]()
			kubelet := New(memStore, mockRT, "node-1")

			beforeCall := time.Now()
			result, _ := kubelet.createAndRun(tt.pod)

			if result.Status != tt.expectedStatus {
				t.Errorf("expected status %s, got %s", tt.expectedStatus, result.Status)
			}

			if result.RetryCount != tt.expectedRetry {
				t.Errorf("expected retry count %d, got %d", tt.expectedRetry, result.RetryCount)
			}

			if result.ContainerID != tt.expectedContainerID {
				t.Errorf("expected container ID %s, got %s", tt.expectedContainerID, result.ContainerID)
			}

			if tt.expectNextRetrySet {
				if !result.NextRetryAt.After(beforeCall) {
					t.Errorf("expected NextRetryAt to be set in the future, got %v", result.NextRetryAt)
				}
			}
		})
	}
}

func TestReconcilePod(t *testing.T) {
	tests := []struct {
		name              string
		pod               types.Pod
		containerState    *types.ContainerState
		getStatusErr      error
		runFunc           func(types.PodSpec) (string, error)
		expectedStatus    types.PodStatus
		expectedContainer string
		expectStoreUpdate bool
	}{
		{
			name: "pending + no container -> create and run",
			pod: types.Pod{
				Spec:   types.PodSpec{Name: "test-pod", Image: "nginx"},
				Status: types.PodStatusPending,
			},
			containerState: nil,
			runFunc: func(spec types.PodSpec) (string, error) {
				return "new-container-id", nil
			},
			expectedStatus:    types.PodStatusRunning,
			expectedContainer: "new-container-id",
			expectStoreUpdate: true,
		},
		{
			name: "pending + container running -> update to running",
			pod: types.Pod{
				Spec:        types.PodSpec{Name: "test-pod", Image: "nginx"},
				Status:      types.PodStatusPending,
				ContainerID: "existing-container",
			},
			containerState: &types.ContainerState{
				Status: types.ContainerStatusRunning,
			},
			expectedStatus:    types.PodStatusRunning,
			expectedContainer: "existing-container",
			expectStoreUpdate: true,
		},
		{
			name: "running + running -> no change (converged)",
			pod: types.Pod{
				Spec:        types.PodSpec{Name: "test-pod", Image: "nginx"},
				Status:      types.PodStatusRunning,
				ContainerID: "running-container",
			},
			containerState: &types.ContainerState{
				Status: types.ContainerStatusRunning,
			},
			expectedStatus:    types.PodStatusRunning,
			expectedContainer: "running-container",
			expectStoreUpdate: false,
		},
		{
			name: "running + exited -> mark failed",
			pod: types.Pod{
				Spec:        types.PodSpec{Name: "test-pod", Image: "nginx"},
				Status:      types.PodStatusRunning,
				ContainerID: "exited-container",
			},
			containerState: &types.ContainerState{
				Status:   types.ContainerStatusExited,
				ExitCode: 1,
			},
			expectedStatus:    types.PodStatusFailed,
			expectedContainer: "exited-container",
			expectStoreUpdate: true,
		},
		{
			name: "running + container missing -> reset to pending",
			pod: types.Pod{
				Spec:        types.PodSpec{Name: "test-pod", Image: "nginx"},
				Status:      types.PodStatusRunning,
				ContainerID: "vanished-container",
			},
			containerState:    nil,
			expectedStatus:    types.PodStatusPending,
			expectedContainer: "", // cleared
			expectStoreUpdate: true,
		},
		{
			name: "pending + no container + backoff active -> skip",
			pod: types.Pod{
				Spec:        types.PodSpec{Name: "test-pod", Image: "nginx"},
				Status:      types.PodStatusPending,
				NextRetryAt: time.Now().Add(time.Hour),
			},
			containerState:    nil,
			expectStoreUpdate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRT := &mockRuntime{
				runFunc: tt.runFunc,
				getStatusFunc: func(containerID string) (*types.ContainerState, error) {
					return tt.containerState, tt.getStatusErr
				},
			}
			memStore := store.NewMemStore[types.Pod]()
			memStore.Put(tt.pod.Spec.Name, tt.pod)
			kubelet := New(memStore, mockRT, "node-1")

			kubelet.reconcilePod(tt.pod)

			storedPod, found := memStore.Get(tt.pod.Spec.Name)

			if tt.expectStoreUpdate {
				if !found {
					t.Fatal("expected pod to be in store")
				}
				if storedPod.Status != tt.expectedStatus {
					t.Errorf("expected status %s, got %s", tt.expectedStatus, storedPod.Status)
				}
				if storedPod.ContainerID != tt.expectedContainer {
					t.Errorf("expected container ID %s, got %s", tt.expectedContainer, storedPod.ContainerID)
				}
			} else {
				// no-update cases
				// just verify original pod is unchanged
				if found && storedPod.Status != tt.pod.Status {
					t.Errorf("expected status to remain %s, got %s", tt.pod.Status, storedPod.Status)
				}
			}
		})
	}
}

func TestSync(t *testing.T) {
	tests := []struct {
		name               string
		existingPods       []types.Pod
		existingContainers []runtime.ContainerInfo
		expectedPodState   map[string]types.Pod // pod name -> expected state after sync
		expectStopped      []string             // container IDs that should be stopped
		expectRemoved      []string             // container IDs that should be removed
	}{
		{
			name: "link container to pod",
			existingPods: []types.Pod{
				{Spec: types.PodSpec{Name: "nginx-1", NodeName: "node-1"}, Status: types.PodStatusPending},
			},
			existingContainers: []runtime.ContainerInfo{
				{ID: "abc123", Name: "nginx-1"},
			},
			expectedPodState: map[string]types.Pod{
				"nginx-1": {
					Spec:        types.PodSpec{Name: "nginx-1", NodeName: "node-1"},
					Status:      types.PodStatusRunning,
					ContainerID: "abc123",
				},
			},
			expectStopped: []string{},
			expectRemoved: []string{},
		},
		{
			name: "container already linked",
			existingPods: []types.Pod{
				{Spec: types.PodSpec{Name: "nginx-1", NodeName: "node-1"}, Status: types.PodStatusRunning, ContainerID: "abc123"},
			},
			existingContainers: []runtime.ContainerInfo{
				{ID: "abc123", Name: "nginx-1"},
			},
			expectedPodState: map[string]types.Pod{
				"nginx-1": {
					Spec:        types.PodSpec{Name: "nginx-1", NodeName: "node-1"},
					Status:      types.PodStatusRunning,
					ContainerID: "abc123",
				},
			},
			expectStopped: []string{},
			expectRemoved: []string{},
		},
		{
			name:         "orphan container removed",
			existingPods: []types.Pod{},
			existingContainers: []runtime.ContainerInfo{
				{ID: "orphan123", Name: "nginx-orphan"},
			},
			expectedPodState: map[string]types.Pod{},
			expectStopped:    []string{"orphan123"},
			expectRemoved:    []string{"orphan123"},
		},
		{
			name: "mixed - link one, remove orphan",
			existingPods: []types.Pod{
				{Spec: types.PodSpec{Name: "nginx-1", NodeName: "node-1"}, Status: types.PodStatusPending},
			},
			existingContainers: []runtime.ContainerInfo{
				{ID: "abc123", Name: "nginx-1"},
				{ID: "orphan456", Name: "nginx-orphan"},
			},
			expectedPodState: map[string]types.Pod{
				"nginx-1": {
					Spec:        types.PodSpec{Name: "nginx-1", NodeName: "node-1"},
					Status:      types.PodStatusRunning,
					ContainerID: "abc123",
				},
			},
			expectStopped: []string{"orphan456"},
			expectRemoved: []string{"orphan456"},
		},
		{
			name: "no containers",
			existingPods: []types.Pod{
				{Spec: types.PodSpec{Name: "nginx-1", NodeName: "node-1"}, Status: types.PodStatusPending},
			},
			existingContainers: []runtime.ContainerInfo{},
			expectedPodState: map[string]types.Pod{
				"nginx-1": {
					Spec:   types.PodSpec{Name: "nginx-1", NodeName: "node-1"},
					Status: types.PodStatusPending,
				},
			},
			expectStopped: []string{},
			expectRemoved: []string{},
		},
		{
			name: "update stale container ID",
			existingPods: []types.Pod{
				{Spec: types.PodSpec{Name: "nginx-1", NodeName: "node-1"}, Status: types.PodStatusRunning, ContainerID: "old-id"},
			},
			existingContainers: []runtime.ContainerInfo{
				{ID: "new-id", Name: "nginx-1"},
			},
			expectedPodState: map[string]types.Pod{
				"nginx-1": {
					Spec:        types.PodSpec{Name: "nginx-1", NodeName: "node-1"},
					Status:      types.PodStatusRunning,
					ContainerID: "new-id",
				},
			},
			expectStopped: []string{},
			expectRemoved: []string{},
		},
		{
			name: "skip pods assigned to other nodes",
			existingPods: []types.Pod{
				{Spec: types.PodSpec{Name: "nginx-1", NodeName: "node-2"}, Status: types.PodStatusPending},
			},
			existingContainers: []runtime.ContainerInfo{
				{ID: "abc123", Name: "nginx-1"},
			},
			expectedPodState: map[string]types.Pod{
				"nginx-1": {
					Spec:   types.PodSpec{Name: "nginx-1", NodeName: "node-2"},
					Status: types.PodStatusPending, // unchanged - not our pod
				},
			},
			expectStopped: []string{},
			expectRemoved: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stoppedContainers []string
			var removedContainers []string

			mockRT := &mockRuntime{
				listFunc: func() ([]runtime.ContainerInfo, error) {
					return tt.existingContainers, nil
				},
				stopFunc: func(id string) error {
					stoppedContainers = append(stoppedContainers, id)
					return nil
				},
				removeFunc: func(id string) error {
					removedContainers = append(removedContainers, id)
					return nil
				},
			}

			podStore := store.NewMemStore[types.Pod]()
			for _, pod := range tt.existingPods {
				podStore.Put(pod.Spec.Name, pod)
			}

			kubelet := New(podStore, mockRT, "node-1")
			kubelet.Sync()

			for name, expectedPod := range tt.expectedPodState {
				actualPod, found := podStore.Get(name)
				if !found {
					t.Errorf("expected pod %s to exist", name)
					continue
				}
				if actualPod.Status != expectedPod.Status {
					t.Errorf("pod %s: expected status %s, got %s", name, expectedPod.Status, actualPod.Status)
				}
				if actualPod.ContainerID != expectedPod.ContainerID {
					t.Errorf("pod %s: expected containerID %s, got %s", name, expectedPod.ContainerID, actualPod.ContainerID)
				}
			}

			if len(stoppedContainers) != len(tt.expectStopped) {
				t.Errorf("expected %d stopped containers, got %d", len(tt.expectStopped), len(stoppedContainers))
			}
			for _, expectedID := range tt.expectStopped {
				found := slices.Contains(stoppedContainers, expectedID)
				if !found {
					t.Errorf("expected container %s to be stopped", expectedID)
				}
			}

			if len(removedContainers) != len(tt.expectRemoved) {
				t.Errorf("expected %d removed containers, got %d", len(tt.expectRemoved), len(removedContainers))
			}
			for _, expectedID := range tt.expectRemoved {
				found := slices.Contains(removedContainers, expectedID)
				if !found {
					t.Errorf("expected container %s to be removed", expectedID)
				}
			}
		})
	}
}
