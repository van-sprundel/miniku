package controller

import (
	"miniku/pkg/store"
	"miniku/pkg/types"
	"testing"
)

func TestReconcile(t *testing.T) {
	tests := []struct {
		name             string
		replicaSet       types.ReplicaSet
		existingPods     []types.Pod
		expectedPodCount int
	}{
		{
			name: "scale up from 0",
			replicaSet: types.ReplicaSet{
				Name:         "nginx-rs",
				DesiredCount: 3,
				Selector:     map[string]string{"app": "nginx"},
				Template:     types.PodSpec{Image: "nginx:latest"},
			},
			existingPods:     []types.Pod{},
			expectedPodCount: 3,
		},
		{
			name: "scale up from existing",
			replicaSet: types.ReplicaSet{
				Name:         "nginx-rs",
				DesiredCount: 3,
				Selector:     map[string]string{"app": "nginx"},
				Template:     types.PodSpec{Image: "nginx:latest"},
			},
			existingPods: []types.Pod{
				{Spec: types.PodSpec{Name: "nginx-rs-abc", Labels: map[string]string{"app": "nginx"}}},
			},
			expectedPodCount: 3,
		},
		{
			name: "already converged",
			replicaSet: types.ReplicaSet{
				Name:         "nginx-rs",
				DesiredCount: 3,
				Selector:     map[string]string{"app": "nginx"},
				Template:     types.PodSpec{Image: "nginx:latest"},
			},
			existingPods: []types.Pod{
				{Spec: types.PodSpec{Name: "nginx-rs-1", Labels: map[string]string{"app": "nginx"}}},
				{Spec: types.PodSpec{Name: "nginx-rs-2", Labels: map[string]string{"app": "nginx"}}},
				{Spec: types.PodSpec{Name: "nginx-rs-3", Labels: map[string]string{"app": "nginx"}}},
			},
			expectedPodCount: 3,
		},
		{
			name: "scale down",
			replicaSet: types.ReplicaSet{
				Name:         "nginx-rs",
				DesiredCount: 2,
				Selector:     map[string]string{"app": "nginx"},
				Template:     types.PodSpec{Image: "nginx:latest"},
			},
			existingPods: []types.Pod{
				{Spec: types.PodSpec{Name: "nginx-rs-1", Labels: map[string]string{"app": "nginx"}}},
				{Spec: types.PodSpec{Name: "nginx-rs-2", Labels: map[string]string{"app": "nginx"}}},
				{Spec: types.PodSpec{Name: "nginx-rs-3", Labels: map[string]string{"app": "nginx"}}},
				{Spec: types.PodSpec{Name: "nginx-rs-4", Labels: map[string]string{"app": "nginx"}}},
			},
			expectedPodCount: 2,
		},
		{
			name: "scale to zero",
			replicaSet: types.ReplicaSet{
				Name:         "nginx-rs",
				DesiredCount: 0,
				Selector:     map[string]string{"app": "nginx"},
				Template:     types.PodSpec{Image: "nginx:latest"},
			},
			existingPods: []types.Pod{
				{Spec: types.PodSpec{Name: "nginx-rs-1", Labels: map[string]string{"app": "nginx"}}},
				{Spec: types.PodSpec{Name: "nginx-rs-2", Labels: map[string]string{"app": "nginx"}}},
				{Spec: types.PodSpec{Name: "nginx-rs-3", Labels: map[string]string{"app": "nginx"}}},
			},
			expectedPodCount: 0,
		},
		{
			name: "only count matching pods",
			replicaSet: types.ReplicaSet{
				Name:         "nginx-rs",
				DesiredCount: 2,
				Selector:     map[string]string{"app": "nginx"},
				Template:     types.PodSpec{Image: "nginx:latest"},
			},
			existingPods: []types.Pod{
				{Spec: types.PodSpec{Name: "nginx-rs-1", Labels: map[string]string{"app": "nginx"}}},
				{Spec: types.PodSpec{Name: "redis-1", Labels: map[string]string{"app": "redis"}}},
				{Spec: types.PodSpec{Name: "postgres-1", Labels: map[string]string{"app": "postgres"}}},
			},
			expectedPodCount: 2, // 1 existing nginx + 1 created
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			podStore := store.NewMemStore[types.Pod]()
			rsStore := store.NewMemStore[types.ReplicaSet]()

			for _, pod := range tt.existingPods {
				podStore.Put(pod.Spec.Name, pod)
			}

			rsStore.Put(tt.replicaSet.Name, tt.replicaSet)

			controller := New(podStore, rsStore)
			_ = controller.reconcile(tt.replicaSet)

			// count matching pods after recon.
			matchingPods := controller.getMatchingPods(tt.replicaSet)
			if len(matchingPods) != tt.expectedPodCount {
				t.Errorf("expected %d matching pods, got %d", tt.expectedPodCount, len(matchingPods))
			}

			// verify currentcount was updated
			updatedRS, _ := rsStore.Get(tt.replicaSet.Name)
			if int(updatedRS.CurrentCount) != tt.expectedPodCount {
				t.Errorf("expected CurrentCount %d, got %d", tt.expectedPodCount, updatedRS.CurrentCount)
			}
		})
	}
}

func TestMatchesSelector(t *testing.T) {
	tests := []struct {
		name     string
		pod      types.Pod
		selector map[string]string
		expected bool
	}{
		{
			name:     "exact match",
			pod:      types.Pod{Spec: types.PodSpec{Labels: map[string]string{"app": "nginx"}}},
			selector: map[string]string{"app": "nginx"},
			expected: true,
		},
		{
			name:     "pod has extra labels",
			pod:      types.Pod{Spec: types.PodSpec{Labels: map[string]string{"app": "nginx", "env": "prod"}}},
			selector: map[string]string{"app": "nginx"},
			expected: true,
		},
		{
			name:     "value mismatch",
			pod:      types.Pod{Spec: types.PodSpec{Labels: map[string]string{"app": "redis"}}},
			selector: map[string]string{"app": "nginx"},
			expected: false,
		},
		{
			name:     "missing label",
			pod:      types.Pod{Spec: types.PodSpec{Labels: map[string]string{"env": "prod"}}},
			selector: map[string]string{"app": "nginx"},
			expected: false,
		},
		{
			name:     "empty pod labels",
			pod:      types.Pod{Spec: types.PodSpec{Labels: map[string]string{}}},
			selector: map[string]string{"app": "nginx"},
			expected: false,
		},
		{
			name:     "nil pod labels",
			pod:      types.Pod{Spec: types.PodSpec{Labels: nil}},
			selector: map[string]string{"app": "nginx"},
			expected: false,
		},
		{
			name:     "multi-label selector match",
			pod:      types.Pod{Spec: types.PodSpec{Labels: map[string]string{"app": "nginx", "env": "prod"}}},
			selector: map[string]string{"app": "nginx", "env": "prod"},
			expected: true,
		},
		{
			name:     "multi-label selector partial match",
			pod:      types.Pod{Spec: types.PodSpec{Labels: map[string]string{"app": "nginx"}}},
			selector: map[string]string{"app": "nginx", "env": "prod"},
			expected: false,
		},
		{
			name:     "empty selector matches all",
			pod:      types.Pod{Spec: types.PodSpec{Labels: map[string]string{"app": "nginx"}}},
			selector: map[string]string{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesSelector(tt.pod, tt.selector)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
