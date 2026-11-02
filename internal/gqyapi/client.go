// Package gqyapi 在安装后通知本机 GQY 重新扫描并可选加载插件。
package gqyapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Reload 调用 POST /api/plugins/reload。
func Reload(ctx context.Context, server, token string) error {
	return post(ctx, server, token, "/api/plugins/reload")
}

// Load 调用 POST /api/plugins/{id}/load。
func Load(ctx context.Context, server, token, id string) error {
	if id == "" {
		return fmt.Errorf("gqyapi: 插件 id 为空")
	}
	return post(ctx, server, token, "/api/plugins/"+url.PathEscape(id)+"/load")
}

func post(ctx context.Context, server, token, path string) error {
	base, err := url.Parse(strings.TrimSpace(server))
	if err != nil {
		return fmt.Errorf("gqyapi: 服务地址非法: %w", err)
	}
	if base.Scheme != "http" && base.Scheme != "https" {
		return fmt.Errorf("gqyapi: 服务地址必须是 http(s)")
	}
	if base.Host == "" {
		return fmt.Errorf("gqyapi: 服务地址缺少 host")
	}
	base.Path = strings.TrimRight(base.Path, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base.String(), nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("gqyapi: 请求失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		text := strings.TrimSpace(string(body))
		if text == "" {
			text = resp.Status
		}
		return fmt.Errorf("gqyapi: HTTP %d: %s", resp.StatusCode, text)
	}
	return nil
}
