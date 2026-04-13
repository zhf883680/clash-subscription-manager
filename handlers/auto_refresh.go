package handlers

import (
	"context"
	"log"
	"path/filepath"
	"sync"
	"time"
)

const (
	defaultAutoRefreshInterval = 24 * time.Hour
	defaultAutoRefreshWorkers  = 5
)

type refreshTicker interface {
	Chan() <-chan time.Time
	Stop()
}

type realRefreshTicker struct {
	ticker *time.Ticker
}

func (t *realRefreshTicker) Chan() <-chan time.Time {
	return t.ticker.C
}

func (t *realRefreshTicker) Stop() {
	t.ticker.Stop()
}

type AutoRefresher struct {
	handler     *Handler
	dataFile    string
	interval    time.Duration
	workerCount int
	newTicker   func(time.Duration) refreshTicker

	startOnce sync.Once
	stopOnce  sync.Once
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

func NewAutoRefresher(handler *Handler) *AutoRefresher {
	return newAutoRefresher(handler, defaultAutoRefreshInterval, defaultAutoRefreshWorkers, func(d time.Duration) refreshTicker {
		return &realRefreshTicker{ticker: time.NewTicker(d)}
	})
}

func newAutoRefresher(handler *Handler, interval time.Duration, workerCount int, newTicker func(time.Duration) refreshTicker) *AutoRefresher {
	if interval <= 0 {
		interval = defaultAutoRefreshInterval
	}
	if workerCount <= 0 {
		workerCount = 1
	}
	if newTicker == nil {
		newTicker = func(d time.Duration) refreshTicker {
			return &realRefreshTicker{ticker: time.NewTicker(d)}
		}
	}

	return &AutoRefresher{
		handler:     handler,
		dataFile:    filepath.Join(handler.config.DataDir, "subscriptions.json"),
		interval:    interval,
		workerCount: workerCount,
		newTicker:   newTicker,
	}
}

func (a *AutoRefresher) Start() {
	if a == nil {
		return
	}

	a.startOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		a.cancel = cancel

		ticker := a.newTicker(a.interval)
		a.wg.Add(1)
		go func() {
			defer a.wg.Done()
			defer ticker.Stop()

			a.refreshAllWithContext(ctx)

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.Chan():
					a.refreshAllWithContext(ctx)
				}
			}
		}()
	})
}

func (a *AutoRefresher) Stop() {
	if a == nil {
		return
	}

	a.stopOnce.Do(func() {
		if a.cancel != nil {
			a.cancel()
		}
		a.wg.Wait()
	})
}

func (a *AutoRefresher) refreshAll() {
	a.refreshAllWithContext(context.Background())
}

func (a *AutoRefresher) refreshAllWithContext(ctx context.Context) {
	subscriptions, err := ListSubscriptions(a.dataFile)
	if err != nil {
		log.Printf("auto refresh: load subscriptions: %v", err)
		return
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, a.workerCount)

loop:
	for _, sub := range subscriptions {
		select {
		case <-ctx.Done():
			break loop
		case sem <- struct{}{}:
		}

		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			defer func() {
				<-sem
			}()

			if _, refreshErr := a.handler.refreshSubscription(id, a.dataFile, subscriptionPayload{}); refreshErr != nil {
				log.Printf("auto refresh: subscription %s: %v", id, refreshErr)
			}
		}(sub.ID)
	}

	wg.Wait()
}
