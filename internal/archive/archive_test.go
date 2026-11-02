package archive

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testManifest = `{"id":"demo","name":"Demo","version":"1.0.0","protocol":"gqy-jsonrpc-v1","command":["node","index.js"],"actions":[{"name":"run","description":"运行","schema":{"type":"object"}}]}`

func makePlugin(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(testManifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.js"), []byte("console.log('ok')\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".DS_Store"), []byte("junk"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=x"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestPackDeterministicAndExcludesJunk(t *testing.T) {
	dir := makePlugin(t)
	out1 := filepath.Join(t.TempDir(), "a.tar.gz")
	out2 := filepath.Join(t.TempDir(), "b.tar.gz")
	a, err := Pack(dir, out1)
	if err != nil {
		t.Fatalf("Pack 1 出错: %v", err)
	}
	b, err := Pack(dir, out2)
	if err != nil {
		t.Fatalf("Pack 2 出错: %v", err)
	}
	if a.SHA256 != b.SHA256 || a.Size != b.Size {
		t.Fatalf("同一目录两次打包不一致: %#v vs %#v", a, b)
	}
	f, err := os.Open(out1)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)
	var names []string
	for {
		h, err := tr.Next()
		if err != nil {
			break
		}
		names = append(names, h.Name)
	}
	joined := strings.Join(names, "\n")
	if strings.Contains(joined, ".DS_Store") || strings.Contains(joined, ".env") {
		t.Fatalf("包内包含应排除的文件: %v", names)
	}
	if strings.Contains(joined, "manifest.json") == false || strings.Contains(joined, "index.js") == false {
		t.Fatalf("包内缺少运行文件: %v", names)
	}
}

func TestExtractTarGzRoundTrip(t *testing.T) {
	dir := makePlugin(t)
	out := filepath.Join(t.TempDir(), "demo.tar.gz")
	if _, err := Pack(dir, out); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	dst := t.TempDir()
	if err := ExtractTarGz(f, dst); err != nil {
		t.Fatalf("ExtractTarGz 出错: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dst, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != testManifest {
		t.Fatalf("manifest 内容不一致: %s", got)
	}
}

func TestExtractRejectsTraversal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "evil.tar.gz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	body := []byte("owned")
	h := &tar.Header{Name: "../owned.txt", Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}
	if err := tw.WriteHeader(h); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	_ = tw.Close()
	_ = gz.Close()
	_ = f.Close()
	ef, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer ef.Close()
	err = ExtractTarGz(ef, t.TempDir())
	if err == nil || strings.Contains(err.Error(), "非法路径") == false {
		t.Fatalf("越界路径应被拒绝，得到: %v", err)
	}
}
