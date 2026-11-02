package archive

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	maxFiles       = 10000
	maxExtractSize = int64(512 << 20)
)

// ExtractTarGz 安全解包到 dest。拒绝绝对路径、..、符号链接与设备文件。
func ExtractTarGz(r io.Reader, dest string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("archive: gzip 打开失败: %w", err)
	}
	defer func() { _ = gz.Close() }()
	tr := tar.NewReader(gz)
	var files int
	var total int64
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("archive: 解包失败: %w", err)
		}
		files++
		if files > maxFiles {
			return fmt.Errorf("archive: 文件数超过上限 %d", maxFiles)
		}
		rel, err := safeName(hdr.Name)
		if err != nil {
			return err
		}
		if rel == "" {
			continue
		}
		target := filepath.Join(dest, filepath.FromSlash(rel))
		if inside(dest, target) == false {
			return fmt.Errorf("archive: 路径越界: %q", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if hdr.Size < 0 {
				return fmt.Errorf("archive: 非法文件大小: %s", hdr.Name)
			}
			total += hdr.Size
			if total > maxExtractSize {
				return fmt.Errorf("archive: 解包后体积超过上限 %d", maxExtractSize)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(0o644)
			if hdr.Mode&0o111 != 0 {
				mode = 0o755
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
			if err != nil {
				return err
			}
			if _, err := io.CopyN(f, tr, hdr.Size); err != nil {
				_ = f.Close()
				return fmt.Errorf("archive: 写入 %s 失败: %w", hdr.Name, err)
			}
			if err := f.Close(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("archive: 不允许的条目类型: %s", hdr.Name)
		}
	}
	return nil
}

func safeName(raw string) (string, error) {
	name := strings.ReplaceAll(raw, "\\", "/")
	clean := path.Clean(name)
	if clean == "." {
		return "", nil
	}
	if strings.HasPrefix(clean, "/") || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("archive: 非法路径: %q", raw)
	}
	return clean, nil
}

func inside(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && strings.HasPrefix(rel, ".."+string(filepath.Separator)) == false)
}
