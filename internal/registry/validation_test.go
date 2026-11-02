package registry

import (
	"fmt"
	"strings"
	"testing"
)

func TestValidateRejectsBadVersionMetadata(t *testing.T) {
	cases := []struct {
		name   string
		fields string
		want   string
	}{
		{"mode", `"mode":"daemon",`, "mode 只能"},
		{"protocol", `"protocol":"carrier-pigeon",`, "protocol 非法"},
		{"action", `"actions":["1bad"],`, "action 名非法"},
		{"provides", `"provides":["Bad Service"],`, "provides 项非法"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plugin := fmt.Sprintf(`{"id":"demo","name":"Demo","summary":"测试","versions":[{%s"version":"1.0.0","url":"https://example.com/demo.tar.gz","sha256":%q}]}`, tc.fields, validSHA256)
			_, err := Parse([]byte(docWithPlugin(plugin)))
			if err == nil || strings.Contains(err.Error(), tc.want) == false {
				t.Fatalf("期望错误包含 %q，得到: %v", tc.want, err)
			}
		})
	}
}
