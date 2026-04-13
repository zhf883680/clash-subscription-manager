package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"clash-subscription-manager/handlers"

	"github.com/sirupsen/logrus"
)

func TestLoadConfigUsesDefaultsWhenFileEmpty(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")

	cfg, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	if cfg.Port != 8080 {
		t.Fatalf("cfg.Port = %d, want 8080", cfg.Port)
	}
	if cfg.DataDir != "./data" {
		t.Fatalf("cfg.DataDir = %q, want ./data", cfg.DataDir)
	}
	if cfg.DownloadTimeout != 30*time.Second {
		t.Fatalf("cfg.DownloadTimeout = %v, want %v", cfg.DownloadTimeout, 30*time.Second)
	}
}

func TestLoadConfigReadsYAMLFile(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, configPath, []byte("port: 9090\ndata_dir: ./custom-data\nrate_limit: 12\ndownload_timeout: 45s\ntoken: test-token\n"))

	cfg, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	if cfg.Port != 9090 {
		t.Fatalf("cfg.Port = %d, want 9090", cfg.Port)
	}
	if cfg.DataDir != "./custom-data" {
		t.Fatalf("cfg.DataDir = %q, want ./custom-data", cfg.DataDir)
	}
	if cfg.RateLimit != 12 {
		t.Fatalf("cfg.RateLimit = %d, want 12", cfg.RateLimit)
	}
	if cfg.Token != "test-token" {
		t.Fatalf("cfg.Token = %q, want test-token", cfg.Token)
	}
	if cfg.DownloadTimeout != 45*time.Second {
		t.Fatalf("cfg.DownloadTimeout = %v, want %v", cfg.DownloadTimeout, 45*time.Second)
	}
}

func TestNewRouterServesHealthEndpoint(t *testing.T) {
	cfg := defaultConfig()
	router := newRouter(handlers.NewHandler(newHandlerConfig(cfg)))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestNewRouterAllowsSubscriptionsAPIWithoutAuthentication(t *testing.T) {
	cfg := defaultConfig()
	cfg.DataDir = t.TempDir()
	router := newRouter(handlers.NewHandler(newHandlerConfig(cfg)))

	req := httptest.NewRequest(http.MethodGet, "/api/subscriptions", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"success":true`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestLogStartupIncludesVersionAndPort(t *testing.T) {
	var buffer bytes.Buffer
	originalOut := logger.Out
	originalFormatter := logger.Formatter
	originalLevel := logger.Level
	t.Cleanup(func() {
		logger.SetOutput(originalOut)
		logger.SetFormatter(originalFormatter)
		logger.SetLevel(originalLevel)
	})

	logger.SetOutput(&buffer)
	logger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
		DisableColors:    true,
	})
	logger.SetLevel(logrus.InfoLevel)

	logStartup(defaultConfig())

	output := buffer.String()
	if !strings.Contains(output, Version) {
		t.Fatalf("startup log = %q, want version %q", output, Version)
	}
	if !strings.Contains(output, "8080") {
		t.Fatalf("startup log = %q, want port %q", output, "8080")
	}
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
}
