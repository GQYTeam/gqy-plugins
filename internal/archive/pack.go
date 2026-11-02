// Package archive 负责插件包的确定性打包与安全解包。
package archive

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/GQYTeam/gqy-plugins/internal/manifest"
)

type Summary struct {
	Path    string `json:"path"`
	ID      string `json:"id"`
	Version string `json:"version"`
	Size    int64  `json:"size"`
	SHA256  string `json:"sha256"`
}

// Pack 把插件目录打包成确定性 tar.gz：manifest.json 在包根，文件按路径排序，
// 时间戳与 uid/gid 固定，权限归一为 0644 / 0755。输出先写临时文件再原子替换。
func Pack(root, output string) (*Summary, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(rootAbs, "manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("archive: 读取 manifest.json 失败: %w", err)
	}
	m, err := manifest.Parse(data)
	if err != nil {
		return nil, err
	}
	entries, err := collect(rootAbs)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return nil, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(output), ".gqy-plugin-pack-*")
	if err != nil {
		return nil, err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		_ = tmp.Close()
		if ok == false {
			_ = os.Remove(tmpName)
		}
	}()

	hasher := sha256.New()
	mw := io.MultiWriter(tmp, hasher)
	gz := gzip.NewWriter(mw)
	gz.Header.ModTime = time.Unix(0, 0).UTC()
	gz.Header.OS = 255
	tw := tar.NewWriter(gz)

	for _, entry := range entries {
		hdr, open, err := headerFor(entry)
		if err != nil {
			return nil, err
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, fmt.Errorf("archive: 写 tar header 失败: %w", err)
		}
		if open == nil {
			continue
		}
		if _, err := io.Copy(tw, open); err != nil {
			_ = open.Close()
			return nil, fmt.Errorf("archive: 写入 %s 失败: %w", entry.rel, err)
		}
		if err := open.Close(); err != nil {
			return nil, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	if err := tmp.Close(); err != nil {
		return nil, err
	}
	if err := os.Rename(tmpName, output); err != nil {
		return nil, err
	}
	ok = true
	info, err := os.Stat(output)
	if err != nil {
		return nil, err
	}
	return &Summary{
		Path:    output,
		ID:      m.ID,
		Version: m.Version,
		Size:    info.Size(),
		SHA256:  hex.EncodeToString(hasher.Sum(nil)),
	}, nil
}
