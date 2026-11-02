package registry

import (
	"strings"
	"testing"
)

func TestResolveVersionLatestAndYanked(t *testing.T) {
	p := &Plugin{ID: "demo", Versions: []Version{
		{Version: "1.0.0", SHA256: validSHA256, URL: "https://example.com/1.tar.gz"},
		{Version: "1.2.0", SHA256: validSHA256, URL: "https://example.com/2.tar.gz"},
		{Version: "2.0.0", SHA256: validSHA256, URL: "https://example.com/3.tar.gz", Yanked: true},
	}}
	v, err := p.ResolveVersion("")
	if err != nil {
		t.Fatal(err)
	}
	if v.Version != "1.2.0" {
		t.Fatalf("最新版本 = %s，期望 1.2.0", v.Version)
	}
	if _, err := p.ResolveVersion("2.0.0"); err == nil || strings.Contains(err.Error(), "撤回") == false {
		t.Fatalf("yanked 版本应被拒绝，得到: %v", err)
	}
	if _, err := p.ResolveVersion("9.9.9"); err == nil || strings.Contains(err.Error(), "没有版本") == false {
		t.Fatalf("不存在版本应报错，得到: %v", err)
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.2.3", "1.2.2", 1},
		{"1.2.2", "1.3.0", -1},
		{"10.0.0", "9.9.9", 1},
	}
	for _, tc := range cases {
		if got := CompareVersions(tc.a, tc.b); got != tc.want {
			t.Errorf("CompareVersions(%s,%s)=%d，期望 %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestFind(t *testing.T) {
	doc := &Document{Plugins: []Plugin{{ID: "a"}, {ID: "b"}}}
	if p, err := doc.Find("b"); err != nil || p.ID != "b" {
		t.Fatalf("Find = %#v, %v", p, err)
	}
	if _, err := doc.Find("c"); err == nil {
		t.Fatal("不存在的插件应报错")
	}
}
