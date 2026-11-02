package main

import (
	"flag"
	"testing"
)

func TestParseInterspersed(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	force := fs.Bool("force", false, "force")
	name := fs.String("name", "", "name")
	pos, err := parseInterspersed(fs, []string{"demo", "--force", "--name", "x"})
	if err != nil {
		t.Fatal(err)
	}
	if len(pos) != 1 || pos[0] != "demo" {
		t.Fatalf("pos = %v", pos)
	}
	if *force == false || *name != "x" {
		t.Fatalf("flags force=%v name=%q", *force, *name)
	}
}

func TestSplitIDVersion(t *testing.T) {
	id, ver := splitIDVersion("demo@1.2.3")
	if id != "demo" || ver != "1.2.3" {
		t.Fatalf("split = %q %q", id, ver)
	}
	id, ver = splitIDVersion("demo")
	if id != "demo" || ver != "" {
		t.Fatalf("split = %q %q", id, ver)
	}
}

func TestParseIssueForm(t *testing.T) {
	body := `### 插件 id

demo

### 版本

1.2.3

### 来源仓库

https://github.com/acme/gqy-demo

### 产物 URL

https://github.com/acme/gqy-demo/releases/download/v1.2.3/gqy-demo-1.2.3.tar.gz

### SHA-256

` + "`" + `0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef` + "`" + `

### 产物字节数

12345
`
	fields := parseIssueForm(body)
	if fieldValue(fields, "来源仓库") != "https://github.com/acme/gqy-demo" {
		t.Fatalf("repository = %q", fieldValue(fields, "来源仓库"))
	}
	if fieldValue(fields, "产物 url") == "" {
		t.Fatal("缺少产物 URL")
	}
	if fieldValue(fields, "sha-256") != "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" {
		t.Fatalf("sha = %q", fieldValue(fields, "sha-256"))
	}
	if fieldValue(fields, "产物字节数") != "12345" {
		t.Fatalf("size = %q", fieldValue(fields, "产物字节数"))
	}
}
