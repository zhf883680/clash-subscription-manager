package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"clash-subscription-manager/handlers"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// Version is the application version, set via ldflags during build
var Version = "v1.0.2"

type Config struct {
	Port              int           `yaml:"port"`
	DataDir           string        `yaml:"data_dir"`
	MaxFileSize       int64         `yaml:"max_file_size"`
	DownloadTimeout   time.Duration `yaml:"download_timeout"`
	RateLimit         int           `yaml:"rate_limit"`
	Token             string        `yaml:"token"`
	HTTPS             bool          `yaml:"https"`
	BackupEnabled     bool          `yaml:"backup_enabled"`
	BackupInterval    time.Duration `yaml:"backup_interval"`
	FileRetentionDays int           `yaml:"file_retention_days"`
}

var logger = logrus.New()

func main() {
	cfg, err := loadConfig("config.yaml")
	if err != nil {
		logger.Fatalf("failed to load config: %v", err)
	}

	server := newServer(cfg)

	logStartup(cfg)
	if cfg.HTTPS {
		logger.Info("HTTPS enabled")
		if err := server.ListenAndServeTLS("cert.pem", "key.pem"); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("failed to start HTTPS server: %v", err)
		}
		return
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("failed to start server: %v", err)
	}
}

func logStartup(cfg Config) {
	logger.Infof("starting clash-subscription-manager %s on port %d", Version, cfg.Port)
}

func defaultConfig() Config {
	return Config{
		Port:              8080,
		DataDir:           "./data",
		MaxFileSize:       50 * 1024 * 1024,
		DownloadTimeout:   30 * time.Second,
		RateLimit:         60,
		Token:             "your-secret-token",
		HTTPS:             false,
		BackupEnabled:     true,
		BackupInterval:    24 * time.Hour,
		FileRetentionDays: 30,
	}
}

func loadConfig(path string) (Config, error) {
	cfg := defaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	if len(data) == 0 {
		return cfg, nil
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}

func newServer(cfg Config) *http.Server {
	return &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      newRouter(newHandlerConfig(cfg)),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

func newRouter(cfg *handlers.Config) http.Handler {
	h := handlers.NewHandler(cfg)
	router := mux.NewRouter()

	router.HandleFunc("/", h.HomeHandler).Methods(http.MethodGet)

	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/subscribe", h.SubscribeHandler).Methods(http.MethodPost)
	api.HandleFunc("/subscriptions", h.ListSubscriptionsHandler).Methods(http.MethodGet)
	api.HandleFunc("/subscribe/{id}", h.SubscriptionHandler).Methods(http.MethodGet, http.MethodDelete)
	api.HandleFunc("/subscribe/{id}/refresh", h.RefreshSubscriptionHandler).Methods(http.MethodPost)
	api.HandleFunc("/templates", h.ListTemplatesHandler).Methods(http.MethodGet)
	api.HandleFunc("/templates", h.TemplatesHandler).Methods(http.MethodPost)
	api.HandleFunc("/templates/default/render", h.RenderDefaultTemplateHandler).Methods(http.MethodGet)
	api.HandleFunc("/templates/default/render-proxies", h.RenderDefaultTemplateProxiesHandler).Methods(http.MethodGet)
	api.HandleFunc("/templates/{id}", h.TemplateHandler).Methods(http.MethodGet, http.MethodPut, http.MethodDelete)
	api.HandleFunc("/templates/{id}/default", h.SetDefaultTemplateHandler).Methods(http.MethodPost)
	api.HandleFunc("/templates/{id}/render", h.RenderTemplateHandler).Methods(http.MethodGet)
	api.HandleFunc("/templates/{id}/render-proxies", h.RenderTemplateProxiesHandler).Methods(http.MethodGet)

	router.HandleFunc("/download/{id}", h.RateLimit(h.DownloadHandler)).Methods(http.MethodGet)
	router.HandleFunc("/health", h.HealthHandler).Methods(http.MethodGet)

	fs := http.FileServer(http.Dir("static"))
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	return router
}

func newHandlerConfig(cfg Config) *handlers.Config {
	return &handlers.Config{
		DataDir:         cfg.DataDir,
		MaxFileSize:     cfg.MaxFileSize,
		DownloadTimeout: cfg.DownloadTimeout,
		RateLimit:       cfg.RateLimit,
	}
}
