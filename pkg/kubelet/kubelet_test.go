package kubelet

import (
	"errors"
	"miniku/pkg/store"
	"miniku/pkg/types"
	"testing"
	"time"
)

type mockRuntime struct {
	runFunc       func(types.PodSpec) (string, error)
	stopFunc      func(string) error
	removeFunc    func(string) error
	getStatusFunc func(string) (*types.ContainerState, error)
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
			kubelet := New(memStore, mockRT)

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
			kubelet := New(memStore, mockRT)

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
