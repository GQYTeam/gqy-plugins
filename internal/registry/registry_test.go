package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

// repoRegistry 是仓库根的在线目录文件；测试直接读生产数据，不用夹具。
const repoRegistry = "../../registry.json"

// validSHA256 是格式合法的 sha256 占位值，只用于构造校验样本。
const validSHA256 = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

const testUpdatedAt = "2026-11-02T21:05:35+08:00"

func docJSON(schemaVersion int, extra string) string {
	return fmt.Sprintf(`{"schema_version": %d, "updated_at": %q, "plugins": []%s}`,
		schemaVersion, testUpdatedAt, extra)
}

func docWithPlugin(plugin string) string {
	return fmt.Sprintf(`{"schema_version": 1, "updated_at": %q, "plugins": [%s]}`,
		testUpdatedAt, plugin)
}

func pluginJSON(id, version, artifactURL, sha string) string {
	return fmt.Sprintf(
		`{"id": %q, "name": "示例插件", "summary": "仅用于校验测试", "versions": [{"version": %q, "url": %q, "sha256": %q}]}`,
		id, version, artifactURL, sha)
}

func TestParseRepositoryRegistry(t *testing.T) {
	data, err := os.ReadFile(repoRegistry)
	if err != nil {
		t.Fatalf("读取仓库根 registry.json 失败: %v", err)
	}
	doc, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse(registry.json) 出错: %v", err)
	}

	// 用独立的结构体读同一份文件，确认 Parse 没有丢字段或改写值。
	var raw struct {
		SchemaVersion int    `json:"schema_version"`
		UpdatedAt     string `json:"updated_at"`
		Plugins       []struct {
			ID string `json:"id"`
		} `json:"plugins"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("解析 registry.json 原始内容失败: %v", err)
	}
	if doc.SchemaVersion != SchemaVersion || doc.SchemaVersion != raw.SchemaVersion {
		t.Errorf("schema_version = %d（文件里 %d）, 期望 %d",
			doc.SchemaVersion, raw.SchemaVersion, SchemaVersion)
	}
	if doc.UpdatedAt != raw.UpdatedAt {
		t.Errorf("updated_at = %q, 与文件里的 %q 不一致", doc.UpdatedAt, raw.UpdatedAt)
	}
	if len(doc.Plugins) != len(raw.Plugins) {
		t.Errorf("plugins 长度 = %d, 文件里是 %d", len(doc.Plugins), len(raw.Plugins))
	}
	if doc.Plugins == nil {
		t.Fatal("plugins 为 nil，期望数组")
	}
	if len(doc.Plugins) < 2 {
		t.Fatalf("plugins 长度 = %d，至少应收录三个组织内插件", len(doc.Plugins))
	}
	seen := make(map[string]bool)
	for _, p := range doc.Plugins {
		seen[p.ID] = true
	}
	for _, id := range []string{"show-images-default", "emote-pack-default"} {
		if seen[id] == false {
			t.Errorf("registry.json 缺少官方插件 %s", id)
		}
	}
}

func TestLoadRepositoryRegistryFromPath(t *testing.T) {
	ctx := context.Background()
	doc, err := Load(ctx, repoRegistry)
	if err != nil {
		t.Fatalf("Load(%s) 出错: %v", repoRegistry, err)
	}
	if doc.SchemaVersion != SchemaVersion {
		t.Errorf("schema_version = %d, 期望 %d", doc.SchemaVersion, SchemaVersion)
	}

	got, err := doc.ResolveURL("dist/example-1.0.0.zip")
	if err != nil {
		t.Fatalf("ResolveURL 出错: %v", err)
	}
	if !strings.HasPrefix(got, "file://") || !strings.HasSuffix(got, "/dist/example-1.0.0.zip") {
		t.Errorf("ResolveURL = %q, 期望 file:// 绝对地址且保留相对路径", got)
	}

	if _, err := Load(ctx, "   "); err == nil {
		t.Fatal("Load 空来源应当报错")
	} else if !strings.Contains(err.Error(), "来源为空") {
		t.Errorf("空来源错误信息 = %q, 期望包含「来源为空」", err)
	}

	if _, err := Load(ctx, "../../registry-does-not-exist.json"); err == nil {
		t.Fatal("Load 不存在的文件应当报错")
	}
}

func TestParseRejectsUnknownField(t *testing.T) {
	_, err := Parse([]byte(docJSON(SchemaVersion, `, "extra_field": true`)))
	if err == nil {
		t.Fatal("未知字段应当被拒绝")
	}
	if !strings.Contains(err.Error(), `unknown field "extra_field"`) {
		t.Errorf("错误信息 = %q, 期望包含 unknown field \"extra_field\"", err)
	}
}

func TestParseRejectsWrongSchemaVersion(t *testing.T) {
	for _, version := range []int{0, 2, 99} {
		_, err := Parse([]byte(docJSON(version, "")))
		if err == nil {
			t.Fatalf("schema_version=%d 应当被拒绝", version)
		}
		if !strings.Contains(err.Error(), "schema_version 必须为 1") {
			t.Errorf("schema_version=%d 错误信息 = %q, 期望包含「schema_version 必须为 1」", version, err)
		}
	}
}

func TestParseRejectsInvalidPluginID(t *testing.T) {
	for _, id := range []string{"", "Bad_ID", "-starter", "has space", "a2345678901234567890123456789012345"} {
		plugin := pluginJSON(id, "1.0.0", "https://example.com/example.zip", validSHA256)
		_, err := Parse([]byte(docWithPlugin(plugin)))
		if err == nil {
			t.Fatalf("id %q 应当被拒绝", id)
		}
		if !strings.Contains(err.Error(), "id 必须匹配") {
			t.Errorf("id %q 错误信息 = %q, 期望包含「id 必须匹配」", id, err)
		}
	}
}

func TestParseRejectsInvalidSHA256(t *testing.T) {
	for _, sha := range []string{
		"",
		"abc",
		strings.ToUpper(validSHA256),
		validSHA256[:63],
		validSHA256 + "0",
	} {
		plugin := pluginJSON("demo", "1.0.0", "https://example.com/example.zip", sha)
		_, err := Parse([]byte(docWithPlugin(plugin)))
		if err == nil {
			t.Fatalf("sha256 %q 应当被拒绝", sha)
		}
		if !strings.Contains(err.Error(), "sha256 必须是 64 位小写十六进制") {
			t.Errorf("sha256 %q 错误信息 = %q, 期望包含 sha256 格式提示", sha, err)
		}
	}
}

func TestParseRejectsInvalidVersion(t *testing.T) {
	for _, version := range []string{"1.0", "v1.0.0", "1.0.0-beta", ""} {
		plugin := pluginJSON("demo", version, "https://example.com/example.zip", validSHA256)
		_, err := Parse([]byte(docWithPlugin(plugin)))
		if err == nil {
			t.Fatalf("version %q 应当被拒绝", version)
		}
		if !strings.Contains(err.Error(), "version 必须匹配") {
			t.Errorf("version %q 错误信息 = %q, 期望包含「version 必须匹配」", version, err)
		}
	}
}

func TestValidateStrictRequiresHTTPS(t *testing.T) {
	doc, err := Parse([]byte(docWithPlugin(
		pluginJSON("demo", "1.0.0", "https://example.com/example.zip", validSHA256))))
	if err != nil {
		t.Fatalf("https 绝对地址的样本应当解析通过: %v", err)
	}
	if err := doc.Validate(true); err != nil {
		t.Fatalf("https 绝对地址应当通过 strict 校验: %v", err)
	}

	relative, err := Parse([]byte(docWithPlugin(
		pluginJSON("demo", "1.0.0", "dist/example.zip", validSHA256))))
	if err != nil {
		t.Fatalf("非 strict 模式应当接受相对地址: %v", err)
	}
	err = relative.Validate(true)
	if err == nil {
		t.Fatal("strict 模式应当拒绝相对地址")
	}
	if !strings.Contains(err.Error(), "strict 模式要求 url 是 https 绝对地址") {
		t.Errorf("strict 错误信息 = %q, 期望包含 strict 模式要求", err)
	}
}
