// Package install 实现在线目录条目的下载、校验与原子安装。
package install

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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
	"github.com/GQYTeam/gqy-plugins/internal/registry"
)

const maxArtifactSize = int64(256 << 20)

type Options struct {
	PluginsDir string
	Force      bool
	DryRun     bool
}

type Result struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	Dir         string `json:"dir"`
	ArtifactURL string `json:"artifact_url"`
	SHA256      string `json:"sha256"`
	Size        int64  `json:"size"`
	Replaced    bool   `json:"replaced"`
	DryRun      bool   `json:"dry_run"`
}

// Install 按 registry 条目安装一个常驻插件。安装是原子的：
// 先下载、校验 sha256、解包到插件目录下的临时目录，最后 rename 到 <plugins>/<id>。
func Install(ctx context.Context, doc *registry.Document, id, version string, opts Options) (*Result, error) {
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("install: 插件 id 为空")
	}
	plugin, err := doc.Find(id)
	if err != nil {
		return nil, err
	}
	v, err := plugin.ResolveVersion(version)
	if err != nil {
		return nil, err
	}
	if v.Mode == "oneshot" {
		return nil, fmt.Errorf("install: %s@%s 是一次性插件，不能安装为常驻插件", id, v.Version)
	}
	if strings.TrimSpace(opts.PluginsDir) == "" {
		return nil, errors.New("install: 插件目录为空")
	}
	pluginsDir, err := filepath.Abs(opts.PluginsDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		return nil, err
	}
	lock := filepath.Join(pluginsDir, ".install-"+id+".lock")
	if err := os.Mkdir(lock, 0o700); err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("install: 另一个安装进程正在处理 %s", id)
		}
		return nil, err
	}
	defer func() { _ = os.Remove(lock) }()

	stage, err := os.MkdirTemp(pluginsDir, ".gqy-plugin-stage-"+id+"-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(stage) }()

	artifactURL, err := doc.ResolveURL(v.URL)
	if err != nil {
		return nil, err
	}
	artifactPath := filepath.Join(stage, "plugin.tar.gz")
	size, sum, err := fetch(ctx, artifactURL, artifactPath)
	if err != nil {
		return nil, err
	}
	if sum != strings.ToLower(v.SHA256) {
		return nil, fmt.Errorf("install: sha256 不匹配：期望 %s，实际 %s", v.SHA256, sum)
	}
	if v.Size > 0 && size != v.Size {
		return nil, fmt.Errorf("install: 大小不匹配：期望 %d，实际 %d", v.Size, size)
	}
	extractDir := filepath.Join(stage, "unpacked")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return nil, err
	}
	af, err := os.Open(artifactPath)
	if err != nil {
		return nil, err
	}
	err = archive.ExtractTarGz(af, extractDir)
	_ = af.Close()
	if err != nil {
		return nil, err
	}
	pluginRoot, err := archive.LocatePluginRoot(extractDir)
	if err != nil {
		return nil, err
	}
	manifestData, err := os.ReadFile(filepath.Join(pluginRoot, "manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("install: 读取包内 manifest.json 失败: %w", err)
	}
	m, err := manifest.Parse(manifestData)
	if err != nil {
		return nil, err
	}
	if m.ID != id {
		return nil, fmt.Errorf("install: 包内 id %q 与目录条目 %q 不一致", m.ID, id)
	}
	if m.Version != v.Version {
		return nil, fmt.Errorf("install: 包内版本 %q 与目录条目 %q 不一致", m.Version, v.Version)
	}
	if m.Mode == "oneshot" {
		return nil, fmt.Errorf("install: 包内 manifest 声明为 oneshot，不能常驻安装")
	}
	if v.Protocol != "" && m.Protocol != "" && v.Protocol != m.Protocol {
		return nil, fmt.Errorf("install: 包内 protocol %q 与目录条目 %q 不一致", m.Protocol, v.Protocol)
	}
	target := filepath.Join(pluginsDir, id)
	existed := false
	if _, statErr := os.Lstat(target); statErr == nil {
		existed = true
		if opts.Force == false {
			return nil, fmt.Errorf("install: %s 已存在；要覆盖请加 --force", target)
		}
	}
	res := &Result{
		ID:          id,
		Version:     v.Version,
		Dir:         target,
		ArtifactURL: artifactURL,
		SHA256:      sum,
		Size:        size,
		Replaced:    existed,
		DryRun:      opts.DryRun,
	}
	if opts.DryRun {
		return res, nil
	}
	var backup string
	if existed {
		backup = filepath.Join(pluginsDir, ".gqy-plugin-backup-"+id+"-"+fmt.Sprint(time.Now().UnixNano()))
		if err := os.Rename(target, backup); err != nil {
			return nil, fmt.Errorf("install: 备份旧版本失败: %w", err)
		}
	}
	if err := os.Rename(pluginRoot, target); err != nil {
		if backup != "" {
			_ = os.Rename(backup, target)
		}
		return nil, fmt.Errorf("install: 安装到 %s 失败: %w", target, err)
	}
	if backup != "" {
		_ = os.RemoveAll(backup)
	}
	return res, nil
}

func fetch(ctx context.Context, rawURL, dest string) (int64, string, error) {
	rc, err := openArtifact(ctx, rawURL)
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
		return 0, "", fmt.Errorf("install: 下载失败: %w", copyErr)
	}
	if closeErr != nil {
		return 0, "", closeErr
	}
	if n > maxArtifactSize {
		return 0, "", fmt.Errorf("install: 产物超过 %d 字节上限", maxArtifactSize)
	}
	return n, hex.EncodeToString(h.Sum(nil)), nil
}

func openArtifact(ctx context.Context, rawURL string) (io.ReadCloser, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("install: 产物 URL 非法: %w", err)
	}
	switch u.Scheme {
	case "http", "https":
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, err
		}
		resp, err := (&http.Client{Timeout: 180 * time.Second}).Do(req)
		if err != nil {
			return nil, fmt.Errorf("install: 下载失败: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("install: 下载返回 HTTP %d", resp.StatusCode)
		}
		return resp.Body, nil
	case "file":
		return os.Open(filepath.FromSlash(u.Path))
	case "":
		return os.Open(rawURL)
	default:
		return nil, fmt.Errorf("install: 不支持的产物协议 %q", u.Scheme)
	}
}
