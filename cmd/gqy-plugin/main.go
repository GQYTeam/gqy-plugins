// Command gqy-plugin 浏览 GQY 插件在线目录、打包与安装插件。
package main

import (
	"errors"
	"fmt"
	"os"
)

// version 可在构建时用 -ldflags "-X main.version=..." 覆盖。
var version = "0.2.0"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "gqy-plugin:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printHelp()
		return errors.New("缺少命令")
	}
	switch args[0] {
	case "list":
		return runList(args[1:])
	case "search":
		return runSearch(args[1:])
	case "info":
		return runInfo(args[1:])
	case "install":
		return runInstall(args[1:])
	case "verify":
		return runVerify(args[1:])
	case "package":
		return runPackage(args[1:])
	case "sync-org":
		return runSyncOrg(args[1:])
	case "add-external":
		return runAddExternal(args[1:])
	case "version", "--version", "-v":
		fmt.Println("gqy-plugin " + version)
		return nil
	case "help", "--help", "-h":
		printHelp()
		return nil
	default:
		printHelp()
		return fmt.Errorf("未知命令 %q", args[0])
	}
}

func printHelp() {
	fmt.Print(`gqy-plugin — GQY 插件在线目录客户端

用法:
  gqy-plugin list    [--registry URL|PATH] [--json]
  gqy-plugin search  <关键词> [--registry URL|PATH] [--json]
  gqy-plugin info    <id> [--registry URL|PATH] [--json]
  gqy-plugin install <id>[@version] [--plugins-dir DIR] [--force] [--dry-run]
                     [--server URL] [--token TOKEN] [--load] [--json]
  gqy-plugin verify  [--registry URL|PATH] [--strict] [--artifacts] [--json]
  gqy-plugin package <插件目录> [--output PATH] [--json]
  gqy-plugin sync-org [--org GQYTeam] [--registry PATH] [--json]
  gqy-plugin add-external --repository URL --artifact-url URL [--issue-body PATH]
                           [--registry PATH] [--json]
  gqy-plugin version

环境变量:
  GQY_PLUGIN_REGISTRY  目录地址，默认官方 main 分支的 registry.json
  GQY_HOME             GQY 数据目录，默认 ~/.gqy
  GQY_AUTH_TOKEN       GQY 管理接口令牌（也可用 --token）
  GITHUB_TOKEN         组织插件自动收录用的 GitHub API token

安装只把文件原子放到 $GQY_HOME/plugins/<id>/；默认不自动加载。
`)
}
