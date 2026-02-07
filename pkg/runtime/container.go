package runtime

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type containerProcess struct {
	ID       string
	Name     string
	PID      int
	RootFS   string
	Cmd      *exec.Cmd
	ExitCode int
	Exited   bool
}

type containerMeta struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Image     string    `json:"image"`
	PID       int       `json:"pid"`
	CreatedAt time.Time `json:"created_at"`
}

func saveMeta(dir string, meta containerMeta) error {
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "meta.json"), data, 0644)
}

func loadMeta(dir string) (containerMeta, error) {
	data, err := os.ReadFile(filepath.Join(dir, "meta.json"))
	if err != nil {
		return containerMeta{}, err
	}
	var meta containerMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return containerMeta{}, err
	}
	return meta, nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		// handle symlinks (Walk uses Lstat, so ModeSymlink is set)
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		}

		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode fs.FileMode) error {
	sf, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open src %s: %w", src, err)
	}
	defer func() { _ = sf.Close() }()

	df, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("open dst %s: %w", dst, err)
	}

	if _, err = io.Copy(df, sf); err != nil {
		_ = df.Close()
		return err
	}
	return df.Close()
}
