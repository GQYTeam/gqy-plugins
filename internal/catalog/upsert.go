package catalog

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/GQYTeam/gqy-plugins/internal/manifest"
	"github.com/GQYTeam/gqy-plugins/internal/registry"
)

type UpsertResult struct {
	ID             string `json:"id"`
	Version        string `json:"version"`
	Changed        bool   `json:"changed"`
	Conflict       bool   `json:"conflict"`
	ConflictReason string `json:"conflict_reason,omitempty"`
}

// VersionConflictError 表示同 id@version 已登记了不同产物。
type VersionConflictError struct {
	ID      string
	Version string
	Reason  string
}

func (e *VersionConflictError) Error() string {
	return fmt.Sprintf("catalog: %s@%s 已登记不同产物：%s；请发布新版本", e.ID, e.Version, e.Reason)
}

// Upsert 把已校验的插件包写入目录。
// sourceType 只能是 org / external；existingConflict=true 时保留已有产物并只报告冲突。
func Upsert(doc *registry.Document, info *PackageInfo, sourceType, repository string, tags []string, now time.Time, existingConflict bool) (*UpsertResult, error) {
	if doc == nil || info == nil || info.Manifest == nil {
		return nil, fmt.Errorf("catalog: 目录或产物信息为空")
	}
	if sourceType != "org" && sourceType != "external" {
		return nil, fmt.Errorf("catalog: source_type 只能是 org / external")
	}
	m := info.Manifest
	result := &UpsertResult{ID: m.ID, Version: m.Version}
	var plugin *registry.Plugin
	for i := range doc.Plugins {
		if doc.Plugins[i].ID == m.ID {
			plugin = &doc.Plugins[i]
			break
		}
	}
	if plugin == nil {
		doc.Plugins = append(doc.Plugins, registry.Plugin{ID: m.ID})
		plugin = &doc.Plugins[len(doc.Plugins)-1]
		result.Changed = true
	}
	if plugin.SourceType != "" && plugin.SourceType != sourceType {
		return nil, fmt.Errorf("catalog: 插件 %s 来源类型冲突：目录为 %s，本次为 %s", m.ID, plugin.SourceType, sourceType)
	}
	name := strings.TrimSpace(m.Name)
	summary := strings.TrimSpace(m.Description)
	if summary == "" {
		summary = name
	}
	summary = truncate(summary, 160)
	description := truncate(strings.TrimSpace(m.Description), 2000)
	cleanTags := normalizeTags(tags)
	if plugin.Name != name {
		plugin.Name = name
		result.Changed = true
	}
	if plugin.Summary != summary {
		plugin.Summary = summary
		result.Changed = true
	}
	if description != "" && plugin.Description != description {
		plugin.Description = description
		result.Changed = true
	}
	if ker := sourceType; plugin.SourceType != ker {
		plugin.SourceType = ker
		result.Changed = true
	}
	if repository != "" && plugin.Repository != repository {
		plugin.Repository = repository
		result.Changed = true
	}
	if repository != "" && plugin.Homepage != repository {
		plugin.Homepage = repository
		result.Changed = true
	}
	if len(cleanTags) > 0 && len(plugin.Tags) == 0 {
		plugin.Tags = cleanTags
		result.Changed = true
	}

	mode := strings.TrimSpace(m.Mode)
	if mode == "" {
		mode = "resident"
	}
	want := registry.Version{
		Version:     m.Version,
		Channel:     "stable",
		PublishedAt: now.UTC().Format(time.RFC3339),
		Mode:        mode,
		Protocol:    m.Protocol,
		Actions:     actionNames(m),
		Provides:    append([]string(nil), m.Provides...),
		Requires:    append([]string(nil), m.Requires...),
		URL:         info.ArtifactURL,
		SHA256:      info.SHA256,
		Size:        info.Size,
	}
	index := -1
	for i := range plugin.Versions {
		if plugin.Versions[i].Version == m.Version {
			index = i
		}
	}
	if index >= 0 {
		old := &plugin.Versions[index]
		if old.URL != want.URL || old.SHA256 != want.SHA256 || (old.Size > 0 && old.Size != want.Size) {
			reason := fmt.Sprintf("已有 url=%s sha256=%s size=%d，新 url=%s sha256=%s size=%d", old.URL, old.SHA256, old.Size, want.URL, want.SHA256, want.Size)
			result.Conflict = true
			result.ConflictReason = reason
			if existingConflict == false {
				return nil, &VersionConflictError{ID: m.ID, Version: m.Version, Reason: reason}
			}
		} else {
			if old.Mode != want.Mode || old.Protocol != want.Protocol || reflect.DeepEqual(old.Actions, want.Actions) == false ||
				reflect.DeepEqual(old.Provides, want.Provides) == false || reflect.DeepEqual(old.Requires, want.Requires) == false {
				old.Mode = want.Mode
				old.Protocol = want.Protocol
				old.Actions = want.Actions
				old.Provides = want.Provides
				old.Requires = want.Requires
				result.Changed = true
			}
		}
	} else {
		plugin.Versions = append(plugin.Versions, want)
		sort.SliceStable(plugin.Versions, func(i, j int) bool {
			return registry.CompareVersions(plugin.Versions[i].Version, plugin.Versions[j].Version) > 0
		})
		result.Changed = true
	}
	sort.SliceStable(doc.Plugins, func(i, j int) bool { return doc.Plugins[i].ID < doc.Plugins[j].ID })
	if result.Changed {
		doc.UpdatedAt = now.UTC().Format(time.RFC3339)
	}
	return result, nil
}

func actionNames(m *manifest.Manifest) []string {
	out := make([]string, 0, len(m.Actions))
	for _, action := range m.Actions {
		out = append(out, action.Name)
	}
	return out
}

func normalizeTags(tags []string) []string {
	seen := make(map[string]bool, len(tags))
	out := make([]string, 0, len(tags))
	for _, raw := range tags {
		tag := strings.TrimSpace(raw)
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		out = append(out, tag)
		if len(out) == 8 {
			break
		}
	}
	return out
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}
