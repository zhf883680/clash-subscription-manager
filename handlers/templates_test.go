package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"clash-subscription-manager/models"

	"github.com/gorilla/mux"
)

func TestTemplateLifecycleAndDefaultSelection(t *testing.T) {
	tempDir := t.TempDir()
	templatesFile := filepath.Join(tempDir, "templates.json")

	first, err := AddTemplate(models.Template{
		Name:      "default",
		Content:   "port: 7890\n",
		UpdatedAt: time.Now(),
	}, templatesFile)
	if err != nil {
		t.Fatalf("AddTemplate(first) error = %v", err)
	}
	if !first.IsDefault {
		t.Fatalf("first template should become default")
	}

	second, err := AddTemplate(models.Template{
		Name:      "backup",
		Content:   "mixed-port: 7890\n",
		UpdatedAt: time.Now(),
	}, templatesFile)
	if err != nil {
		t.Fatalf("AddTemplate(second) error = %v", err)
	}
	if second.IsDefault {
		t.Fatalf("second template should not be default initially")
	}

	if _, err := SetDefaultTemplate(second.ID, templatesFile); err != nil {
		t.Fatalf("SetDefaultTemplate() error = %v", err)
	}

	templates, err := ListTemplates(templatesFile)
	if err != nil {
		t.Fatalf("ListTemplates() error = %v", err)
	}
	if len(templates) != 2 {
		t.Fatalf("len(templates) = %d, want 2", len(templates))
	}

	var defaultCount int
	for _, item := range templates {
		if item.IsDefault {
			defaultCount++
			if item.ID != second.ID {
				t.Fatalf("default template id = %q, want %q", item.ID, second.ID)
			}
		}
	}
	if defaultCount != 1 {
		t.Fatalf("defaultCount = %d, want 1", defaultCount)
	}
}

func TestRenderTemplateHandlerInjectsCurrentSubscriptionsAsProxyProviders(t *testing.T) {
	dataDir := t.TempDir()

	if err := SaveSubscriptions([]models.Subscription{
		{
			ID:        "sub-west",
			Name:      "🛫 West",
			URL:       "https://example.com/west",
			Filter:    "(?i)港|hk|hongkong|hong kong",
			FilePath:  "west-source.yaml",
			Type:      "clash",
			UpdatedAt: time.Now(),
			Status:    "active",
		},
		{
			ID:        "sub-ai",
			Name:      "✨ AI Select",
			URL:       "https://example.com/ai",
			FilePath:  "ai-source.yaml",
			Type:      "clash",
			UpdatedAt: time.Now(),
			Status:    "active",
		},
	}, filepath.Join(dataDir, "subscriptions.json")); err != nil {
		t.Fatalf("SaveSubscriptions() error = %v", err)
	}

	templateRecord, err := AddTemplate(models.Template{
		Name: "main",
		Content: strings.TrimSpace(`
listeners:
  - name: USASocks
    type: socks
    port: 10809
    proxy: 🇺🇸 USA
proxy-providers:
  old:
    type: http
    url: https://invalid.example.com
proxy-groups:
  - name: 🚩 PROXY
    type: select
    proxies:
      - ♻️ 自动选择
rules:
  - MATCH,🚩 PROXY
`) + "\n",
		UpdatedAt: time.Now(),
		IsDefault: true,
	}, filepath.Join(dataDir, "templates.json"))
	if err != nil {
		t.Fatalf("AddTemplate() error = %v", err)
	}

	handler := NewHandler(&Config{
		DataDir:         dataDir,
		MaxFileSize:     4096,
		DownloadTimeout: 0,
		RateLimit:       10,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/templates/"+templateRecord.ID+"/render", nil)
	req = mux.SetURLVars(req, map[string]string{"id": templateRecord.ID})
	rec := httptest.NewRecorder()

	handler.RenderTemplateHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "https://invalid.example.com") {
		t.Fatalf("body should replace template proxy-providers, got %s", body)
	}
	if !strings.Contains(body, "proxy-providers:") {
		t.Fatalf("body missing proxy-providers section: %s", body)
	}
	if !strings.Contains(body, "/download/sub-west") {
		t.Fatalf("body missing generated download url: %s", body)
	}
	if !strings.Contains(body, "additional-prefix: \"West |\"") {
		t.Fatalf("body missing sanitized additional-prefix: %s", body)
	}
	if !strings.Contains(body, "path: ./proxies/west-source.yaml") {
		t.Fatalf("body missing current file path: %s", body)
	}
	if !strings.Contains(body, `filter: "(?i)港|hk|hongkong|hong kong"`) {
		t.Fatalf("body missing provider filter: %s", body)
	}
	if !strings.Contains(body, "\"AI Select |\"") {
		t.Fatalf("body missing second provider prefix: %s", body)
	}
	if strings.Contains(body, "ai-source.yaml\n    filter:") {
		t.Fatalf("body should omit empty filter field: %s", body)
	}
}

func TestRenderTemplateHandlerKeepsEmojiAsUTF8Characters(t *testing.T) {
	dataDir := t.TempDir()

	templateRecord, err := AddTemplate(models.Template{
		Name: "emoji",
		Content: strings.TrimSpace(`
proxy-groups:
  - {name: 🇺🇸 USA, type: select, proxies: [DIRECT]}
rules:
  - MATCH,DIRECT
`) + "\n",
		UpdatedAt: time.Now(),
		IsDefault: true,
	}, filepath.Join(dataDir, "templates.json"))
	if err != nil {
		t.Fatalf("AddTemplate() error = %v", err)
	}

	handler := NewHandler(&Config{
		DataDir:         dataDir,
		MaxFileSize:     4096,
		DownloadTimeout: 0,
		RateLimit:       10,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/templates/"+templateRecord.ID+"/render", nil)
	req = mux.SetURLVars(req, map[string]string{"id": templateRecord.ID})
	rec := httptest.NewRecorder()

	handler.RenderTemplateHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, `\U0001F1FA\U0001F1F8`) {
		t.Fatalf("body should contain emoji characters instead of unicode escapes: %s", body)
	}
	if !strings.Contains(body, "name: 🇺🇸 USA") {
		t.Fatalf("body should preserve utf-8 emoji characters: %s", body)
	}
}

func TestRenderTemplateHandlerPreservesNonProviderTemplateText(t *testing.T) {
	dataDir := t.TempDir()
	if err := SaveSubscriptions([]models.Subscription{
		{
			ID:        "sub-west",
			Name:      "🛫 West",
			URL:       "https://example.com/west",
			FilePath:  "west-source.yaml",
			Type:      "clash",
			UpdatedAt: time.Now(),
			Status:    "active",
		},
	}, filepath.Join(dataDir, "subscriptions.json")); err != nil {
		t.Fatalf("SaveSubscriptions() error = %v", err)
	}

	templateContent := strings.TrimSpace(`
proxy-groups:
  - {
      name: 🇺🇸 USA,
      type: fallback,
      lazy:false,
      use: [🛫 West],
      filter: "美国|United States",
    }
rules:
  - GEOSITE,anthropic,🤖 人工智能
  - GEOSITE,category-ai-chat-!cn,🤖 人工智能
  - GEOIP,google,🚩 PROXY
  - MATCH,🚩 PROXY
proxy-providers:
  stale:
    type: http
`) + "\n"

	templateRecord, err := AddTemplate(models.Template{
		Name:      "preserve",
		Content:   templateContent,
		UpdatedAt: time.Now(),
		IsDefault: true,
	}, filepath.Join(dataDir, "templates.json"))
	if err != nil {
		t.Fatalf("AddTemplate() error = %v", err)
	}

	handler := NewHandler(&Config{
		DataDir:         dataDir,
		MaxFileSize:     4096,
		DownloadTimeout: 0,
		RateLimit:       10,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/templates/"+templateRecord.ID+"/render", nil)
	req = mux.SetURLVars(req, map[string]string{"id": templateRecord.ID})
	rec := httptest.NewRecorder()

	handler.RenderTemplateHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	body := rec.Body.String()
	if !strings.Contains(body, "lazy:false,") {
		t.Fatalf("body should preserve raw template flow map text: %s", body)
	}
	if strings.Contains(body, `'lazy:false': ''`) {
		t.Fatalf("body should not rewrite lazy:false into invalid mapping: %s", body)
	}
	if !strings.Contains(body, "- GEOSITE,anthropic,🤖 人工智能") {
		t.Fatalf("body should preserve raw rule text without forced quotes: %s", body)
	}
	if !strings.Contains(body, "- MATCH,🚩 PROXY") {
		t.Fatalf("body should preserve raw match rule text without forced quotes: %s", body)
	}
	if strings.Contains(body, "stale:") {
		t.Fatalf("body should replace stale proxy-providers section: %s", body)
	}
}

func TestListTemplatesHandlerReturnsSavedTemplates(t *testing.T) {
	dataDir := t.TempDir()
	if _, err := AddTemplate(models.Template{
		Name:      "demo",
		Content:   "port: 7890\n",
		UpdatedAt: time.Now(),
		IsDefault: true,
	}, filepath.Join(dataDir, "templates.json")); err != nil {
		t.Fatalf("AddTemplate() error = %v", err)
	}

	handler := NewHandler(&Config{
		DataDir:         dataDir,
		MaxFileSize:     1024,
		DownloadTimeout: 0,
		RateLimit:       10,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/templates", nil)
	rec := httptest.NewRecorder()

	handler.ListTemplatesHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	items, ok := resp.Data.([]any)
	if !ok {
		t.Fatalf("response data type = %T, want array", resp.Data)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
}
