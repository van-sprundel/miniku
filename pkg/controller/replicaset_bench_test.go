package controller

import (
	"fmt"
	"miniku/pkg/testutil"
	"miniku/pkg/types"
	"testing"
)

func BenchmarkMatchesSelector(b *testing.B) {
	pod := types.Pod{
		Spec: types.PodSpec{
			Name:  "test-pod",
			Image: "nginx",
			Labels: map[string]string{
				"app":     "nginx",
				"env":     "prod",
				"version": "v1",
			},
		},
	}

	selector := map[string]string{
		"app": "nginx",
		"env": "prod",
	}

	for b.Loop() {
		matchesSelector(pod, selector)
	}
}

func BenchmarkGetMatchingPods(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("pods=%d", size), func(b *testing.B) {
			env := testutil.NewTestEnv()
			defer env.Close()

			// half match, half don't
			for i := range size {
				labels := map[string]string{"app": "other"}
				if i%2 == 0 {
					labels = map[string]string{"app": "nginx"}
				}
				env.PodStore.Put(fmt.Sprintf("pod-%d", i), types.Pod{
					Spec: types.PodSpec{
						Name:   fmt.Sprintf("pod-%d", i),
						Image:  "nginx",
						Labels: labels,
					},
				})
			}

			rs := types.ReplicaSet{
				Name:     "nginx-rs",
				Selector: map[string]string{"app": "nginx"},
			}

			ctrl := New(env.Client)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = ctrl.getMatchingPods(rs)
			}
		})
	}
}

func BenchmarkReconcile(b *testing.B) {
	env := testutil.NewTestEnv()
	defer env.Close()

	rs := types.ReplicaSet{
		Name:         "nginx-rs",
		DesiredCount: 3,
		Selector:     map[string]string{"app": "nginx"},
		Template: types.PodSpec{
			Image: "nginx:latest",
		},
	}
	env.RSStore.Put(rs.Name, rs)

	ctrl := New(env.Client)

	for b.Loop() {
		for _, pod := range env.PodStore.List() {
			env.PodStore.Delete(pod.Spec.Name)
		}
		_ = ctrl.reconcile(rs)
	}
}
