// Package registry 读取并校验 GQY 插件在线目录 registry.json。
//
// registry.json 是目录的唯一事实来源：静态站点和 gqy-plugin CLI 都读它。
// 包只做元数据校验，不下载插件文件；下载与安装（internal/install、gqy-plugin CLI）
// 尚未实现，见 README 的路线图。
package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const SchemaVersion = 1

const DefaultURL = "https://raw.githubusercontent.com/GQYTeam/gqy-plugins/main/registry.json"
const maxBytes = 4 << 20

var (
	idRe      = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)
	versionRe = regexp.MustCompile(`^\d{1,4}\.\d{1,4}\.\d{1,4}$`)
	sha256Re  = regexp.MustCompile(`^[a-f0-9]{64}$`)
	actionRe  = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{0,63}$`)
	serviceRe = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,63}$`)
)

type Document struct {
	SchemaVersion int      `json:"schema_version"`
	UpdatedAt     string   `json:"updated_at"`
	Plugins       []Plugin `json:"plugins"`
	base          string
}

type Plugin struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Summary     string    `json:"summary"`
	Description string    `json:"description,omitempty"`
	Homepage    string    `json:"homepage,omitempty"`
	SourceType  string    `json:"source_type,omitempty"`
	Repository  string    `json:"repository,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	Versions    []Version `json:"versions"`
}

type Version struct {
	Version     string   `json:"version"`
	Channel     string   `json:"channel,omitempty"`
	PublishedAt string   `json:"published_at,omitempty"`
	Mode        string   `json:"mode,omitempty"`
	Protocol    string   `json:"protocol,omitempty"`
	Actions     []string `json:"actions,omitempty"`
	Provides    []string `json:"provides,omitempty"`
	Requires    []string `json:"requires,omitempty"`
	URL         string   `json:"url"`
	SHA256      string   `json:"sha256"`
	Size        int64    `json:"size,omitempty"`
	Yanked      bool     `json:"yanked,omitempty"`
}

// Load 从 https URL、file:// URL 或本地路径读取目录。
func Load(ctx context.Context, source string) (*Document, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil, errors.New("registry: 来源为空")
	}
	data, base, err := readSource(ctx, source)
	if err != nil {
		return nil, err
	}
	doc, err := Parse(data)
	if err != nil {
		return nil, err
	}
	doc.base = base
	return doc, nil
}

// ParseJSON 解析并校验目录 JSON，并记录用于解析相对 URL 的 base。
func ParseJSON(data []byte, base string) (*Document, error) {
	doc, err := Parse(data)
	if err != nil {
		return nil, err
	}
	doc.base = base
	return doc, nil
}

// Parse 解析并做基础校验。未知字段直接拒绝，避免发布元数据拼写错误。
func Parse(data []byte) (*Document, error) {
	var doc Document
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("registry: JSON 解析失败: %w", err)
	}
	if err := doc.Validate(false); err != nil {
		return nil, err
	}
	return &doc, nil
}

// Validate 校验目录。strict=true 时要求 artifact 使用 HTTPS 绝对地址。
func (d *Document) Validate(strict bool) error {
	if d.SchemaVersion != SchemaVersion {
		return fmt.Errorf("registry: schema_version 必须为 %d", SchemaVersion)
	}
	if strings.TrimSpace(d.UpdatedAt) == "" {
		return errors.New("registry: updated_at 不能为空")
	}
	if _, err := time.Parse(time.RFC3339, d.UpdatedAt); err != nil {
		return fmt.Errorf("registry: updated_at 必须是 RFC3339: %w", err)
	}
	if d.Plugins == nil {
		return errors.New("registry: plugins 必须是数组（空目录写 []）")
	}
	seen := make(map[string]bool, len(d.Plugins))
	for i := range d.Plugins {
		p := &d.Plugins[i]
		if err := validatePlugin(p, strict); err != nil {
			return fmt.Errorf("registry: plugins[%d] %q: %w", i, p.ID, err)
		}
		if seen[p.ID] {
			return fmt.Errorf("registry: 插件 id 重复: %s", p.ID)
		}
		seen[p.ID] = true
	}
	return nil
}

// ResolveURL 把目录里的相对 artifact 地址按目录来源解析成绝对地址。
// base 由 Load / ParseJSON 记下；没记下 base 时原样返回。
func (d *Document) ResolveURL(raw string) (string, error) {
	ref, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("registry: url 非法: %w", err)
	}
	if d == nil || d.base == "" || ref.IsAbs() {
		return ref.String(), nil
	}
	base, err := url.Parse(d.base)
	if err != nil {
		return "", fmt.Errorf("registry: base 非法: %w", err)
	}
	return base.ResolveReference(ref).String(), nil
}

// validatePlugin 校验单个插件条目。strict=true 时要求 artifact 用 HTTPS 绝对地址。
func validatePlugin(p *Plugin, strict bool) error {
	if !idRe.MatchString(p.ID) {
		return fmt.Errorf("registry: id 必须匹配 %s，得到 %q", idRe.String(), p.ID)
	}
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("registry: name 不能为空")
	}
	if strings.TrimSpace(p.Summary) == "" {
		return errors.New("registry: summary 不能为空")
	}
	for _, tag := range p.Tags {
		if strings.TrimSpace(tag) == "" {
			return errors.New("registry: tags 里不能有空字符串")
		}
	}
	if p.Homepage != "" {
		u, err := url.Parse(p.Homepage)
		if err != nil {
			return fmt.Errorf("registry: homepage 非法: %w", err)
		}
		if u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
			return fmt.Errorf("registry: homepage 必须是 http(s) 绝对地址，得到 %q", p.Homepage)
		}
	}
	if p.SourceType != "" && p.SourceType != "org" && p.SourceType != "external" {
		return fmt.Errorf("registry: source_type 只能是 org / external: %q", p.SourceType)
	}
	if p.Repository != "" {
		u, err := url.Parse(p.Repository)
		if err != nil {
			return fmt.Errorf("registry: repository 非法: %w", err)
		}
		if u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
			return fmt.Errorf("registry: repository 必须是 http(s) 绝对地址，得到 %q", p.Repository)
		}
		if strict && u.Scheme != "https" {
			return fmt.Errorf("registry: strict 要求 repository 使用 https")
		}
	}
	if p.SourceType != "" && p.Repository == "" {
		return errors.New("registry: 设置了 source_type 时必须提供 repository")
	}
	if len(p.Versions) == 0 {
		return errors.New("registry: versions 不能为空")
	}
	seen := make(map[string]bool, len(p.Versions))
	for i := range p.Versions {
		v := &p.Versions[i]
		if err := validateVersion(v, strict); err != nil {
			return fmt.Errorf("versions[%d] (v%s): %w", i, v.Version, err)
		}
		if seen[v.Version] {
			return fmt.Errorf("registry: 版本重复: %s", v.Version)
		}
		seen[v.Version] = true
	}
	return nil
}

// validateVersion 校验单个版本条目。
func validateVersion(v *Version, strict bool) error {
	if !versionRe.MatchString(v.Version) {
		return fmt.Errorf("registry: version 必须匹配 %s，得到 %q", versionRe.String(), v.Version)
	}
	if v.PublishedAt != "" {
		if _, err := time.Parse(time.RFC3339, v.PublishedAt); err != nil {
			return fmt.Errorf("registry: published_at 必须是 RFC3339: %w", err)
		}
	}
	if !sha256Re.MatchString(v.SHA256) {
		return fmt.Errorf("registry: sha256 必须是 64 位小写十六进制，得到 %q", v.SHA256)
	}
	if v.Size < 0 {
		return fmt.Errorf("registry: size 不能为负数: %d", v.Size)
	}
	if v.Mode != "" && v.Mode != "resident" && v.Mode != "oneshot" {
		return fmt.Errorf("registry: mode 只能是 resident / oneshot: %q", v.Mode)
	}
	if v.Protocol != "" {
		switch v.Protocol {
		case "gqy-jsonrpc-v1", "mcp", "unix", "tcp":
		default:
			return fmt.Errorf("registry: protocol 非法: %q", v.Protocol)
		}
	}
	for _, action := range v.Actions {
		if actionRe.MatchString(action) == false {
			return fmt.Errorf("registry: action 名非法: %q", action)
		}
	}
	for _, list := range []struct {
		name string
		vals []string
	}{
		{name: "provides", vals: v.Provides},
		{name: "requires", vals: v.Requires},
	} {
		for _, val := range list.vals {
			if serviceRe.MatchString(val) == false {
				return fmt.Errorf("registry: %s 项非法: %q", list.name, val)
			}
		}
	}
	if err := validateArtifactURL(v.URL, strict); err != nil {
		return err
	}
	for _, list := range []struct {
		name string
		vals []string
	}{
		{name: "actions", vals: v.Actions},
		{name: "provides", vals: v.Provides},
		{name: "requires", vals: v.Requires},
	} {
		for _, val := range list.vals {
			if strings.TrimSpace(val) == "" {
				return fmt.Errorf("registry: %s 里不能有空字符串", list.name)
			}
		}
	}
	return nil
}

// validateArtifactURL 校验 artifact 地址：默认允许相对地址与 http(s) 绝对地址，
// strict=true 时只接受 HTTPS 绝对地址（发布目录用这个口径）。
func validateArtifactURL(raw string, strict bool) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return errors.New("registry: url 不能为空")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("registry: url 非法: %w", err)
	}
	if !u.IsAbs() {
		if strict {
			return fmt.Errorf("registry: strict 模式要求 url 是 https 绝对地址，得到 %q", raw)
		}
		return nil
	}
	if u.Scheme != "https" && (strict || u.Scheme != "http") {
		return fmt.Errorf("registry: url 只接受 %s，得到 %q", schemeRule(strict), raw)
	}
	if u.Host == "" {
		return fmt.Errorf("registry: url 缺少 host: %q", raw)
	}
	return nil
}

func schemeRule(strict bool) string {
	if strict {
		return "https"
	}
	return "http(s)"
}

// readSource 读取目录内容，并返回用于解析相对 URL 的 base。
// 支持 https/http URL、file:// URL 与本地路径。
func readSource(ctx context.Context, source string) ([]byte, string, error) {
	lower := strings.ToLower(source)
	switch {
	case strings.HasPrefix(lower, "https://"), strings.HasPrefix(lower, "http://"):
		return readHTTP(ctx, source)
	case strings.HasPrefix(lower, "file://"):
		return readFileURL(source)
	default:
		return readFile(source)
	}
}

// readHTTP 通过 HTTP(S) 拉取目录；base 取 URL 所在目录。
func readHTTP(ctx context.Context, source string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return nil, "", fmt.Errorf("registry: 构造请求失败: %w", err)
	}
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("registry: 拉取 %s 失败: %w", source, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("registry: 拉取 %s 返回 %s", source, resp.Status)
	}
	data, err := readLimited(resp.Body)
	if err != nil {
		return nil, "", err
	}
	return data, dirURL(resp.Request.URL.String()), nil
}

// readFileURL 读取 file:// URL 指向的本地文件；base 为文件所在目录的 file URL。
func readFileURL(source string) ([]byte, string, error) {
	u, err := url.Parse(source)
	if err != nil {
		return nil, "", fmt.Errorf("registry: file:// 来源非法: %w", err)
	}
	if u.Host != "" && u.Host != "localhost" {
		return nil, "", fmt.Errorf("registry: 不支持远程 file:// 来源: %s", u.Host)
	}
	if u.Path == "" {
		return nil, "", errors.New("registry: file:// 来源缺少路径")
	}
	return readFile(filepath.FromSlash(u.Path))
}

// readFile 读取本地文件；base 为文件所在目录的 file URL。
func readFile(name string) ([]byte, string, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, "", fmt.Errorf("registry: 打开 %s 失败: %w", name, err)
	}
	defer func() {
		_ = f.Close()
	}()
	data, err := readLimited(f)
	if err != nil {
		return nil, "", fmt.Errorf("registry: 读取 %s 失败: %w", name, err)
	}
	abs, err := filepath.Abs(name)
	if err != nil {
		return nil, "", fmt.Errorf("registry: 解析 %s 的绝对路径失败: %w", name, err)
	}
	base := &url.URL{Scheme: "file", Path: filepath.ToSlash(filepath.Dir(abs)) + "/"}
	return data, base.String(), nil
}

// dirURL 把来源 URL 截到所在目录，作为相对地址的 base。
func dirURL(source string) string {
	u, err := url.Parse(source)
	if err != nil {
		return ""
	}
	u.RawQuery, u.Fragment = "", ""
	if i := strings.LastIndex(u.Path, "/"); i >= 0 {
		u.Path = u.Path[:i+1]
	}
	return u.String()
}

// readLimited 读取内容并守住体积上限，避免误读超大文件。
func readLimited(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("registry: 读取内容失败: %w", err)
	}
	if len(data) > maxBytes {
		return nil, fmt.Errorf("registry: 目录超过 %d 字节上限", maxBytes)
	}
	return data, nil
}
