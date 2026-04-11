package handlers

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"clash-subscription-manager/models"

	"github.com/gorilla/mux"
)

func TestSubscribeHandlerDownloadsAndStoresSubscriptionFile(t *testing.T) {
	dataDir := t.TempDir()
	handler := NewHandler(&Config{
		DataDir:         dataDir,
		MaxFileSize:     1024,
		DownloadTimeout: 0,
	})
	var seenHeaders http.Header
	handler.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			seenHeaders = req.Header.Clone()
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("proxy-config-data")),
			}, nil
		}),
	}

	body := bytes.NewBufferString(`{"name":"demo","url":"https://example.com/subscription","filter":"(?i)港|hk|hongkong|hong kong","request_headers":{"User-Agent":"custom-agent","X-Test-Header":"abc123"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/subscribe", body)
	rec := httptest.NewRecorder()

	handler.SubscribeHandler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var resp Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	created, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("response data type = %T, want object", resp.Data)
	}

	filePath, ok := created["file_path"].(string)
	if !ok || filePath == "" {
		t.Fatalf("file_path = %#v, want non-empty string", created["file_path"])
	}

	storedPath := filepath.Join(dataDir, filePath)
	content, err := os.ReadFile(storedPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", storedPath, err)
	}
	if string(content) != "proxy-config-data" {
		t.Fatalf("stored content = %q, want %q", string(content), "proxy-config-data")
	}

	subs, err := ListSubscriptions(filepath.Join(dataDir, "subscriptions.json"))
	if err != nil {
		t.Fatalf("ListSubscriptions() error = %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("len(subs) = %d, want 1", len(subs))
	}
	if subs[0].FilePath != filePath {
		t.Fatalf("subs[0].FilePath = %q, want %q", subs[0].FilePath, filePath)
	}
	if got := seenHeaders.Get("User-Agent"); got != "custom-agent" {
		t.Fatalf("User-Agent header = %q, want %q", got, "custom-agent")
	}
	if got := seenHeaders.Get("X-Test-Header"); got != "abc123" {
		t.Fatalf("X-Test-Header = %q, want %q", got, "abc123")
	}
	if got := subs[0].RequestHeaders["X-Test-Header"]; got != "abc123" {
		t.Fatalf("saved request header = %q, want %q", got, "abc123")
	}
	if got := subs[0].Filter; got != "(?i)港|hk|hongkong|hong kong" {
		t.Fatalf("saved filter = %q, want %q", got, "(?i)港|hk|hongkong|hong kong")
	}
}

func TestDownloadHandlerServesStoredSubscriptionFile(t *testing.T) {
	dataDir := t.TempDir()
	storedName := "subscription-test.txt"
	storedPath := filepath.Join(dataDir, storedName)
	writeHandlerTestFile(t, storedPath, []byte("stored-proxy-file"))

	err := SaveSubscriptions([]models.Subscription{
		{
			ID:       "sub-1",
			Name:     "demo",
			URL:      "http://example.invalid/sub",
			FilePath: storedName,
			FileSize: int64(len("stored-proxy-file")),
		},
	}, filepath.Join(dataDir, "subscriptions.json"))
	if err != nil {
		t.Fatalf("SaveSubscriptions() error = %v", err)
	}

	handler := NewHandler(&Config{
		DataDir:         dataDir,
		MaxFileSize:     1024,
		DownloadTimeout: 0,
	})

	req := httptest.NewRequest(http.MethodGet, "/download/sub-1", nil)
	rec := httptest.NewRecorder()
	req = mux.SetURLVars(req, map[string]string{"id": "sub-1"})

	handler.DownloadHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if rec.Body.String() != "stored-proxy-file" {
		t.Fatalf("body = %q, want %q", rec.Body.String(), "stored-proxy-file")
	}
}

func TestHomeHandlerRendersStaticAssetLinks(t *testing.T) {
	handler := NewHandler(&Config{
		DataDir:         t.TempDir(),
		MaxFileSize:     1024,
		DownloadTimeout: 0,
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.HomeHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `/static/css/style.css`) {
		t.Fatalf("body missing stylesheet link: %s", body)
	}
	if !strings.Contains(body, `/static/js/main.js`) {
		t.Fatalf("body missing script link: %s", body)
	}
	if !strings.Contains(body, `复制下载地址`) {
		t.Fatalf("body missing copy-download wording: %s", body)
	}
	if strings.Contains(body, `选择机场`) {
		t.Fatalf("body should not contain provider selector wording: %s", body)
	}
	if !strings.Contains(body, `自定义请求头`) {
		t.Fatalf("body missing request headers wording: %s", body)
	}
	if strings.Contains(body, `复制默认模板地址`) {
		t.Fatalf("body should not contain default template copy button: %s", body)
	}
	if strings.Contains(body, `复制默认全节点地址`) {
		t.Fatalf("body should not contain default expanded template copy button: %s", body)
	}
	if strings.Contains(body, `x-auth-token`) {
		t.Fatalf("body should not contain x-auth-token template: %s", body)
	}
}

func TestRefreshSubscriptionUpdatesURLHeadersAndCachedFile(t *testing.T) {
	dataDir := t.TempDir()
	cachedName := "sub-1.yaml"
	cachedPath := filepath.Join(dataDir, cachedName)
	writeHandlerTestFile(t, cachedPath, []byte("old-content"))

	err := SaveSubscriptions([]models.Subscription{
		{
			ID:       "sub-1",
			Name:     "before",
			URL:      "https://example.com/old",
			Filter:   "旧规则",
			Type:     "clash",
			FilePath: cachedName,
			FileSize: int64(len("old-content")),
			RequestHeaders: map[string]string{
				"User-Agent": "before-agent",
			},
		},
	}, filepath.Join(dataDir, "subscriptions.json"))
	if err != nil {
		t.Fatalf("SaveSubscriptions() error = %v", err)
	}

	handler := NewHandler(&Config{
		DataDir:         dataDir,
		MaxFileSize:     1024,
		DownloadTimeout: 0,
	})

	var requestedURL string
	var seenHeaders http.Header
	handler.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requestedURL = req.URL.String()
			seenHeaders = req.Header.Clone()
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("new-content")),
			}, nil
		}),
	}

	body := bytes.NewBufferString(`{"name":"after","url":"https://example.com/new","filter":"(?i)港|hk|hongkong|hong kong","request_headers":{"User-Agent":"after-agent","X-Refresh-Key":"new-token"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/subscribe/sub-1/refresh", body)
	req = mux.SetURLVars(req, map[string]string{"id": "sub-1"})
	rec := httptest.NewRecorder()

	handler.RefreshSubscriptionHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if requestedURL != "https://example.com/new" {
		t.Fatalf("requested URL = %q, want %q", requestedURL, "https://example.com/new")
	}
	if got := seenHeaders.Get("X-Refresh-Key"); got != "new-token" {
		t.Fatalf("X-Refresh-Key = %q, want %q", got, "new-token")
	}

	content, err := os.ReadFile(cachedPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", cachedPath, err)
	}
	if string(content) != "new-content" {
		t.Fatalf("cached content = %q, want %q", string(content), "new-content")
	}

	sub, err := GetSubscription("sub-1", filepath.Join(dataDir, "subscriptions.json"))
	if err != nil {
		t.Fatalf("GetSubscription() error = %v", err)
	}
	if sub.Name != "after" {
		t.Fatalf("name = %q, want %q", sub.Name, "after")
	}
	if sub.URL != "https://example.com/new" {
		t.Fatalf("url = %q, want %q", sub.URL, "https://example.com/new")
	}
	if sub.FileSize != int64(len("new-content")) {
		t.Fatalf("file size = %d, want %d", sub.FileSize, len("new-content"))
	}
	if sub.RequestHeaders["User-Agent"] != "after-agent" {
		t.Fatalf("request header User-Agent = %q, want %q", sub.RequestHeaders["User-Agent"], "after-agent")
	}
	if sub.Filter != "(?i)港|hk|hongkong|hong kong" {
		t.Fatalf("filter = %q, want %q", sub.Filter, "(?i)港|hk|hongkong|hong kong")
	}
}

func TestUpdateSubscriptionWithoutRefreshKeepsCachedFile(t *testing.T) {
	dataDir := t.TempDir()
	cachedName := "sub-1.yaml"
	cachedPath := filepath.Join(dataDir, cachedName)
	writeHandlerTestFile(t, cachedPath, []byte("cached-content"))

	lastCheck := time.Now().Add(-2 * time.Hour).UTC()
	updatedAt := time.Now().Add(-time.Hour).UTC()
	err := SaveSubscriptions([]models.Subscription{
		{
			ID:       "sub-1",
			Name:     "before",
			URL:      "https://example.com/old",
			Filter:   "旧规则",
			Type:     "clash",
			FilePath: cachedName,
			FileSize: int64(len("cached-content")),
			UpdatedAt: updatedAt,
			LastCheck: lastCheck,
			RequestHeaders: map[string]string{
				"User-Agent": "before-agent",
			},
			Status: "active",
		},
	}, filepath.Join(dataDir, "subscriptions.json"))
	if err != nil {
		t.Fatalf("SaveSubscriptions() error = %v", err)
	}

	handler := NewHandler(&Config{
		DataDir:         dataDir,
		MaxFileSize:     1024,
		DownloadTimeout: 0,
	})

	body := bytes.NewBufferString(`{"name":"after","url":"https://example.com/new","filter":"(?i)港|hk","request_headers":{"User-Agent":"after-agent"}}`)
	req := httptest.NewRequest(http.MethodPut, "/api/subscribe/sub-1", body)
	req = mux.SetURLVars(req, map[string]string{"id": "sub-1"})
	rec := httptest.NewRecorder()

	handler.SubscriptionHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	content, err := os.ReadFile(cachedPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", cachedPath, err)
	}
	if string(content) != "cached-content" {
		t.Fatalf("cached content = %q, want %q", string(content), "cached-content")
	}

	sub, err := GetSubscription("sub-1", filepath.Join(dataDir, "subscriptions.json"))
	if err != nil {
		t.Fatalf("GetSubscription() error = %v", err)
	}
	if sub.Name != "after" {
		t.Fatalf("name = %q, want %q", sub.Name, "after")
	}
	if sub.URL != "https://example.com/new" {
		t.Fatalf("url = %q, want %q", sub.URL, "https://example.com/new")
	}
	if sub.Filter != "(?i)港|hk" {
		t.Fatalf("filter = %q, want %q", sub.Filter, "(?i)港|hk")
	}
	if sub.RequestHeaders["User-Agent"] != "after-agent" {
		t.Fatalf("request header User-Agent = %q, want %q", sub.RequestHeaders["User-Agent"], "after-agent")
	}
	if sub.FileSize != int64(len("cached-content")) {
		t.Fatalf("file size = %d, want %d", sub.FileSize, len("cached-content"))
	}
	if !sub.LastCheck.Equal(lastCheck) {
		t.Fatalf("last check = %s, want %s", sub.LastCheck, lastCheck)
	}
	if !sub.UpdatedAt.After(updatedAt) {
		t.Fatalf("updated_at = %s, want time after %s", sub.UpdatedAt, updatedAt)
	}
}

func TestSubscribeHandlerDecompressesGzipResponseWhenAcceptEncodingHeaderIsSet(t *testing.T) {
	dataDir := t.TempDir()
	handler := NewHandler(&Config{
		DataDir:         dataDir,
		MaxFileSize:     4096,
		DownloadTimeout: 0,
	})

	handler.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Header.Get("Accept-Encoding"); got != "gzip" {
				t.Fatalf("Accept-Encoding = %q, want %q", got, "gzip")
			}

			var buffer bytes.Buffer
			zipWriter := gzip.NewWriter(&buffer)
			if _, err := zipWriter.Write([]byte("yaml-content")); err != nil {
				t.Fatalf("gzip write error = %v", err)
			}
			if err := zipWriter.Close(); err != nil {
				t.Fatalf("gzip close error = %v", err)
			}

			header := make(http.Header)
			header.Set("Content-Encoding", "gzip")

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     header,
				Body:       io.NopCloser(bytes.NewReader(buffer.Bytes())),
			}, nil
		}),
	}

	body := bytes.NewBufferString(`{"name":"1yunti","url":"https://example.com/subscription","request_headers":{"Accept-Encoding":"gzip"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/subscribe", body)
	rec := httptest.NewRecorder()

	handler.SubscribeHandler(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	subs, err := ListSubscriptions(filepath.Join(dataDir, "subscriptions.json"))
	if err != nil {
		t.Fatalf("ListSubscriptions() error = %v", err)
	}
	content, err := os.ReadFile(filepath.Join(dataDir, subs[0].FilePath))
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(content) != "yaml-content" {
		t.Fatalf("stored content = %q, want %q", string(content), "yaml-content")
	}
}

func writeHandlerTestFile(t *testing.T, path string, data []byte) {
	t.Helper()

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
