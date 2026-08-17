package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// newTargetServer 返回一个返回固定 body 的目标服务
func newTargetServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func TestRequestServesFromCache(t *testing.T) {
	var requests atomic.Int32
	target := newTargetServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte(`"ok"`))
	})
	defer target.Close()

	proxy := newForwardingProxy(t)
	defer proxy.Close()
	t.Setenv("PROXY_URL", proxy.URL)

	d := NewBaseAPI(target.URL, map[string]string{})
	ctx := context.Background()
	cacheCfg := &CacheConfig{Key: "cache-hit-test-key", TTL: 3600}

	// 第一次请求应打到上游
	data, status, _, err := d.Request(ctx, "GET", "/data", nil, nil, nil, cacheCfg)
	if err != nil || status != 200 || string(data) != `"ok"` {
		t.Fatalf("first request: data=%s status=%d err=%v", data, status, err)
	}
	if requests.Load() != 1 {
		t.Fatalf("expected 1 upstream request, got %d", requests.Load())
	}

	// 第二次请求应命中缓存，不再打上游
	data, _, _, err = d.Request(ctx, "GET", "/data", nil, nil, nil, cacheCfg)
	if err != nil || string(data) != `"ok"` {
		t.Fatalf("cached request: data=%s err=%v", data, err)
	}
	if requests.Load() != 1 {
		t.Fatalf("cached request hit upstream: %d requests", requests.Load())
	}
}

func TestRequestConcurrentDedup(t *testing.T) {
	var requests atomic.Int32
	target := newTargetServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		time.Sleep(150 * time.Millisecond)
		_, _ = w.Write([]byte(`"ok"`))
	})
	defer target.Close()

	proxy := newForwardingProxy(t)
	defer proxy.Close()
	t.Setenv("PROXY_URL", proxy.URL)

	d := NewBaseAPI(target.URL, map[string]string{})
	cacheCfg := &CacheConfig{Key: "dedup-test-key", TTL: 3600}

	const n = 8
	start := make(chan struct{})
	errs := make(chan error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			data, status, _, err := d.Request(ctx, "GET", "/data", nil, nil, nil, cacheCfg)
			if err != nil {
				errs <- err
				return
			}
			if status != 200 || string(data) != `"ok"` {
				errs <- fmt.Errorf("unexpected response: data=%s status=%d", data, status)
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("request failed: %v", err)
	}

	if requests.Load() != 1 {
		t.Fatalf("expected 1 upstream request, got %d", requests.Load())
	}
}

func TestRequestWaiterRetriesAfterLeaderFailure(t *testing.T) {
	var requests atomic.Int32
	target := newTargetServer(t, func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			// 4xx（非 429/418）不会被 ProxyFetch 重试，作为 leader 的确定性失败
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(`"ok"`))
	})
	defer target.Close()

	proxy := newForwardingProxy(t)
	defer proxy.Close()
	t.Setenv("PROXY_URL", proxy.URL)

	d := NewBaseAPI(target.URL, map[string]string{})
	cacheCfg := &CacheConfig{Key: "retry-test-key", TTL: 3600}

	const n = 4
	start := make(chan struct{})
	statuses := make(chan int, n)
	errs := make(chan error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, status, _, err := d.Request(ctx, "GET", "/data", nil, nil, nil, cacheCfg)
			if err != nil {
				errs <- err
				return
			}
			statuses <- status
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	close(statuses)
	for err := range errs {
		t.Fatalf("request failed: %v", err)
	}

	// 首个 leader 失败（404，未写缓存），其余等待者中应只有一人重试成功（200）
	var got404, got200 int
	for status := range statuses {
		switch status {
		case 200:
			got200++
		case 404:
			got404++
		default:
			t.Fatalf("unexpected status %d", status)
		}
	}
	if got404 != 1 {
		t.Fatalf("expected exactly 1 failed response, got %d", got404)
	}
	if requests.Load() != 2 {
		t.Fatalf("expected 2 upstream requests (1 fail + 1 retry), got %d", requests.Load())
	}
}
