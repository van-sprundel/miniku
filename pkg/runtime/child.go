package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
)

type childConfig struct {
	RootFS   string            `json:"rootfs"`
	Hostname string            `json:"hostname"`
	Command  []string          `json:"command"`
	Env      map[string]string `json:"env"`
}

func init() {
	if os.Getenv("MINIKU_CHILD") != "1" {
		return
	}
	runChild()
	os.Exit(0)
}

func runChild() {
	runtime.LockOSThread()

	var config childConfig
	if err := json.NewDecoder(os.Stdin).Decode(&config); err != nil {
		fmt.Fprintf(os.Stderr, "child: decode config: %v\n", err)
		os.Exit(1)
	}

	// set hostname in the new UTS namespace
	if err := syscall.Sethostname([]byte(config.Hostname)); err != nil {
		fmt.Fprintf(os.Stderr, "child: sethostname: %v\n", err)
		os.Exit(1)
	}

	// pivot_root into the container rootfs
	if err := pivotRoot(config.RootFS); err != nil {
		fmt.Fprintf(os.Stderr, "child: pivot_root: %v\n", err)
		os.Exit(1)
	}

	// mount /proc in the new PID namespace
	if err := os.MkdirAll("/proc", 0755); err != nil {
		fmt.Fprintf(os.Stderr, "child: mkdir /proc: %v\n", err)
		os.Exit(1)
	}
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		fmt.Fprintf(os.Stderr, "child: mount /proc: %v\n", err)
		os.Exit(1)
	}

	// build environment
	env := os.Environ()
	for k, v := range config.Env {
		env = append(env, k+"="+v)
	}

	// resolve and exec the command
	cmdPath, err := resolveCommand(config.Command[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "child: resolve command: %v\n", err)
		os.Exit(1)
	}

	if err := syscall.Exec(cmdPath, config.Command, env); err != nil {
		fmt.Fprintf(os.Stderr, "child: exec %s: %v\n", cmdPath, err)
		os.Exit(1)
	}
}

func pivotRoot(rootfs string) error {
	// bind mount rootfs onto itself (required for pivot_root)
	if err := syscall.Mount(rootfs, rootfs, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("bind mount: %w", err)
	}

	pivotOld := filepath.Join(rootfs, ".pivot_old")
	if err := os.MkdirAll(pivotOld, 0700); err != nil {
		return fmt.Errorf("mkdir pivot_old: %w", err)
	}

	if err := syscall.PivotRoot(rootfs, pivotOld); err != nil {
		return fmt.Errorf("pivot_root: %w", err)
	}

	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("chdir: %w", err)
	}

	// unmount and remove the old root
	if err := syscall.Unmount("/.pivot_old", syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount pivot_old: %w", err)
	}

	return os.RemoveAll("/.pivot_old")
}

func resolveCommand(cmd string) (string, error) {
	// absolute path — use directly
	if filepath.IsAbs(cmd) {
		if _, err := os.Stat(cmd); err == nil {
			return cmd, nil
		}
		return "", fmt.Errorf("command not found: %s", cmd)
	}

	// search common directories
	dirs := []string{"/bin", "/usr/bin", "/sbin", "/usr/sbin", "/usr/local/bin"}
	for _, dir := range dirs {
		path := filepath.Join(dir, cmd)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("command not found: %s", cmd)
}
