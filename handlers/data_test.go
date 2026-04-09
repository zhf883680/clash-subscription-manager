package handlers

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"clash-subscription-manager/models"
)

func TestLoadSubscriptions(t *testing.T) {
	// Create temporary directory for test data
	tempDir := t.TempDir()
	testDataFile := filepath.Join(tempDir, "subscriptions.json")

	// Test 1: Load from non-existent file
	subs, err := LoadSubscriptions(testDataFile)
	if err != nil {
		t.Errorf("LoadSubscriptions() should not return error for non-existent file, got: %v", err)
	}
	if subs == nil {
		t.Error("LoadSubscriptions() should return empty slice, not nil")
	}
	if len(subs) != 0 {
		t.Errorf("LoadSubscriptions() should return empty slice, got %d items", len(subs))
	}

	// Test 2: Load from existing file with data
	// Write test data
	data := `[
		{
			"id": "test1",
			"name": "Test Subscription",
			"url": "http://example.com",
			"type": "v2ray",
			"updated_at": "2024-01-01T00:00:00Z",
			"last_check": "2024-01-01T00:00:00Z",
			"status": "active",
			"node_count": 10,
			"traffic": {
				"total": 0,
				"upload": 0,
				"download": 0
			}
		}
	]`
	if err := os.WriteFile(testDataFile, []byte(data), 0644); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}

	// Load and verify
	subs, err = LoadSubscriptions(testDataFile)
	if err != nil {
		t.Errorf("LoadSubscriptions() returned error: %v", err)
	}
	if len(subs) != 1 {
		t.Errorf("LoadSubscriptions() returned %d items, want 1", len(subs))
	}
	if subs[0].ID != "test1" {
		t.Errorf("LoadSubscriptions() ID = %s, want test1", subs[0].ID)
	}
}

func TestSaveSubscriptions(t *testing.T) {
	tempDir := t.TempDir()
	testDataFile := filepath.Join(tempDir, "subscriptions.json")

	// Test saving subscriptions
	subs := []models.Subscription{
		{
			ID:        "test1",
			Name:      "Test Subscription",
			URL:       "http://example.com",
			Type:      "v2ray",
			UpdatedAt: time.Now(),
			Status:    "active",
			NodeCount: 10,
		},
	}

	err := SaveSubscriptions(subs, testDataFile)
	if err != nil {
		t.Errorf("SaveSubscriptions() returned error: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(testDataFile); os.IsNotExist(err) {
		t.Error("SaveSubscriptions() did not create file")
	}

	// Verify content by loading it back
	loadedSubs, err := LoadSubscriptions(testDataFile)
	if err != nil {
		t.Errorf("Failed to load saved subscriptions: %v", err)
	}
	if len(loadedSubs) != 1 {
		t.Errorf("Loaded %d subscriptions, want 1", len(loadedSubs))
	}
	if loadedSubs[0].ID != "test1" {
		t.Errorf("Loaded ID = %s, want test1", loadedSubs[0].ID)
	}
}

func TestGenerateID(t *testing.T) {
	// Test that generateID creates unique IDs
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateID()
		if id == "" {
			t.Error("generateID() returned empty string")
		}
		if ids[id] {
			t.Errorf("generateID() returned duplicate ID: %s", id)
		}
		ids[id] = true
	}
}

func TestAddSubscription(t *testing.T) {
	tempDir := t.TempDir()
	testDataFile := filepath.Join(tempDir, "subscriptions.json")

	// Create initial subscriptions file
	initialSubs := []models.Subscription{}
	if err := SaveSubscriptions(initialSubs, testDataFile); err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Test adding a subscription
	newSub := models.Subscription{
		Name:      "New Subscription",
		URL:       "http://example.com/new",
		Type:      "v2ray",
		UpdatedAt: time.Now(),
		Status:    "active",
		NodeCount: 5,
	}

	addedSub, err := AddSubscription(newSub, testDataFile)
	if err != nil {
		t.Errorf("AddSubscription() returned error: %v", err)
	}
	if addedSub.ID == "" {
		t.Error("AddSubscription() did not generate ID")
	}

	// Verify subscription was added
	subs, err := LoadSubscriptions(testDataFile)
	if err != nil {
		t.Errorf("Failed to load subscriptions: %v", err)
	}
	if len(subs) != 1 {
		t.Errorf("Expected 1 subscription, got %d", len(subs))
	}
	if subs[0].Name != "New Subscription" {
		t.Errorf("Expected name 'New Subscription', got '%s'", subs[0].Name)
	}
}

func TestDeleteSubscription(t *testing.T) {
	tempDir := t.TempDir()
	testDataFile := filepath.Join(tempDir, "subscriptions.json")
	cachedFile := filepath.Join(tempDir, "cached-subscription.yaml")
	if err := os.WriteFile(cachedFile, []byte("proxy"), 0644); err != nil {
		t.Fatalf("Failed to create cached file: %v", err)
	}

	// Create initial subscriptions
	initialSubs := []models.Subscription{
		{
			ID:        "test1",
			Name:      "Test 1",
			URL:       "http://example.com/1",
			FilePath:  filepath.Base(cachedFile),
			Type:      "v2ray",
			UpdatedAt: time.Now(),
			Status:    "active",
			NodeCount: 10,
		},
		{
			ID:        "test2",
			Name:      "Test 2",
			URL:       "http://example.com/2",
			Type:      "v2ray",
			UpdatedAt: time.Now(),
			Status:    "active",
			NodeCount: 20,
		},
	}
	if err := SaveSubscriptions(initialSubs, testDataFile); err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Test deleting existing subscription
	err := DeleteSubscription("test1", testDataFile)
	if err != nil {
		t.Errorf("DeleteSubscription() returned error: %v", err)
	}

	// Verify deletion
	subs, err := LoadSubscriptions(testDataFile)
	if err != nil {
		t.Errorf("Failed to load subscriptions: %v", err)
	}
	if len(subs) != 1 {
		t.Errorf("Expected 1 subscription after deletion, got %d", len(subs))
	}
	if subs[0].ID != "test2" {
		t.Errorf("Expected remaining subscription ID 'test2', got '%s'", subs[0].ID)
	}
	if _, err := os.Stat(cachedFile); !os.IsNotExist(err) {
		t.Errorf("Expected cached file to be removed, stat err = %v", err)
	}

	// Test deleting non-existent subscription
	err = DeleteSubscription("nonexistent", testDataFile)
	if err == nil {
		t.Error("DeleteSubscription() should return error for non-existent ID")
	}
}

func TestGetSubscription(t *testing.T) {
	tempDir := t.TempDir()
	testDataFile := filepath.Join(tempDir, "subscriptions.json")

	// Create initial subscriptions
	initialSubs := []models.Subscription{
		{
			ID:        "test1",
			Name:      "Test 1",
			URL:       "http://example.com/1",
			Type:      "v2ray",
			UpdatedAt: time.Now(),
			Status:    "active",
			NodeCount: 10,
		},
	}
	if err := SaveSubscriptions(initialSubs, testDataFile); err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Test getting existing subscription
	sub, err := GetSubscription("test1", testDataFile)
	if err != nil {
		t.Errorf("GetSubscription() returned error: %v", err)
	}
	if sub == nil {
		t.Error("GetSubscription() returned nil for existing subscription")
	}
	if sub.ID != "test1" {
		t.Errorf("GetSubscription() returned wrong ID: %s", sub.ID)
	}

	// Test getting non-existent subscription
	sub, err = GetSubscription("nonexistent", testDataFile)
	if err == nil {
		t.Error("GetSubscription() should return error for non-existent ID")
	}
	if sub != nil {
		t.Error("GetSubscription() should return nil for non-existent subscription")
	}
}

func TestListSubscriptions(t *testing.T) {
	tempDir := t.TempDir()
	testDataFile := filepath.Join(tempDir, "subscriptions.json")

	// Test listing empty subscriptions
	subs, err := ListSubscriptions(testDataFile)
	if err != nil {
		t.Errorf("ListSubscriptions() returned error: %v", err)
	}
	if len(subs) != 0 {
		t.Errorf("ListSubscriptions() returned %d items, want 0", len(subs))
	}

	// Add subscriptions and test listing
	initialSubs := []models.Subscription{
		{
			ID:        "test1",
			Name:      "Test 1",
			URL:       "http://example.com/1",
			Type:      "v2ray",
			UpdatedAt: time.Now(),
			Status:    "active",
			NodeCount: 10,
		},
		{
			ID:        "test2",
			Name:      "Test 2",
			URL:       "http://example.com/2",
			Type:      "shadowsocks",
			UpdatedAt: time.Now(),
			Status:    "active",
			NodeCount: 20,
		},
	}
	if err := SaveSubscriptions(initialSubs, testDataFile); err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	subs, err = ListSubscriptions(testDataFile)
	if err != nil {
		t.Errorf("ListSubscriptions() returned error: %v", err)
	}
	if len(subs) != 2 {
		t.Errorf("ListSubscriptions() returned %d items, want 2", len(subs))
	}
}

func TestUpdateSubscription(t *testing.T) {
	tempDir := t.TempDir()
	testDataFile := filepath.Join(tempDir, "subscriptions.json")

	err := SaveSubscriptions([]models.Subscription{
		{
			ID:   "test1",
			Name: "before",
			URL:  "https://example.com/old",
			RequestHeaders: map[string]string{
				"User-Agent": "before",
			},
			FilePath:  "before.yaml",
			FileSize:  10,
			Type:      "clash",
			Status:    "active",
			UpdatedAt: time.Now().Add(-time.Hour),
			LastCheck: time.Now().Add(-time.Hour),
		},
	}, testDataFile)
	if err != nil {
		t.Fatalf("SaveSubscriptions() error = %v", err)
	}

	updatedAt := time.Now()
	updatedSub, err := UpdateSubscription("test1", testDataFile, func(sub *models.Subscription) error {
		sub.Name = "after"
		sub.URL = "https://example.com/new"
		sub.RequestHeaders = map[string]string{
			"X-Test-Header": "token-1",
		}
		sub.FileSize = 99
		sub.UpdatedAt = updatedAt
		sub.LastCheck = updatedAt
		return nil
	})
	if err != nil {
		t.Fatalf("UpdateSubscription() error = %v", err)
	}

	if updatedSub.Name != "after" {
		t.Fatalf("updated name = %q, want %q", updatedSub.Name, "after")
	}

	loaded, err := GetSubscription("test1", testDataFile)
	if err != nil {
		t.Fatalf("GetSubscription() error = %v", err)
	}
	if loaded.URL != "https://example.com/new" {
		t.Fatalf("loaded URL = %q, want %q", loaded.URL, "https://example.com/new")
	}
	if loaded.RequestHeaders["X-Test-Header"] != "token-1" {
		t.Fatalf("request header = %q, want %q", loaded.RequestHeaders["X-Test-Header"], "token-1")
	}
	if loaded.FileSize != 99 {
		t.Fatalf("file size = %d, want %d", loaded.FileSize, 99)
	}
}
