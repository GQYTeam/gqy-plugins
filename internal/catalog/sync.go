package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/GQYTeam/gqy-plugins/internal/registry"
)

const (
	DefaultOrg             = "GQYTeam"
	DefaultPluginPrefix    = "gqy-plugins-"
	DefaultAPISource       = "https://api.github.com"
	DefaultRegistryPath    = "registry.json"
	defaultSyncHTTPTimeout = 45 * time.Second
)

type SyncOptions struct {
	Org          string
	Prefix       string
	APIBase      string
	Token        string
	RegistryPath string
	Now          time.Time
	Exclude      map[string]bool
	Client       *http.Client
}

type SkippedRepo struct {
	Repo   string `json:"repo"`
	Reason string `json:"reason"`
}

type SyncResult struct {
	Changed   bool          `json:"changed"`
	Included  []string      `json:"included"`
	Skipped   []SkippedRepo `json:"skipped"`
	Conflicts []string      `json:"conflicts,omitempty"`
}

// SyncOrg 拉取组织内所有公开的 gqy-plugins-* 仓库，校验其源码包并写入目录。
// 组织内插件直接进入目录；同版本产物不覆盖，只在结果里报告冲突。
func SyncOrg(ctx context.Context, opts SyncOptions) (*SyncResult, error) {
	opts.Org = strings.TrimSpace(opts.Org)
	if opts.Org == "" {
		opts.Org = DefaultOrg
	}
	if strings.TrimSpace(opts.Prefix) == "" {
		opts.Prefix = DefaultPluginPrefix
	}
	if strings.TrimSpace(opts.APIBase) == "" {
		opts.APIBase = DefaultAPISource
	}
	if strings.TrimSpace(opts.RegistryPath) == "" {
		opts.RegistryPath = DefaultRegistryPath
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	doc, err := loadOrNew(ctx, opts.RegistryPath)
	if err != nil {
		return nil, err
	}
	client := NewGitHubClient(opts.APIBase, opts.Token)
	if opts.Client != nil {
		client.HTTP = opts.Client
	}
	client.HTTP.Timeout = defaultSyncHTTPTimeout
	repos, err := client.ListOrgRepos(ctx, opts.Org)
	if err != nil {
		return nil, err
	}
	result := &SyncResult{Included: []string{}, Skipped: []SkippedRepo{}}
	for _, repo := range repos {
		if repo.Private || repo.Archived || repo.Fork {
			continue
		}
		if strings.HasPrefix(repo.Name, opts.Prefix) == false {
			continue
		}
		if opts.Exclude[repo.Name] {
			result.Skipped = append(result.Skipped, SkippedRepo{Repo: repo.FullName, Reason: "excluded by configuration"})
			continue
		}
		owner, name, ok := splitFullName(repo.FullName)
		if ok == false {
			result.Skipped = append(result.Skipped, SkippedRepo{Repo: repo.FullName, Reason: "invalid full_name"})
			continue
		}
		branch := strings.TrimSpace(repo.DefaultBranch)
		if branch == "" {
			branch = "main"
		}
		commit, err := client.HeadCommit(ctx, owner, name, branch)
		if err != nil {
			result.Skipped = append(result.Skipped, SkippedRepo{Repo: repo.FullName, Reason: err.Error()})
			continue
		}
		artifactURL := client.TarballURL(owner, name, commit.SHA)
		info, err := Inspect(ctx, artifactURL, InspectOptions{Token: opts.Token, Client: client.HTTP})
		if err != nil {
			result.Skipped = append(result.Skipped, SkippedRepo{Repo: repo.FullName, Reason: err.Error()})
			continue
		}
		tags := normalizeTags(repo.Topics)
		if len(tags) == 0 {
			tags = []string{"组织内"}
		}
		up, err := Upsert(doc, info, "org", repo.HTMLURL, tags, opts.Now, true)
		if err != nil {
			result.Skipped = append(result.Skipped, SkippedRepo{Repo: repo.FullName, Reason: err.Error()})
			continue
		}
		result.Included = append(result.Included, up.ID)
		if up.Conflict {
			result.Conflicts = append(result.Conflicts, fmt.Sprintf("%s@%s: %s", up.ID, up.Version, up.ConflictReason))
		}
		if up.Changed {
			result.Changed = true
		}
	}
	if result.Changed {
		sort.SliceStable(doc.Plugins, func(i, j int) bool { return doc.Plugins[i].ID < doc.Plugins[j].ID })
		if err := Write(opts.RegistryPath, doc); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func loadOrNew(ctx context.Context, path string) (*registry.Document, error) {
	if _, err := os.Stat(path); err == nil {
		return registry.Load(ctx, path)
	} else if errors.Is(err, os.ErrNotExist) == false {
		return nil, err
	}
	return &registry.Document{SchemaVersion: registry.SchemaVersion, UpdatedAt: time.Now().UTC().Format(time.RFC3339), Plugins: []registry.Plugin{}}, nil
}

// Write 用 strict 口径校验目录后原子写入 JSON。
func Write(path string, doc *registry.Document) error {
	if err := doc.Validate(true); err != nil {
		return err
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".registry-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		_ = tmp.Close()
		if ok == false {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	ok = true
	return nil
}

func splitFullName(full string) (string, string, bool) {
	parts := strings.Split(full, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}
