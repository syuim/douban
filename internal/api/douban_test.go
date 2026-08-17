package api

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "douban-api-test")
	if err != nil {
		panic(err)
	}
	_ = os.Setenv("DATABASE_PATH", dir+"/test.db")
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func newForwardingProxy(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload proxyPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		req, err := http.NewRequest(payload.Method, payload.URL, io.NopCloser(strings.NewReader(payload.Body)))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for k, v := range payload.Headers {
			req.Header.Set(k, v)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		for k, values := range resp.Header {
			for _, v := range values {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(body)
	}))
}

func TestProxyFetchRetriesAndReturnsFinalStatus(t *testing.T) {
	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			http.Error(w, "retry", http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	ctx := context.Background()
	resp, err := ProxyFetch(ctx, srv.URL, ProxyRequest{URL: "https://example.com", Method: "GET"})
	if err != nil {
		t.Fatalf("ProxyFetch error: %v", err)
	}
	defer resp.Body.Close()

	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestProxyFetchRetriesNetworkError(t *testing.T) {
	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			panic(http.ErrAbortHandler)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	ctx := context.Background()
	resp, err := ProxyFetch(ctx, srv.URL, ProxyRequest{URL: "https://example.com", Method: "GET"})
	if err != nil {
		t.Fatalf("ProxyFetch error: %v", err)
	}
	defer resp.Body.Close()

	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestGetDoulistItemsMaxPages(t *testing.T) {
	var paths []string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[],"total":0}`))
	}))
	defer target.Close()

	proxy := newForwardingProxy(t)
	defer proxy.Close()
	t.Setenv("PROXY_URL", proxy.URL)
	d := &DoubanAPI{BaseAPI: NewBaseAPI(target.URL, map[string]string{})}
	items, err := d.GetDoulistItems(context.Background(), "test", 1)
	if err != nil {
		t.Fatalf("GetDoulistItems error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected no items, got %d", len(items))
	}
	if len(paths) != 1 {
		t.Fatalf("expected 1 request, got %d", len(paths))
	}
}

func TestGetDoulistItemsMaxPagesStopsAfterPageCount(t *testing.T) {
	var targetRequests int
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetRequests++
		w.Header().Set("Content-Type", "application/json")
		items := make([]map[string]any, 25)
		for i := range items {
			items[i] = map[string]any{"id": i + 1, "target_id": 100 + i + 1, "type": "movie", "title": "A"}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"items": items, "total": 100})
		t.Logf("target request %s", r.URL.String())
	}))
	defer target.Close()

	proxy := newForwardingProxy(t)
	defer proxy.Close()
	t.Setenv("PROXY_URL", proxy.URL)
	d := &DoubanAPI{BaseAPI: NewBaseAPI(target.URL, map[string]string{})}
	// 与 TestGetDoulistItemsMaxPages 使用不同 doulist ID，避免命中其缓存
	items, err := d.GetDoulistItems(context.Background(), "test-paged", 2)
	if err != nil {
		t.Fatalf("GetDoulistItems error: %v", err)
	}
	t.Logf("items=%d", len(items))
	if len(items) != 50 {
		t.Fatalf("expected 50 items from 2 pages, got %d", len(items))
	}
	if targetRequests != 2 {
		t.Fatalf("expected 2 target requests, got %d", targetRequests)
	}
}

func TestGetTmdbAPIReusesInstance(t *testing.T) {
	a := GetTmdbAPI("key-a")
	b := GetTmdbAPI("key-a")
	if a != b {
		t.Fatal("expected same TMDB instance for same key")
	}
}

func TestGetTmdbAPIHandlesConcurrentFirstAccess(t *testing.T) {
	done := make(chan *TmdbAPI, 16)
	for i := 0; i < 16; i++ {
		go func() {
			done <- GetTmdbAPI("concurrent-key")
		}()
	}
	seen := make(map[*TmdbAPI]bool)
	for i := 0; i < 16; i++ {
		seen[<-done] = true
	}
	if len(seen) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(seen))
	}
}

func TestGetFanartAPIReusesInstance(t *testing.T) {
	a := GetFanartAPI("key-a")
	b := GetFanartAPI("key-a")
	if a != b {
		t.Fatal("expected same Fanart instance for same key")
	}
}

func TestProxyFetchRespectsContextCancellation(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("tcp listen: %v", err)
	}
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				time.Sleep(5 * time.Second)
			}(conn)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err = proxyFetchWithHTTPClient(ctx, http.DefaultClient, "http://"+ln.Addr().String(), ProxyRequest{URL: "https://example.com", Method: "GET"})
	if err == nil {
		t.Fatal("expected context error")
	}
}
