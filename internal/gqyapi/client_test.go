package gqyapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReloadAndLoad(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, 期望 POST", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		paths = append(paths, r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()
	if err := Reload(context.Background(), server.URL, "secret"); err != nil {
		t.Fatal(err)
	}
	if err := Load(context.Background(), server.URL, "secret", "demo"); err != nil {
		t.Fatal(err)
	}
	want := "/api/plugins/reload,/api/plugins/demo/load"
	if strings.Join(paths, ",") != want {
		t.Fatalf("paths = %v, 期望 %s", paths, want)
	}
}

func TestPostError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no", http.StatusUnauthorized)
	}))
	defer server.Close()
	err := Reload(context.Background(), server.URL, "bad")
	if err == nil || strings.Contains(err.Error(), "401") == false {
		t.Fatalf("期望 401 错误，得到: %v", err)
	}
	if _, err := http.NewRequest(http.MethodGet, server.URL, nil); err != nil {
		t.Fatal(err)
	}
}
