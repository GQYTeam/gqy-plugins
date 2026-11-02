package registry

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Find 查找插件。id 必须精确匹配。
func (d *Document) Find(id string) (*Plugin, error) {
	for i := range d.Plugins {
		if d.Plugins[i].ID == id {
			return &d.Plugins[i], nil
		}
	}
	return nil, fmt.Errorf("registry: 未收录插件 %q", id)
}

// ResolveVersion 选择版本；version 为空时选最新的非 yanked 版本。
func (p *Plugin) ResolveVersion(version string) (*Version, error) {
	if version != "" {
		for i := range p.Versions {
			if p.Versions[i].Version != version {
				continue
			}
			if p.Versions[i].Yanked {
				return nil, fmt.Errorf("registry: %s@%s 已被撤回", p.ID, version)
			}
			return &p.Versions[i], nil
		}
		return nil, fmt.Errorf("registry: 插件 %s 没有版本 %s", p.ID, version)
	}
	candidates := make([]*Version, 0, len(p.Versions))
	for i := range p.Versions {
		if p.Versions[i].Yanked == false {
			candidates = append(candidates, &p.Versions[i])
		}
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("registry: 插件 %s 没有可安装版本", p.ID)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return CompareVersions(candidates[i].Version, candidates[j].Version) > 0
	})
	return candidates[0], nil
}

// CompareVersions 比较两个三位数字版本；非法版本按字典序兜底。
func CompareVersions(a, b string) int {
	pa, okA := parseVersion(a)
	pb, okB := parseVersion(b)
	if okA == false || okB == false {
		return strings.Compare(a, b)
	}
	for i := 0; i < 3; i++ {
		if pa[i] < pb[i] {
			return -1
		}
		if pa[i] > pb[i] {
			return 1
		}
	}
	return 0
}

func parseVersion(s string) ([3]int, bool) {
	var out [3]int
	if versionRe.MatchString(s) == false {
		return out, false
	}
	for i, part := range strings.Split(s, ".") {
		n, err := strconv.Atoi(part)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}
