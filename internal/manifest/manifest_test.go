package manifest

import (
	"strings"
	"testing"
)

const validJSON = `{
  "id":"echo",
  "name":"Echo 示例",
  "version":"1.0.0",
  "protocol":"gqy-jsonrpc-v1",
  "command":["node","index.js"],
  "cwd":".",
  "timeout_ms":10000,
  "concurrency":"concurrent",
  "actions":[{"name":"echo","description":"回显","schema":{"type":"object"}}],
  "provides":["text-echo"],
  "requires":[]
}`

func TestParseValid(t *testing.T) {
	m, err := Parse([]byte(validJSON))
	if err != nil {
		t.Fatalf("Parse 出错: %v", err)
	}
	if m.ID != "echo" || m.Version != "1.0.0" || m.TimeoutMS != 10000 {
		t.Fatalf("解析结果不对: %#v", m)
	}
}

func TestParseRejectsBadFields(t *testing.T) {
	cases := []struct {
		name string
		old  string
		new  string
		want string
	}{
		{"空 id", `"id":"echo"`, `"id":""`, "id 非法"},
		{"空 name", `"name":"Echo 示例"`, `"name":""`, "name 必须"},
		{"坏版本", `"version":"1.0.0"`, `"version":"v1"`, "version 非法"},
		{"缺 command", `"command":["node","index.js"]`, `"command":[]`, "需要 command"},
		{"越界 cwd", `"cwd":"."`, `"cwd":"../x"`, "cwd 必须"},
		{"坏 action", `"name":"echo"`, `"name":"1bad"`, "action name 非法"},
		{"坏 schema", `{"type":"object"}`, `{"type":"string"}`, "schema 根 type"},
		{"坏 mode", `"requires":[]`, `"requires":[],"mode":"daemon"`, "mode 只能"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := validJSON
			if strings.Contains(input, tc.old) == false {
				t.Fatalf("样本缺少替换点 %q", tc.old)
			}
			input = strings.Replace(input, tc.old, tc.new, 1)
			_, err := Parse([]byte(input))
			if err == nil {
				t.Fatalf("期望报错，输入: %s", input)
			}
			if strings.Contains(err.Error(), tc.want) == false {
				t.Fatalf("错误 = %q，期望包含 %q", err, tc.want)
			}
		})
	}
}
