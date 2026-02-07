package runtime

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"miniku/pkg/types"
)

func skipIfNotRoot(t *testing.T) {
	t.Helper()
	if os.Getuid() != 0 {
		t.Skip("requires root")
	}
}

func newTestRuntime(t *testing.T) *NamespaceRuntime {
	t.Helper()
	skipIfNotRoot(t)

	dir := t.TempDir()
	rt, err := NewNamespaceRuntime(dir)
	if err != nil {
		t.Fatalf("NewNamespaceRuntime: %v", err)
	}
	return rt
}

func TestEnsureImage(t *testing.T) {
	skipIfNotRoot(t)

	dir := t.TempDir()
	im := newImageManager(dir)

	rootfs, err := im.ensureImage("alpine")
	if err != nil {
		t.Fatalf("ensureImage: %v", err)
	}

	// verify /bin/sh exists
	if _, err := os.Stat(filepath.Join(rootfs, "bin", "sh")); err != nil {
		t.Fatalf("rootfs missing /bin/sh: %v", err)
	}

	// second call should be cached (no download)
	rootfs2, err := im.ensureImage("alpine")
	if err != nil {
		t.Fatalf("ensureImage (cached): %v", err)
	}
	if rootfs != rootfs2 {
		t.Fatalf("expected same path, got %s vs %s", rootfs, rootfs2)
	}
}

func TestRunAndGetStatus(t *testing.T) {
	rt := newTestRuntime(t)

	id, err := rt.Run(types.PodSpec{
		Name:    "test-run",
		Image:   "alpine",
		Command: []string{"/bin/sh", "-c", "sleep 60"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// give the process a moment to start
	time.Sleep(200 * time.Millisecond)

	status, err := rt.GetStatus(id)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if status.Status != types.ContainerStatusRunning {
		t.Fatalf("expected Running, got %s", status.Status)
	}

	// cleanup
	_ = rt.Stop(id)
	_ = rt.Remove(id)
}

func TestStopContainer(t *testing.T) {
	rt := newTestRuntime(t)

	id, err := rt.Run(types.PodSpec{
		Name:    "test-stop",
		Image:   "alpine",
		Command: []string{"/bin/sh", "-c", "sleep 60"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	if err := rt.Stop(id); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	status, err := rt.GetStatus(id)
	if err != nil {
		t.Fatalf("GetStatus after stop: %v", err)
	}
	if status.Status != types.ContainerStatusExited {
		t.Fatalf("expected Exited, got %s", status.Status)
	}

	// cleanup
	_ = rt.Remove(id)
}

func TestRemoveContainer(t *testing.T) {
	rt := newTestRuntime(t)

	id, err := rt.Run(types.PodSpec{
		Name:    "test-remove",
		Image:   "alpine",
		Command: []string{"/bin/sh", "-c", "sleep 60"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	time.Sleep(200 * time.Millisecond)
	_ = rt.Stop(id)

	if err := rt.Remove(id); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// container dir should be gone
	containerDir := filepath.Join(rt.rootDir, "containers", id)
	if _, err := os.Stat(containerDir); !os.IsNotExist(err) {
		t.Fatalf("expected container dir to be removed")
	}
}

func TestList(t *testing.T) {
	rt := newTestRuntime(t)

	id1, err := rt.Run(types.PodSpec{
		Name:    "test-list-1",
		Image:   "alpine",
		Command: []string{"/bin/sh", "-c", "sleep 60"},
	})
	if err != nil {
		t.Fatalf("Run 1: %v", err)
	}

	id2, err := rt.Run(types.PodSpec{
		Name:    "test-list-2",
		Image:   "alpine",
		Command: []string{"/bin/sh", "-c", "sleep 60"},
	})
	if err != nil {
		t.Fatalf("Run 2: %v", err)
	}

	list, err := rt.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(list))
	}

	// cleanup
	_ = rt.Stop(id1)
	_ = rt.Stop(id2)
	_ = rt.Remove(id1)
	_ = rt.Remove(id2)
}

func TestListRecovery(t *testing.T) {
	rt := newTestRuntime(t)
	rootDir := rt.rootDir

	id, err := rt.Run(types.PodSpec{
		Name:    "test-recovery",
		Image:   "alpine",
		Command: []string{"/bin/sh", "-c", "sleep 60"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// create a fresh runtime pointing at the same rootDir
	rt2, err := NewNamespaceRuntime(rootDir)
	if err != nil {
		t.Fatalf("NewNamespaceRuntime: %v", err)
	}

	list, err := rt2.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 recovered container, got %d", len(list))
	}
	if list[0].ID != id {
		t.Fatalf("expected id %s, got %s", id, list[0].ID)
	}

	// cleanup using original runtime (it has the cmd reference)
	_ = rt.Stop(id)
	_ = rt.Remove(id)
}
