package catalog

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GQYTeam/gqy-plugins/internal/archive"
	"github.com/GQYTeam/gqy-plugins/internal/manifest"
	"github.com/GQYTeam/gqy-plugins/internal/registry"
)

func pluginArchive(t *testing.T, id, version string) []byte {
	t.Helper()
	root := t.TempDir()
	data := fmt.Sprintf(`{"id":%q,"name":"Demo","version":%q,"description":"演示插件","protocol":"gqy-jsonrpc-v1","command":["node","index.js"],"actions":[{"name":"run","description":"运行","schema":{"type":"object"}}]}`, id, version)
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.js"), []byte("console.log('ok')\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), id+".tar.gz")
	if _, err := archive.Pack(root, out); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestInspectLocalArchive(t *testing.T) {
	body := pluginArchive(t, "demo", "1.0.0")
	path := filepath.Join(t.TempDir(), "demo.tar.gz")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := Inspect(context.Background(), path, InspectOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if info.Manifest.ID != "demo" || info.Manifest.Version != "1.0.0" {
		t.Fatalf("manifest = %#v", info.Manifest)
	}
	if info.Size != int64(len(body)) || len(info.SHA256) != 64 {
		t.Fatalf("size=%d sha=%q", info.Size, info.SHA256)
	}
}

func TestSyncOrg(t *testing.T) {
	body := pluginArchive(t, "demo", "1.0.0")
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/orgs/GQYTeam/repos":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[{"name":"gqy-plugins-Demo-default","full_name":"GQYTeam/gqy-plugins-Demo-default","html_url":"https://github.com/GQYTeam/gqy-plugins-Demo-default","default_branch":"main","private":false,"archived":false,"fork":false,"topics":["gqy","demo"],"pushed_at":"2026-11-02T10:00:00Z"}]`)
		case r.URL.Path == "/repos/GQYTeam/gqy-plugins-Demo-default/commits/main":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"sha":"abc123","commit":{"committer":{"date":"2026-11-02T10:05:00Z"}}}`)
		case r.URL.Path == "/repos/GQYTeam/gqy-plugins-Demo-default/tarball/abc123":
			_, _ = w.Write(body)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	regPath := filepath.Join(t.TempDir(), "registry.json")
	if err := os.WriteFile(regPath, []byte(`{"schema_version":1,"updated_at":"2026-11-02T10:00:00Z","plugins":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	opts := SyncOptions{
		Org: "GQYTeam", Prefix: "gqy-plugins-", APIBase: server.URL,
		Token: "test-token", RegistryPath: regPath, Now: now, Client: server.Client(),
	}
	res, err := SyncOrg(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if res.Changed == false || len(res.Included) != 1 || res.Included[0] != "demo" {
		t.Fatalf("sync result = %#v", res)
	}
	doc, err := registry.Load(context.Background(), regPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Plugins) != 1 {
		t.Fatalf("plugins = %d", len(doc.Plugins))
	}
	p := doc.Plugins[0]
	if p.SourceType != "org" || p.Repository != "https://github.com/GQYTeam/gqy-plugins-Demo-default" {
		t.Fatalf("plugin source = %#v", p)
	}
	if len(p.Versions) != 1 || p.Versions[0].Version != "1.0.0" || strings.HasPrefix(p.Versions[0].URL, server.URL) == false {
		t.Fatalf("versions = %#v", p.Versions)
	}
	before, err := os.ReadFile(regPath)
	if err != nil {
		t.Fatal(err)
	}
	res, err = SyncOrg(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if res.Changed {
		t.Fatalf("第二次同步不应有变化: %#v", res)
	}
	after, err := os.ReadFile(regPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("无变化时不应改写 registry.json")
	}
}

func TestExternalValidation(t *testing.T) {
	owner, repo, canonical, err := validateExternalRepository("https://github.com/acme/gqy-demo")
	if err != nil || owner != "acme" || repo != "gqy-demo" || canonical != "https://github.com/acme/gqy-demo" {
		t.Fatalf("validate repo = %q %q %q, %v", owner, repo, canonical, err)
	}
	if _, _, _, err := validateExternalRepository("https://github.com/GQYTeam/demo"); err != nil {
		t.Fatal(err)
	}
	if err := validateExternalArtifact("https://github.com/acme/gqy-demo/releases/download/v1/gqy-demo-1.0.0.tar.gz", "acme", "gqy-demo"); err != nil {
		t.Fatal(err)
	}
	if err := validateExternalArtifact("https://example.com/acme/gqy-demo.tar.gz", "acme", "gqy-demo"); err == nil {
		t.Fatal("非 GitHub Release 应被拒绝")
	}
	if err := validateExternalArtifact("https://github.com/acme/other/releases/download/v1/demo.tar.gz", "acme", "gqy-demo"); err == nil {
		t.Fatal("不同仓库的产物应被拒绝")
	}
}

func TestUpsertConflict(t *testing.T) {
	m := &manifest.Manifest{ID: "demo", Name: "Demo", Version: "1.0.0", Protocol: "gqy-jsonrpc-v1"}
	doc := &registry.Document{SchemaVersion: 1, UpdatedAt: "2026-11-02T10:00:00Z", Plugins: []registry.Plugin{}}
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	first := &PackageInfo{Manifest: m, ArtifactURL: "https://github.com/acme/demo/releases/download/v1/a.tar.gz", SHA256: strings.Repeat("a", 64), Size: 10}
	if _, err := Upsert(doc, first, "external", "https://github.com/acme/demo", nil, now, false); err != nil {
		t.Fatal(err)
	}
	second := &PackageInfo{Manifest: m, ArtifactURL: "https://github.com/acme/demo/releases/download/v1/b.tar.gz", SHA256: strings.Repeat("b", 64), Size: 20}
	_, err := Upsert(doc, second, "external", "https://github.com/acme/demo", nil, now, false)
	var conflict *VersionConflictError
	if errors.As(err, &conflict) == false {
		t.Fatalf("期望 VersionConflictError，得到 %v", err)
	}
	res, err := Upsert(doc, second, "external", "https://github.com/acme/demo", nil, now, true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Conflict == false {
		t.Fatal("existingConflict=true 时应报告冲突")
	}
}

func TestAddExternalRejectsOrganizationAndCrossRepository(t *testing.T) {
	doc := &registry.Document{SchemaVersion: 1, UpdatedAt: "2026-11-02T10:00:00Z", Plugins: []registry.Plugin{}}
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	_, err := AddExternal(context.Background(), doc, ExternalSpec{
		Repository: "https://github.com/GQYTeam/gqy-demo", ArtifactURL: "https://github.com/GQYTeam/gqy-demo/releases/download/v1/gqy-demo.tar.gz",
	}, now)
	if err == nil || strings.Contains(err.Error(), "应走组织内自动收录") == false {
		t.Fatalf("组织仓库应被拒绝，得到: %v", err)
	}
	_, err = AddExternal(context.Background(), doc, ExternalSpec{
		Repository: "https://github.com/acme/gqy-demo", ArtifactURL: "https://github.com/acme/other/releases/download/v1/other.tar.gz",
	}, now)
	if err == nil || strings.Contains(err.Error(), "同一来源仓库") == false {
		t.Fatalf("跨仓库产物应被拒绝，得到: %v", err)
	}
}
