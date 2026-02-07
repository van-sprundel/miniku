package runtime

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"miniku/pkg/types"
)

type NamespaceRuntime struct {
	mu         sync.Mutex
	containers map[string]*containerProcess
	rootDir    string
	images     *imageManager
}

func NewNamespaceRuntime(rootDir string) (*NamespaceRuntime, error) {
	containersDir := filepath.Join(rootDir, "containers")
	if err := os.MkdirAll(containersDir, 0755); err != nil {
		return nil, fmt.Errorf("create containers dir: %w", err)
	}

	return &NamespaceRuntime{
		containers: make(map[string]*containerProcess),
		rootDir:    rootDir,
		images:     newImageManager(rootDir),
	}, nil
}

func (nr *NamespaceRuntime) Run(pod types.PodSpec) (string, error) {
	imgRootFS, err := nr.images.ensureImage(pod.Image)
	if err != nil {
		return "", fmt.Errorf("ensure image: %w", err)
	}

	id, err := randomID()
	if err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}

	containerDir := filepath.Join(nr.rootDir, "containers", id)
	rootfs := filepath.Join(containerDir, "rootfs")

	if err := os.MkdirAll(containerDir, 0755); err != nil {
		return "", fmt.Errorf("create container dir: %w", err)
	}

	log.Printf("runtime: copying rootfs for container %s...", id)
	if err := copyDir(imgRootFS, rootfs); err != nil {
		_ = os.RemoveAll(containerDir)
		return "", fmt.Errorf("copy rootfs: %w", err)
	}

	cp, err := nr.startChild(pod, id, containerDir, rootfs)
	if err != nil {
		_ = os.RemoveAll(containerDir)
		return "", err
	}

	nr.mu.Lock()
	nr.containers[id] = cp
	nr.mu.Unlock()

	log.Printf("runtime: started container %s (pid %d) for pod %s", id, cp.PID, pod.Name)
	return id, nil
}

func (nr *NamespaceRuntime) startChild(pod types.PodSpec, id, containerDir, rootfs string) (*containerProcess, error) {
	command := pod.Command
	if len(command) == 0 {
		command = []string{"/bin/sh"}
	}

	hostname := pod.Name
	if hostname == "" {
		hostname = id
	}

	configJSON, err := json.Marshal(childConfig{
		RootFS:   rootfs,
		Hostname: hostname,
		Command:  command,
		Env:      pod.Env,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}

	cmd := exec.Command("/proc/self/exe")
	cmd.Env = []string{"MINIKU_CHILD=1"}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWUTS,
	}
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start child: %w", err)
	}

	if _, err := stdin.Write(configJSON); err != nil {
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("write config: %w", err)
	}
	_ = stdin.Close()

	cp := &containerProcess{
		ID:     id,
		Name:   pod.Name,
		PID:    cmd.Process.Pid,
		RootFS: rootfs,
		Cmd:    cmd,
	}

	if err := saveMeta(containerDir, containerMeta{
		ID:        id,
		Name:      pod.Name,
		Image:     pod.Image,
		PID:       cmd.Process.Pid,
		CreatedAt: time.Now(),
	}); err != nil {
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("save meta: %w", err)
	}

	// background wait for process exit
	go func() {
		waitErr := cmd.Wait()
		nr.mu.Lock()
		defer nr.mu.Unlock()
		cp.Exited = true
		if waitErr != nil {
			if exitErr, ok := waitErr.(*exec.ExitError); ok {
				cp.ExitCode = exitErr.ExitCode()
			} else {
				cp.ExitCode = -1
			}
		}
	}()

	return cp, nil
}

func (nr *NamespaceRuntime) Stop(containerID string) error {
	nr.mu.Lock()
	cp, ok := nr.containers[containerID]
	nr.mu.Unlock()

	if !ok {
		return fmt.Errorf("container %s not found", containerID)
	}

	if cp.Exited {
		return nil
	}

	// SIGTERM first
	if err := syscall.Kill(cp.PID, syscall.SIGTERM); err != nil {
		if err == syscall.ESRCH {
			return nil
		}
		return fmt.Errorf("sigterm: %w", err)
	}

	// wait up to 10s for graceful shutdown
	deadline := time.After(10 * time.Second)
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-deadline:
			log.Printf("runtime: force killing container %s (pid %d)", containerID, cp.PID)
			_ = syscall.Kill(cp.PID, syscall.SIGKILL)
			time.Sleep(200 * time.Millisecond)
			return nil
		case <-tick.C:
			nr.mu.Lock()
			exited := cp.Exited
			nr.mu.Unlock()
			if exited {
				return nil
			}
		}
	}
}

func (nr *NamespaceRuntime) Remove(containerID string) error {
	nr.mu.Lock()
	cp, ok := nr.containers[containerID]
	if ok && !cp.Exited {
		nr.mu.Unlock()
		return fmt.Errorf("container %s is still running", containerID)
	}
	delete(nr.containers, containerID)
	nr.mu.Unlock()

	containerDir := filepath.Join(nr.rootDir, "containers", containerID)
	if err := os.RemoveAll(containerDir); err != nil {
		return fmt.Errorf("remove container dir: %w", err)
	}
	return nil
}

func (nr *NamespaceRuntime) GetStatus(containerID string) (*types.ContainerState, error) {
	nr.mu.Lock()
	cp, ok := nr.containers[containerID]
	nr.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("container %s not found", containerID)
	}

	if cp.Exited {
		return &types.ContainerState{
			Status:   types.ContainerStatusExited,
			ExitCode: cp.ExitCode,
		}, nil
	}

	// probe if process is still alive
	if err := syscall.Kill(cp.PID, 0); err != nil {
		nr.mu.Lock()
		cp.Exited = true
		nr.mu.Unlock()
		return &types.ContainerState{
			Status:   types.ContainerStatusExited,
			ExitCode: -1,
		}, nil
	}

	return &types.ContainerState{
		Status: types.ContainerStatusRunning,
	}, nil
}

func (nr *NamespaceRuntime) List() ([]ContainerInfo, error) {
	nr.mu.Lock()
	populated := len(nr.containers) > 0
	nr.mu.Unlock()

	if !populated {
		if err := nr.recoverFromDisk(); err != nil {
			log.Printf("runtime: recovery from disk: %v", err)
		}
	}

	nr.mu.Lock()
	defer nr.mu.Unlock()

	var result []ContainerInfo
	for _, cp := range nr.containers {
		result = append(result, ContainerInfo{
			ID:   cp.ID,
			Name: cp.Name,
		})
	}
	return result, nil
}

func (nr *NamespaceRuntime) recoverFromDisk() error {
	containersDir := filepath.Join(nr.rootDir, "containers")
	entries, err := os.ReadDir(containersDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	nr.mu.Lock()
	defer nr.mu.Unlock()

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		id := entry.Name()
		if _, exists := nr.containers[id]; exists {
			continue
		}

		meta, err := loadMeta(filepath.Join(containersDir, id))
		if err != nil {
			log.Printf("runtime: skip container %s: %v", id, err)
			continue
		}

		alive := syscall.Kill(meta.PID, 0) == nil

		nr.containers[id] = &containerProcess{
			ID:     meta.ID,
			Name:   meta.Name,
			PID:    meta.PID,
			RootFS: filepath.Join(containersDir, id, "rootfs"),
			Exited: !alive,
		}
	}

	return nil
}

func randomID() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
