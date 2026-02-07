package runtime

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const alpineURL = "https://dl-cdn.alpinelinux.org/alpine/v3.21/releases/x86_64/alpine-minirootfs-3.21.3-x86_64.tar.gz"

type imageManager struct {
	imagesDir string
}

func newImageManager(rootDir string) *imageManager {
	return &imageManager{imagesDir: filepath.Join(rootDir, "images")}
}

// ensureImage returns the path to a ready rootfs for the given image.
// Currently all images map to Alpine minirootfs.
func (m *imageManager) ensureImage(image string) (string, error) {
	rootfs := filepath.Join(m.imagesDir, "alpine")

	// check if already extracted
	if _, err := os.Stat(filepath.Join(rootfs, "bin", "sh")); err == nil {
		return rootfs, nil
	}

	if image != "alpine" {
		log.Printf("runtime: image %q not recognized, using alpine", image)
	}

	log.Printf("runtime: downloading alpine rootfs...")

	if err := os.MkdirAll(rootfs, 0755); err != nil {
		return "", fmt.Errorf("create image dir: %w", err)
	}

	resp, err := http.Get(alpineURL)
	if err != nil {
		return "", fmt.Errorf("download image: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download image: HTTP %d", resp.StatusCode)
	}

	if err := extractTarGz(resp.Body, rootfs); err != nil {
		_ = os.RemoveAll(rootfs)
		return "", fmt.Errorf("extract image: %w", err)
	}

	log.Printf("runtime: alpine rootfs ready at %s", rootfs)
	return rootfs, nil
}

func extractTarGz(r io.Reader, dst string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		if err := extractTarEntry(hdr, tr, dst); err != nil {
			return err
		}
	}
}

func extractTarEntry(hdr *tar.Header, tr io.Reader, dst string) error {
	target := filepath.Join(dst, hdr.Name)

	switch hdr.Typeflag {
	case tar.TypeDir:
		return os.MkdirAll(target, os.FileMode(hdr.Mode))
	case tar.TypeReg:
		return extractRegularFile(target, tr, os.FileMode(hdr.Mode))
	case tar.TypeSymlink:
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return os.Symlink(hdr.Linkname, target)
	case tar.TypeLink:
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return os.Link(filepath.Join(dst, hdr.Linkname), target)
	}

	return nil
}

func extractRegularFile(target string, r io.Reader, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}

	if _, err := io.Copy(f, r); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
