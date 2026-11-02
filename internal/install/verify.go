package install

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GQYTeam/gqy-plugins/internal/archive"
	"github.com/GQYTeam/gqy-plugins/internal/manifest"
	"github.com/GQYTeam/gqy-plugins/internal/registry"
)

type VerifyResult struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
	Size    int64  `json:"size"`
	OK      bool   `json:"ok"`
}

// VerifyArtifact 下载并完整校验一个目录产物：哈希、大小、包内 manifest 身份。
func VerifyArtifact(ctx context.Context, doc *registry.Document, p *registry.Plugin, v *registry.Version) (*VerifyResult, error) {
	dir, err := os.MkdirTemp("", "gqy-plugin-verify-*")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	artifactURL, err := doc.ResolveURL(v.URL)
	if err != nil {
		return nil, err
	}
	artifactPath := filepath.Join(dir, "plugin.tar.gz")
	size, sum, err := fetch(ctx, artifactURL, artifactPath)
	if err != nil {
		return nil, err
	}
	if sum != strings.ToLower(v.SHA256) {
		return nil, fmt.Errorf("verify: %s@%s sha256 不匹配：期望 %s，实际 %s", p.ID, v.Version, v.SHA256, sum)
	}
	if v.Size > 0 && size != v.Size {
		return nil, fmt.Errorf("verify: %s@%s 大小不匹配：期望 %d，实际 %d", p.ID, v.Version, v.Size, size)
	}
	unpacked := filepath.Join(dir, "unpacked")
	if err := os.MkdirAll(unpacked, 0o755); err != nil {
		return nil, err
	}
	f, err := os.Open(artifactPath)
	if err != nil {
		return nil, err
	}
	err = archive.ExtractTarGz(f, unpacked)
	_ = f.Close()
	if err != nil {
		return nil, err
	}
	root, err := archive.LocatePluginRoot(unpacked)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		return nil, err
	}
	m, err := manifest.Parse(data)
	if err != nil {
		return nil, err
	}
	if m.ID != p.ID || m.Version != v.Version {
		return nil, fmt.Errorf("verify: 包内身份 %s@%s 与目录 %s@%s 不一致", m.ID, m.Version, p.ID, v.Version)
	}
	return &VerifyResult{ID: p.ID, Version: v.Version, URL: artifactURL, SHA256: sum, Size: size, OK: true}, nil
}
