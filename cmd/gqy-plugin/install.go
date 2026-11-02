package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GQYTeam/gqy-plugins/internal/gqyapi"
	"github.com/GQYTeam/gqy-plugins/internal/install"
	"github.com/GQYTeam/gqy-plugins/internal/registry"
)

type installOutput struct {
	install.Result
	ServerSynced bool     `json:"server_synced,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
}

func runInstall(args []string) error {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	source := fs.String("registry", defaultRegistry(), "registry.json URL 或本地路径")
	pluginsDirFlag := fs.String("plugins-dir", "", "插件安装目录，默认 $GQY_HOME/plugins")
	force := fs.Bool("force", false, "覆盖已有同 id 插件目录")
	dryRun := fs.Bool("dry-run", false, "只下载并校验，不落盘")
	server := fs.String("server", os.Getenv("GQY_URL"), "GQY 服务地址，如 http://127.0.0.1:5173")
	token := fs.String("token", os.Getenv("GQY_AUTH_TOKEN"), "GQY 管理接口令牌")
	loadAfter := fs.Bool("load", false, "安装成功后请求 GQY 加载插件")
	asJSON := fs.Bool("json", false, "输出 JSON")
	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return err
	}
	if len(pos) == 0 || strings.TrimSpace(pos[0]) == "" {
		return fmt.Errorf("install 需要 id 或 id@version")
	}
	if *loadAfter && strings.TrimSpace(*server) == "" {
		return fmt.Errorf("--load 需要同时提供 --server 或 GQY_URL")
	}
	id, ver := splitIDVersion(strings.TrimSpace(pos[0]))
	pluginsDir, err := resolvePluginsDir(*pluginsDirFlag)
	if err != nil {
		return err
	}
	doc, err := registry.Load(context.Background(), *source)
	if err != nil {
		return err
	}
	res, err := install.Install(context.Background(), doc, id, ver, install.Options{
		PluginsDir: pluginsDir,
		Force:      *force,
		DryRun:     *dryRun,
	})
	if err != nil {
		return err
	}
	out := installOutput{Result: *res}
	if *dryRun == false && strings.TrimSpace(*server) != "" {
		ctx := context.Background()
		if err := gqyapi.Reload(ctx, *server, *token); err != nil {
			out.Warnings = append(out.Warnings, "重新扫描失败: "+err.Error())
		} else {
			out.ServerSynced = true
			if *loadAfter {
				if err := gqyapi.Load(ctx, *server, *token, id); err != nil {
					out.Warnings = append(out.Warnings, "加载失败: "+err.Error())
				}
			}
		}
	}
	if *asJSON {
		return writeJSON(out)
	}
	printInstallResult(res, out, *loadAfter)
	return nil
}

func resolvePluginsDir(flagValue string) (string, error) {
	if strings.TrimSpace(flagValue) != "" {
		return filepath.Abs(flagValue)
	}
	home := strings.TrimSpace(os.Getenv("GQY_HOME"))
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		home = filepath.Join(userHome, ".gqy")
	}
	return filepath.Abs(filepath.Join(home, "plugins"))
}

func printInstallResult(res *install.Result, out installOutput, loadAfter bool) {
	action := "已安装"
	if res.DryRun {
		action = "校验通过（dry-run，未落盘）"
	}
	fmt.Printf("%s %s@%s\n", action, res.ID, res.Version)
	fmt.Printf("目录: %s\n", res.Dir)
	fmt.Printf("sha256: %s\n", res.SHA256)
	if res.Replaced {
		fmt.Println("已覆盖同 id 的旧目录。")
	}
	if out.ServerSynced {
		fmt.Println("已通知 GQY 重新扫描。")
		if loadAfter {
			fmt.Println("已请求 GQY 加载插件。")
		}
	}
	for _, warning := range out.Warnings {
		fmt.Fprintln(os.Stderr, "警告:", warning)
	}
	if res.DryRun || out.ServerSynced {
		return
	}
	fmt.Println("\n下一步（GQY 已启动时）：")
	fmt.Println("  curl -X POST http://127.0.0.1:5173/api/plugins/reload")
	fmt.Printf("  curl -X POST http://127.0.0.1:5173/api/plugins/%s/load\n", res.ID)
}
