package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type GitHubRepo struct {
	Name          string    `json:"name"`
	FullName      string    `json:"full_name"`
	HTMLURL       string    `json:"html_url"`
	DefaultBranch string    `json:"default_branch"`
	Private       bool      `json:"private"`
	Archived      bool      `json:"archived"`
	Fork          bool      `json:"fork"`
	Topics        []string  `json:"topics"`
	PushedAt      time.Time `json:"pushed_at"`
}

type GitHubCommit struct {
	SHA  string    `json:"sha"`
	Date time.Time `json:"-"`
}

type githubCommitResponse struct {
	SHA    string `json:"sha"`
	Commit struct {
		Committer struct {
			Date time.Time `json:"date"`
		} `json:"committer"`
		Author struct {
			Date time.Time `json:"date"`
		} `json:"author"`
	} `json:"commit"`
}

type GitHubClient struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func NewGitHubClient(baseURL, token string) *GitHubClient {
	return &GitHubClient{
		BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		Token:   strings.TrimSpace(token),
		HTTP:    &http.Client{Timeout: 45 * time.Second},
	}
}

func (c *GitHubClient) ListOrgRepos(ctx context.Context, org string) ([]GitHubRepo, error) {
	org = strings.TrimSpace(org)
	if org == "" {
		return nil, fmt.Errorf("catalog: 组织名为空")
	}
	var all []GitHubRepo
	for page := 1; page <= 50; page++ {
		var repos []GitHubRepo
		endpoint := fmt.Sprintf("%s/orgs/%s/repos?type=public&per_page=100&page=%d", c.BaseURL, url.PathEscape(org), page)
		if err := c.getJSON(ctx, endpoint, &repos); err != nil {
			return nil, err
		}
		all = append(all, repos...)
		if len(repos) < 100 {
			break
		}
	}
	return all, nil
}

func (c *GitHubClient) HeadCommit(ctx context.Context, owner, repo, ref string) (*GitHubCommit, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/%s/commits/%s", c.BaseURL, url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(ref))
	var raw githubCommitResponse
	if err := c.getJSON(ctx, endpoint, &raw); err != nil {
		return nil, err
	}
	date := raw.Commit.Committer.Date
	if date.IsZero() {
		date = raw.Commit.Author.Date
	}
	if raw.SHA == "" {
		return nil, fmt.Errorf("catalog: %s/%s 提交响应缺少 sha", owner, repo)
	}
	return &GitHubCommit{SHA: raw.SHA, Date: date}, nil
}

func (c *GitHubClient) TarballURL(owner, repo, ref string) string {
	return fmt.Sprintf("%s/repos/%s/%s/tarball/%s", c.BaseURL, url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(ref))
}

func (c *GitHubClient) getJSON(ctx context.Context, endpoint string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "gqy-plugins-catalog")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 45 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("catalog: GitHub 请求失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("catalog: 读取 GitHub 响应失败: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("catalog: GitHub API %s 返回 HTTP %d: %s", endpoint, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("catalog: 解析 GitHub 响应失败: %w", err)
	}
	return nil
}
