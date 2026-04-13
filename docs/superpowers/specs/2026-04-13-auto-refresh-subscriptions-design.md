# Auto Refresh Subscriptions Design Document

**Date:** 2026-04-13
**Status:** Draft
**Author:** Claude

## Overview

Add automatic subscription refresh functionality that runs every 24 hours with concurrent control and error display in the UI.

## Requirements

### Functional Requirements

1. **Auto Refresh**: Automatically refresh all subscriptions every 24 hours
2. **Concurrent Control**: Maximum 5 subscriptions refreshing simultaneously
3. **Error Handling**:
   - Display refresh failures in the UI without affecting existing subscriptions
   - Keep original cached files when refresh fails
   - Record error information for display
4. **Startup Behavior**: Start first refresh immediately on service startup, then every 24 hours

### Non-Functional Requirements

1. **No Breaking Changes**: Existing subscriptions and cached files remain intact
2. **Performance**: Minimal resource usage with worker pool pattern
3. **Reliability**: Service restart should trigger immediate refresh to maintain sync

## Architecture

### Components

```
┌─────────────────────────────────────────────────────────────┐
│                         Main Process                         │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐     ┌─────────────────────────────────┐  │
│  │   Handler    │────▶│       AutoRefresher             │  │
│  │              │     │  ┌─────────────────────────────┐│  │
│  └──────────────┘     │  │   Ticker (24h)              ││  │
│                       │  └──────────┬──────────────────┘│  │
│                       │             │                     │  │
│                       │  ┌──────────▼──────────────────┐│  │
│                       │  │   refreshAll()              ││  │
│                       │  └──────────┬──────────────────┘│  │
│                       │             │                     │  │
│                       │  ┌──────────▼──────────────────┐│  │
│                       │  │   Worker Channel (buffer)   ││  │
│                       │  └──────────┬──────────────────┘│  │
│                       │             │                     │  │
│                       │  ┌──────────▼──────────────────┐│  │
│                       │  │  Worker 1  │  Worker 2  │ ...││  │
│                       │  │  (refreshSubscription)     ││  │
│                       │  └─────────────────────────────┘│  │
│                       └─────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Data Flow

```
Service Start
     │
     ▼
Create AutoRefresher
     │
     ▼
Start Workers (5 goroutines)
     │
     ▼
Immediate refreshAll()
     │
     ├─▶ Load all subscriptions
     │
     ├─▶ Send to worker channel
     │
     └─▶ Workers process concurrently
          │
          ├─▶ Download content
          │
          ├─▶ Success?
          │   │
          │   ├─ Yes: Update file, clear error fields
          │   │
          │   └─ No: Keep old file, update error fields
          │
          └─▶ Next tick in 24 hours
```

## Data Model Changes

### Subscription Model (`models/subscription.go`)

Add two new optional fields:

```go
type Subscription struct {
    // ... existing fields ...

    // New fields for error tracking
    LastError     string    `json:"last_error,omitempty"`      // Last refresh error message
    LastErrorTime time.Time `json:"last_error_time,omitempty"` // When the error occurred
}
```

**Field Behavior:**
- `LastError`: Contains error message (e.g., "download failed with status: 404")
  - Cleared on successful refresh
  - Set on refresh failure
- `LastErrorTime`: Timestamp of last error
  - Used for UI time calculations ("X hours ago")
  - Cleared on successful refresh

**Backward Compatibility:**
- Fields are `omitempty`, existing subscriptions.json files work without migration
- Old subscriptions will have empty values for new fields

## Implementation Components

### 1. AutoRefresher Component (`handlers/auto_refresh.go`)

**Purpose:** Manages automatic subscription refresh with worker pool

**Key Functions:**
- `NewAutoRefresher()`: Creates instance with 5 workers
- `Start()`: Starts workers and 24-hour ticker, triggers immediate first refresh
- `Stop()`: Gracefully shuts down workers
- `refreshAll()`: Loads all subscriptions and sends to worker queue
- `refreshSubscription()`: Downloads and updates single subscription
- `updateError()`: Updates error fields only (preserves FilePath)
- `updateSuccess()`: Updates file and clears error fields

**Concurrency Model:**
- 5 worker goroutines process subscriptions from buffered channel
- Context-based cancellation for graceful shutdown
- WaitGroup ensures all workers finish before shutdown

### 2. Main Integration (`main.go`)

**Changes:**
1. Create `AutoRefresher` instance after Handler initialization
2. Call `Start()` before server begins listening
3. Add signal handler for graceful shutdown (SIGINT, SIGTERM)
4. Call `Stop()` in shutdown handler

### 3. Frontend Changes

**Status Column Display:**

Time formatting:
- `< 1 minute`: "刚刚"
- `< 1 hour`: "X分钟前"
- `< 24 hours`: "X小时前"
- `>= 24 hours`: "X天前"

Status display:
- **Success**: "最后更新: X小时前"
- **Failure**: "最后更新: X小时前 (失败: 网络超时)" [red text/icon]

**HTML Structure:**
Add status column to subscription list table
- Calculate time difference from `LastCheck`
- Display error from `LastError` if present
- Use CSS class for error styling

### 4. API Endpoints (Optional Enhancement)

`GET /api/subscriptions/refresh-status`
- Returns: auto-refresh running status, next refresh time
- Purpose: Admin monitoring and debugging

## Error Handling Strategy

### On Refresh Failure

**DO:**
- ✅ Update `LastError` with error message
- ✅ Update `LastErrorTime` with current timestamp
- ✅ Keep `FilePath` unchanged (old file remains)
- ✅ Keep `FileSize` unchanged
- ✅ Log error for monitoring

**DON'T:**
- ❌ Delete subscription
- ❌ Delete cached file
- ❌ Modify `FilePath` or `FileSize`
- ❌ Change `Status` from "active"

### Error Types Handled

1. **Network Errors**: Timeout, DNS failure, connection refused
2. **HTTP Errors**: Non-200 status codes (404, 403, 500, etc.)
3. **Content Errors**: Empty response, exceeds size limit
4. **File System Errors**: Write permissions, disk space

## Configuration

No new configuration required. Auto-refresh is always enabled.

**Current Config (unchanged):**
```yaml
# config.yaml
port: 8080
data_dir: "./data"
download_timeout: 30s
max_file_size: 10485760  # 10MB
rate_limit: 60
```

## Testing Strategy

### Unit Tests

1. **AutoRefresher**:
   - Worker pool correctly processes subscriptions
   - Concurrent updates don't corrupt data
   - Stop() gracefully shuts down all workers

2. **Error Handling**:
   - Failed refresh preserves old file
   - Error fields correctly set/cleared
   - Success clears error fields

### Integration Tests

1. **End-to-End**:
   - Service startup triggers immediate refresh
   - Subsequent refreshes happen every 24 hours
   - Concurrent workers handle 10+ subscriptions

2. **Failure Scenarios**:
   - All subscriptions fail (errors recorded, files preserved)
   - Some subscriptions fail (partial success handled)
   - Worker handles panic/timeout gracefully

### Manual Testing

1. **UI Verification**:
   - Status column displays correctly
   - Error messages shown in red
   - Time calculations accurate

2. **Service Restart**:
   - Immediate refresh on startup
   - No data loss or corruption

## Deployment

### Migration Steps

1. Deploy new binary with updated code
2. Service starts with auto-refresh enabled
3. Existing subscriptions refresh immediately
4. Error fields populate as needed
5. UI shows status for all subscriptions

**No database migration required** - new fields are optional and backward compatible.

### Rollback Plan

If issues occur:
1. Revert to previous binary
2. Existing subscriptions continue working
3. New `LastError` fields ignored by old code (omitempty)

## Performance Considerations

### Resource Usage

- **Memory**: Minimal (5 worker goroutines + buffered channel)
- **Network**: Max 5 concurrent HTTP requests
- **Disk**: Only on successful refresh (overwrites existing files)

### Scalability

- **Current design**: Supports ~100 subscriptions efficiently
- **Bottleneck**: HTTP download time (I/O bound)
- **Optimization path**: If needed, make `maxWorkers` configurable

## Security Considerations

1. **SSRF Protection**: Existing download timeout and size limits apply
2. **Rate Limiting**: Worker pool naturally limits request rate
3. **Data Sanitization**: Error messages logged but not escaped (internal use)

## Future Enhancements

Out of scope for this implementation but possible future improvements:

1. **Configurable Refresh Interval**: Allow per-subscription intervals
2. **Retry Logic**: Automatic retry on transient failures
3. **Notifications**: Email/webhook on persistent failures
4. **Health Check API**: Expose refresh statistics
5. **Manual Trigger**: Admin endpoint to force refresh all
6. **Priority Queue**: Critical subscriptions refresh first

## Glossary

- **Subscription**: A proxy subscription link managed by the application
- **Refresh**: Downloading latest config from subscription URL
- **Worker Pool**: Fixed number of goroutines processing tasks from queue
- **Ticker**: Go's time.Ticker for recurring events
- **LastError**: Error message from most recent failed refresh attempt
