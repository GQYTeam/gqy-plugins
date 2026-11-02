// Package catalog 实现组织插件自动收录、外部产物校验与目录写入。
package catalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GQYTeam/gqy-plugins/internal/archive"
	"github.com/GQYTeam/gqy-plugins/internal/manifest"
)

const maxArtifactSize = int64(256 << 20)

type InspectOptions struct {
	Token  string
	Client *http.Client
}

type PackageInfo struct {
	Manifest    *manifest.Manifest
	ArtifactURL string
	SHA256      string
	Size        int64
}

// Inspect 下载并解析一个插件包，计算真实 SHA-256、大小并校验包内 manifest。
// 支持 https、http、file:// 与本地路径；生产外部入口只允许 HTTPS GitHub Release。
func Inspect(ctx context.Context, artifactURL string, opts InspectOptions) (*PackageInfo, error) {
	client := opts.Client
	if client == nil {
		client = &http.Client{Timeout: 180 * time.Second}
	}
	tmp, err := os.MkdirTemp("", "gqy-catalog-inspect-*")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	artifactPath := filepath.Join(tmp, "plugin.tar.gz")
	size, sum, err := download(ctx, client, artifactURL, artifactPath, opts.Token)
	if err != nil {
		return nil, err
	}
	target := filepath.Join(tmp, "unpacked")
	if err := os.MkdirAll(target, 0o755); err != nil {
		return nil, err
	}
	f, err := os.Open(artifactPath)
	if err != nil {
		return nil, err
	}
	err = archive.ExtractTarGz(f, target)
	_ = f.Close()
	if err != nil {
		return nil, err
	}
	root, err := archive.LocatePluginRoot(target)
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
	return &PackageInfo{Manifest: m, ArtifactURL: artifactURL, SHA256: sum, Size: size}, nil
}

func download(ctx context.Context, client *http.Client, rawURL, dest, token string) (int64, string, error) {
	rc, err := open(ctx, client, rawURL, token)
	if err != nil {
		return 0, "", err
	}
	defer func() { _ = rc.Close() }()
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return 0, "", err
	}
	h := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(f, h), io.LimitReader(rc, maxArtifactSize+1))
	closeErr := f.Close()
	if copyErr != nil {
		return 0, "", fmt.Errorf("catalog: 下载失败: %w", copyErr)
	}
	if closeErr != nil {
		return 0, "", closeErr
	}
	if n > maxArtifactSize {
		return 0, "", fmt.Errorf("catalog: 产物超过 %d 字节上限", maxArtifactSize)
	}
	return n, hex.EncodeToString(h.Sum(nil)), nil
}

func open(ctx context.Context, client *http.Client, rawURL, token string) (io.ReadCloser, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("catalog: URL 非法: %w", err)
	}
	switch u.Scheme {
	case "http", "https":
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "gqy-plugins-catalog")
		req.Header.Set("Accept", "application/vnd.github+json, application/octet-stream")
		if token != "" && strings.EqualFold(u.Host, "api.github.com") {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("catalog: 下载失败: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("catalog: 下载返回 HTTP %d", resp.StatusCode)
		}
		return resp.Body, nil
	case "file":
		return os.Open(filepath.FromSlash(u.Path))
	case "":
		return os.Open(rawURL)
	default:
		return nil, fmt.Errorf("catalog: 不支持的产物协议 %q", u.Scheme)
	}
}
