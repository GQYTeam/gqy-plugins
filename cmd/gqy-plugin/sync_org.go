package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/GQYTeam/gqy-plugins/internal/catalog"
)

func runSyncOrg(args []string) error {
	fs := flag.NewFlagSet("sync-org", flag.ContinueOnError)
	org := fs.String("org", catalog.DefaultOrg, "GitHub 组织")
	prefix := fs.String("prefix", catalog.DefaultPluginPrefix, "插件仓库名前缀")
	apiBase := fs.String("api-base", catalog.DefaultAPISource, "GitHub API 地址")
	token := fs.String("token", os.Getenv("GITHUB_TOKEN"), "GitHub API token（默认读 GITHUB_TOKEN）")
	registryPath := fs.String("registry", catalog.DefaultRegistryPath, "目录文件路径")
	asJSON := fs.Bool("json", false, "输出 JSON")
	exclude := fs.String("exclude", "", "逗号分隔的排除仓库名")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *asJSON && strings.TrimSpace(*registryPath) == "" {
		return fmt.Errorf("--registry 不能为空")
	}
	excludeSet := make(map[string]bool)
	for _, name := range strings.Split(*exclude, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			excludeSet[name] = true
		}
	}
	res, err := catalog.SyncOrg(context.Background(), catalog.SyncOptions{
		Org:          *org,
		Prefix:       *prefix,
		APIBase:      *apiBase,
		Token:        *token,
		RegistryPath: *registryPath,
		Exclude:      excludeSet,
	})
	if err != nil {
		return err
	}
	if *asJSON {
		return writeJSON(res)
	}
	fmt.Printf("组织插件同步完成：changed=%v，收录 %d 个仓库\n", res.Changed, len(res.Included))
	for _, skip := range res.Skipped {
		fmt.Fprintf(os.Stderr, "跳过 %s: %s\n", skip.Repo, skip.Reason)
	}
	for _, conflict := range res.Conflicts {
		fmt.Fprintln(os.Stderr, "版本冲突:", conflict)
	}
	return nil
}
