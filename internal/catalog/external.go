package catalog

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/GQYTeam/gqy-plugins/internal/registry"
)

var sha256Re = regexp.MustCompile(`^[a-f0-9]{64}$`)

type ExternalSpec struct {
	Repository     string
	ArtifactURL    string
	ExpectedSHA256 string
	ExpectedSize   int64
	Tags           []string
}

type ExternalResult struct {
	Upsert     *UpsertResult `json:"upsert"`
	Artifact   *PackageInfo  `json:"artifact"`
	Repository string        `json:"repository"`
}

// AddExternal 校验一个组织外 GitHub Release 产物并登记到目录。
// 组织内仓库必须走 SyncOrg，不能从这里绕过自动收录规则。
func AddExternal(ctx context.Context, doc *registry.Document, spec ExternalSpec, now time.Time) (*ExternalResult, error) {
	repoOwner, repoName, canonicalRepo, err := validateExternalRepository(spec.Repository)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(repoOwner, DefaultOrg) {
		return nil, fmt.Errorf("catalog: %s 属于组织 %s，应走组织内自动收录，不接受外部 PR", canonicalRepo, DefaultOrg)
	}
	if err := validateExternalArtifact(spec.ArtifactURL, repoOwner, repoName); err != nil {
		return nil, err
	}
	info, err := Inspect(ctx, spec.ArtifactURL, InspectOptions{})
	if err != nil {
		return nil, err
	}
	expectedSHA := normalizeSHA(spec.ExpectedSHA256)
	if expectedSHA != "" && expectedSHA != info.SHA256 {
		return nil, fmt.Errorf("catalog: 提交的 sha256 与产物不一致：提交 %s，实际 %s", expectedSHA, info.SHA256)
	}
	if spec.ExpectedSize > 0 && spec.ExpectedSize != info.Size {
		return nil, fmt.Errorf("catalog: 提交的 size 与产物不一致：提交 %d，实际 %d", spec.ExpectedSize, info.Size)
	}
	tags := spec.Tags
	if len(tags) == 0 {
		tags = []string{"外部收录"}
	}
	up, err := Upsert(doc, info, "external", canonicalRepo, tags, now, false)
	if err != nil {
		return nil, err
	}
	return &ExternalResult{Upsert: up, Artifact: info, Repository: canonicalRepo}, nil
}

func validateExternalRepository(raw string) (string, string, string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", "", "", fmt.Errorf("catalog: 来源仓库 URL 非法: %w", err)
	}
	if u.Scheme != "https" || strings.EqualFold(u.Host, "github.com") == false {
		return "", "", "", fmt.Errorf("catalog: 来源仓库必须是 https://github.com/owner/repo")
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", "", fmt.Errorf("catalog: 来源仓库必须是 https://github.com/owner/repo")
	}
	name := strings.TrimSuffix(parts[1], ".git")
	if name == "" {
		return "", "", "", fmt.Errorf("catalog: 来源仓库缺少仓库名")
	}
	canonical := "https://github.com/" + parts[0] + "/" + name
	return parts[0], name, canonical, nil
}

func validateExternalArtifact(raw, owner, repo string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("catalog: 产物 URL 非法: %w", err)
	}
	if u.Scheme != "https" || strings.EqualFold(u.Host, "github.com") == false {
		return fmt.Errorf("catalog: 外部产物必须是 https://github.com 的 Release 资产")
	}
	wantPrefix := "/" + owner + "/" + repo + "/releases/download/"
	if strings.HasPrefix(u.Path, wantPrefix) == false {
		return fmt.Errorf("catalog: 产物必须来自同一来源仓库的 Release: %s", wantPrefix)
	}
	lower := strings.ToLower(u.Path)
	if strings.HasSuffix(lower, ".tar.gz") == false && strings.HasSuffix(lower, ".tgz") == false {
		return fmt.Errorf("catalog: 外部产物必须是 .tar.gz / .tgz")
	}
	return nil
}

func normalizeSHA(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.TrimPrefix(s, "sha256:")
	if s == "" {
		return ""
	}
	if sha256Re.MatchString(s) == false {
		return ""
	}
	return s
}

// ParseSize 解析提交里的字节数；空字符串返回 0。
func ParseSize(raw string) (int64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("catalog: 产物字节数非法: %q", raw)
	}
	return n, nil
}
