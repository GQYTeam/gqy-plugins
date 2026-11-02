package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/GQYTeam/gqy-plugins/internal/catalog"
	"github.com/GQYTeam/gqy-plugins/internal/registry"
)

type externalAddOutput struct {
	Changed     bool   `json:"changed"`
	ID          string `json:"id"`
	Version     string `json:"version"`
	Repository  string `json:"repository"`
	ArtifactURL string `json:"artifact_url"`
	SHA256      string `json:"sha256"`
	Size        int64  `json:"size"`
	Registry    string `json:"registry"`
}

func runAddExternal(args []string) error {
	fs := flag.NewFlagSet("add-external", flag.ContinueOnError)
	registryPath := fs.String("registry", catalog.DefaultRegistryPath, "目录文件路径")
	repository := fs.String("repository", "", "外部插件来源仓库 URL")
	artifactURL := fs.String("artifact-url", "", "外部插件 GitHub Release 产物 URL")
	issueBody := fs.String("issue-body", "", "插件收录 issue body 文件")
	expectedSHA := fs.String("expected-sha256", "", "issue 里提交的 sha256")
	expectedSize := fs.Int64("expected-size", 0, "issue 里提交的产物字节数")
	tags := fs.String("tags", "", "逗号分隔标签")
	asJSON := fs.Bool("json", false, "输出 JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	fields := map[string]string{}
	if strings.TrimSpace(*issueBody) != "" {
		data, err := os.ReadFile(*issueBody)
		if err != nil {
			return err
		}
		fields = parseIssueForm(string(data))
		if *repository == "" {
			*repository = fieldValue(fields, "来源仓库")
		}
		if *artifactURL == "" {
			*artifactURL = fieldValue(fields, "产物 url")
		}
		if *expectedSHA == "" {
			*expectedSHA = fieldValue(fields, "sha-256")
		}
		if *expectedSize == 0 {
			size, err := catalog.ParseSize(fieldValue(fields, "产物字节数"))
			if err != nil {
				return err
			}
			*expectedSize = size
		}
	}
	if strings.TrimSpace(*repository) == "" {
		return errors.New("add-external 需要 --repository 或 issue 里的「来源仓库」")
	}
	if strings.TrimSpace(*artifactURL) == "" {
		return errors.New("add-external 需要 --artifact-url 或 issue 里的「产物 URL」")
	}
	doc, err := loadRegistryOrNew(*registryPath)
	if err != nil {
		return err
	}
	now := time.Now()
	res, err := catalog.AddExternal(context.Background(), doc, catalog.ExternalSpec{
		Repository:     *repository,
		ArtifactURL:    *artifactURL,
		ExpectedSHA256: *expectedSHA,
		ExpectedSize:   *expectedSize,
		Tags:           splitCSV(*tags),
	}, now)
	if err != nil {
		return err
	}
	if issueID := fieldValue(fields, "插件 id"); issueID != "" && issueID != res.Artifact.Manifest.ID {
		return fmt.Errorf("issue 插件 id %q 与包内 manifest id %q 不一致", issueID, res.Artifact.Manifest.ID)
	}
	if issueVersion := fieldValue(fields, "版本"); issueVersion != "" && issueVersion != res.Artifact.Manifest.Version {
		return fmt.Errorf("issue 版本 %q 与包内 manifest version %q 不一致", issueVersion, res.Artifact.Manifest.Version)
	}
	if res.Upsert.Changed {
		if err := catalog.Write(*registryPath, doc); err != nil {
			return err
		}
	}
	out := externalAddOutput{
		Changed:     res.Upsert.Changed,
		ID:          res.Upsert.ID,
		Version:     res.Upsert.Version,
		Repository:  res.Repository,
		ArtifactURL: *artifactURL,
		SHA256:      res.Artifact.SHA256,
		Size:        res.Artifact.Size,
		Registry:    *registryPath,
	}
	if *asJSON {
		return writeJSON(out)
	}
	if out.Changed {
		fmt.Printf("已生成外部插件目录条目 %s@%s，请提交 PR。\n", out.ID, out.Version)
	} else {
		fmt.Printf("目录已是 %s@%s 且产物一致，无需变更。\n", out.ID, out.Version)
	}
	fmt.Printf("repository: %s\nartifact: %s\nsha256: %s\nsize: %d\n", out.Repository, out.ArtifactURL, out.SHA256, out.Size)
	return nil
}

func loadRegistryOrNew(path string) (*registry.Document, error) {
	doc, err := registry.Load(context.Background(), path)
	if err == nil {
		return doc, nil
	}
	if errors.Is(err, os.ErrNotExist) == false {
		return nil, err
	}
	return &registry.Document{SchemaVersion: registry.SchemaVersion, UpdatedAt: time.Now().UTC().Format(time.RFC3339), Plugins: []registry.Plugin{}}, nil
}

func parseIssueForm(body string) map[string]string {
	out := make(map[string]string)
	key := ""
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		trimmed = strings.Trim(trimmed, "`")
		trimmed = strings.TrimSpace(trimmed)
		if strings.HasPrefix(trimmed, "### ") {
			key = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed, "### ")))
			continue
		}
		if key == "" || trimmed == "" || strings.HasPrefix(trimmed, "<!--") || strings.HasPrefix(trimmed, "- [") {
			continue
		}
		if out[key] != "" {
			out[key] += "\n"
		}
		out[key] += trimmed
	}
	return out
}

func fieldValue(fields map[string]string, needle string) string {
	needle = strings.ToLower(needle)
	for key, value := range fields {
		if strings.Contains(key, needle) {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func splitCSV(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
