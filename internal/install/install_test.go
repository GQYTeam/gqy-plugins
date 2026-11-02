package install

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GQYTeam/gqy-plugins/internal/archive"
	"github.com/GQYTeam/gqy-plugins/internal/registry"
)

func buildPackage(t *testing.T, version string) (*archive.Summary, []byte) {
	t.Helper()
	root := t.TempDir()
	manifest := fmt.Sprintf(`{"id":"demo","name":"Demo","version":%q,"protocol":"gqy-jsonrpc-v1","command":["node","index.js"],"actions":[{"name":"run","description":"运行","schema":{"type":"object"}}]}`, version)
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.js"), []byte("console.log('demo')\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "demo.tar.gz")
	sum, err := archive.Pack(root, out)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	return sum, data
}

func servePackage(t *testing.T, data []byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(data)
	}))
}

func makeDoc(t *testing.T, serverURL string, sum *archive.Summary, mode string) *registry.Document {
	t.Helper()
	entry := map[string]any{
		"schema_version": 1,
		"updated_at":     "2026-11-02T21:25:00+08:00",
		"plugins": []any{map[string]any{
			"id": "demo", "name": "Demo", "summary": "测试插件",
			"versions": []any{map[string]any{
				"version": sum.Version, "mode": mode, "protocol": "gqy-jsonrpc-v1",
				"url": serverURL + "/demo.tar.gz", "sha256": sum.SHA256, "size": sum.Size,
			}},
		}},
	}
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := registry.ParseJSON(data, "")
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestInstallEndToEndAndForceReplace(t *testing.T) {
	sum, data := buildPackage(t, "1.0.0")
	server := servePackage(t, data)
	defer server.Close()
	doc := makeDoc(t, server.URL, sum, "resident")
	pluginsDir := t.TempDir()

	res, err := Install(context.Background(), doc, "demo", "", Options{PluginsDir: pluginsDir})
	if err != nil {
		t.Fatalf("Install 出错: %v", err)
	}
	if res.Version != "1.0.0" || res.Replaced {
		t.Fatalf("安装结果不对: %#v", res)
	}
	body, err := os.ReadFile(filepath.Join(pluginsDir, "demo", "index.js"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "demo") == false {
		t.Fatalf("安装后的文件内容不对: %s", body)
	}
	if _, err := Install(context.Background(), doc, "demo", "", Options{PluginsDir: pluginsDir}); err == nil {
		t.Fatal("已存在时未加 --force 应报错")
	}
	again, err := Install(context.Background(), doc, "demo", "", Options{PluginsDir: pluginsDir, Force: true})
	if err != nil {
		t.Fatalf("强制覆盖出错: %v", err)
	}
	if again.Replaced == false {
		t.Fatal("强制覆盖结果应标记 Replaced")
	}
	leftovers, err := filepath.Glob(filepath.Join(pluginsDir, ".gqy-plugin-backup-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(leftovers) != 0 {
		t.Fatalf("覆盖后残留备份: %v", leftovers)
	}
}

func TestInstallRejectsHashMismatchAndOneshot(t *testing.T) {
	sum, data := buildPackage(t, "1.0.0")
	server := servePackage(t, data)
	defer server.Close()
	doc := makeDoc(t, server.URL, sum, "resident")
	pluginsDir := t.TempDir()
	doc.Plugins[0].Versions[0].SHA256 = strings.Repeat("0", 64)
	if _, err := Install(context.Background(), doc, "demo", "", Options{PluginsDir: pluginsDir}); err == nil || strings.Contains(err.Error(), "sha256 不匹配") == false {
		t.Fatalf("哈希不匹配应被拒绝，得到: %v", err)
	}
	doc = makeDoc(t, server.URL, sum, "oneshot")
	if _, err := Install(context.Background(), doc, "demo", "", Options{PluginsDir: pluginsDir}); err == nil || strings.Contains(err.Error(), "一次性插件") == false {
		t.Fatalf("oneshot 应被拒绝，得到: %v", err)
	}
}
