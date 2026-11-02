// Package manifest 校验插件包内的 manifest.json（GQY 04 P3）。
package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	idRe      = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)
	versionRe = regexp.MustCompile(`^\d{1,4}\.\d{1,4}\.\d{1,4}$`)
	serviceRe = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,63}$`)
	actionRe  = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{0,63}$`)
)

type Action struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Schema      map[string]any `json:"schema"`
}

type Manifest struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Protocol    string            `json:"protocol"`
	Community   string            `json:"community,omitempty"`
	Command     []string          `json:"command"`
	Cwd         string            `json:"cwd"`
	Env         map[string]string `json:"env,omitempty"`
	TimeoutMS   int               `json:"timeout_ms"`
	Concurrency string            `json:"concurrency"`
	Actions     []Action          `json:"actions"`
	Address     string            `json:"address,omitempty"`
	Provides    []string          `json:"provides"`
	Requires    []string          `json:"requires"`
	Mode        string            `json:"mode,omitempty"`
	Hash        string            `json:"hash,omitempty"`
}

// Parse 解析并校验 manifest；未知字段忽略，任何非法字段整体拒绝。
func Parse(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("manifest: JSON 解析失败: %w", err)
	}
	m.ID = strings.TrimSpace(m.ID)
	m.Name = strings.TrimSpace(m.Name)
	m.Version = strings.TrimSpace(m.Version)
	m.Description = strings.TrimSpace(m.Description)
	m.Protocol = strings.TrimSpace(m.Protocol)
	m.Mode = strings.TrimSpace(m.Mode)

	if idRe.MatchString(m.ID) == false {
		return nil, fmt.Errorf("manifest: id 非法: %q", m.ID)
	}
	if n := len([]rune(m.Name)); n < 1 || n > 64 {
		return nil, errors.New("manifest: name 必须为 1–64 字符")
	}
	if versionRe.MatchString(m.Version) == false {
		return nil, fmt.Errorf("manifest: version 非法: %q", m.Version)
	}
	if len([]rune(m.Description)) > 200 {
		return nil, errors.New("manifest: description 不能超过 200 字符")
	}
	if m.Mode != "" && m.Mode != "resident" && m.Mode != "oneshot" {
		return nil, fmt.Errorf("manifest: mode 只能是 resident / oneshot: %q", m.Mode)
	}
	if m.Protocol != "" {
		switch m.Protocol {
		case "gqy-jsonrpc-v1", "mcp", "unix", "tcp":
		default:
			return nil, fmt.Errorf("manifest: 未知 protocol: %q", m.Protocol)
		}
	}
	switch m.Protocol {
	case "gqy-jsonrpc-v1", "mcp":
		if len(m.Command) == 0 {
			return nil, fmt.Errorf("manifest: %s 需要 command", m.Protocol)
		}
	case "unix", "tcp":
		if strings.TrimSpace(m.Address) == "" {
			return nil, fmt.Errorf("manifest: %s 需要 address", m.Protocol)
		}
	}
	for _, part := range m.Command {
		if strings.TrimSpace(part) == "" {
			return nil, errors.New("manifest: command 每项必须非空")
		}
	}
	if m.Cwd != "" {
		cwd := filepath.Clean(m.Cwd)
		if filepath.IsAbs(cwd) || cwd == ".." || strings.HasPrefix(cwd, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("manifest: cwd 必须位于插件目录内: %q", m.Cwd)
		}
	}
	if m.TimeoutMS == 0 {
		m.TimeoutMS = 10000
	}
	if m.TimeoutMS < 1 || m.TimeoutMS > 120000 {
		return nil, errors.New("manifest: timeout_ms 必须在 1–120000 之间")
	}
	switch m.Concurrency {
	case "":
		m.Concurrency = "concurrent"
	case "concurrent", "single-channel", "exclusive":
	default:
		return nil, fmt.Errorf("manifest: concurrency 非法: %q", m.Concurrency)
	}
	seenActions := make(map[string]bool, len(m.Actions))
	for i := range m.Actions {
		a := &m.Actions[i]
		a.Name = strings.TrimSpace(a.Name)
		a.Description = strings.TrimSpace(a.Description)
		if actionRe.MatchString(a.Name) == false {
			return nil, fmt.Errorf("manifest: action name 非法: %q", a.Name)
		}
		if seenActions[a.Name] {
			return nil, fmt.Errorf("manifest: action 重复: %s", a.Name)
		}
		seenActions[a.Name] = true
		if n := len([]rune(a.Description)); n < 1 || n > 200 {
			return nil, fmt.Errorf("manifest: action %s description 必须为 1–200 字符", a.Name)
		}
		if a.Schema == nil || a.Schema["type"] != "object" {
			return nil, fmt.Errorf("manifest: action %s schema 根 type 必须为 object", a.Name)
		}
	}
	provides, err := cleanServices("provides", m.Provides)
	if err != nil {
		return nil, err
	}
	requires, err := cleanServices("requires", m.Requires)
	if err != nil {
		return nil, err
	}
	m.Provides = provides
	m.Requires = requires
	return &m, nil
}

func cleanServices(field string, in []string) ([]string, error) {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, raw := range in {
		s := strings.TrimSpace(raw)
		if serviceRe.MatchString(s) == false {
			return nil, fmt.Errorf("manifest: %s 项非法: %q", field, raw)
		}
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out, nil
}
