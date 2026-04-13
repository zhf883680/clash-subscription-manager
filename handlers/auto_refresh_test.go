package handlers

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"clash-subscription-manager/models"
)

func TestAutoRefresherStartTriggersImmediateRefresh(t *testing.T) {
	dataDir := t.TempDir()
	dataFile := filepath.Join(dataDir, "subscriptions.json")
	err := SaveSubscriptions([]models.Subscription{
		{
			ID:       "sub-1",
			Name:     "demo",
			URL:      "https://example.com/sub-1",
			Type:     "clash",
			FilePath: "sub-1.yaml",
			FileSize: int64(len("old-content")),
			Status:   "active",
		},
	}, dataFile)
	if err != nil {
		t.Fatalf("SaveSubscriptions() error = %v", err)
	}
	writeHandlerTestFile(t, filepath.Join(dataDir, "sub-1.yaml"), []byte("old-content"))

	handler := NewHandler(&Config{
		DataDir:         dataDir,
		MaxFileSize:     4096,
		DownloadTimeout: 0,
	})
	handler.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("ss://YWVzLTI1Ni1nY206cGFzcw==@example.com:443#demo")),
			}, nil
		}),
	}

	refresher := newAutoRefresher(handler, 24*time.Hour, 1, func(d time.Duration) refreshTicker {
		return &stubTicker{ch: make(chan time.Time)}
	})
	refresher.Start()
	defer refresher.Stop()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		content, readErr := os.ReadFile(filepath.Join(dataDir, "sub-1.yaml"))
		if readErr == nil && string(content) != "old-content" {
			sub, getErr := GetSubscription("sub-1", dataFile)
			if getErr != nil {
				t.Fatalf("GetSubscription() error = %v", getErr)
			}
			if sub.LastError != "" {
				t.Fatalf("last error = %q, want empty", sub.LastError)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("auto refresher did not run immediate refresh before timeout")
}

func TestAutoRefresherRefreshAllLimitsConcurrency(t *testing.T) {
	dataDir := t.TempDir()
	dataFile := filepath.Join(dataDir, "subscriptions.json")

	subs := make([]models.Subscription, 0, 10)
	for i := 0; i < 10; i++ {
		fileName := filepath.Base(filepath.Join(dataDir, "sub-"+string(rune('a'+i))+".yaml"))
		subs = append(subs, models.Subscription{
			ID:       filepath.Base(fileName),
			Name:     "sub",
			URL:      "https://example.com/" + fileName,
			Type:     "clash",
			FilePath: fileName,
			FileSize: int64(len("old-content")),
			Status:   "active",
		})
		writeHandlerTestFile(t, filepath.Join(dataDir, fileName), []byte("old-content"))
	}
	if err := SaveSubscriptions(subs, dataFile); err != nil {
		t.Fatalf("SaveSubscriptions() error = %v", err)
	}

	handler := NewHandler(&Config{
		DataDir:         dataDir,
		MaxFileSize:     4096,
		DownloadTimeout: 0,
	})

	var current int32
	var maxSeen int32
	started := make(chan struct{}, len(subs))
	release := make(chan struct{})
	handler.httpClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			active := atomic.AddInt32(&current, 1)
			defer atomic.AddInt32(&current, -1)

			for {
				seen := atomic.LoadInt32(&maxSeen)
				if active <= seen || atomic.CompareAndSwapInt32(&maxSeen, seen, active) {
					break
				}
			}

			started <- struct{}{}
			<-release

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("ss://YWVzLTI1Ni1nY206cGFzcw==@example.com:443#demo")),
			}, nil
		}),
	}

	refresher := newAutoRefresher(handler, 24*time.Hour, 5, func(d time.Duration) refreshTicker {
		return &stubTicker{ch: make(chan time.Time)}
	})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		refresher.refreshAll()
	}()

	for i := 0; i < 5; i++ {
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for first refresh workers to start")
		}
	}

	time.Sleep(50 * time.Millisecond)
	if got := atomic.LoadInt32(&maxSeen); got != 5 {
		t.Fatalf("max concurrency = %d, want 5", got)
	}

	close(release)
	wg.Wait()
}

type stubTicker struct {
	ch chan time.Time
}

func (t *stubTicker) Chan() <-chan time.Time {
	return t.ch
}

func (t *stubTicker) Stop() {}
