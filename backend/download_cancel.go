package backend

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrDownloadCancelled = errors.New("download cancelled")

var downloadCancelState = struct {
	sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
	active   int
	stopping bool
}{}

func ActiveDownloadContext() context.Context {
	downloadCancelState.Lock()
	defer downloadCancelState.Unlock()

	if downloadCancelState.ctx == nil {
		return context.Background()
	}
	return downloadCancelState.ctx
}

func ForceStopActiveDownloads() {
	downloadCancelState.Lock()
	cancel := downloadCancelState.cancel
	if cancel != nil {
		downloadCancelState.stopping = true
	}
	downloadCancelState.Unlock()

	if cancel != nil {
		cancel()
	}

	CancelQueuedAndDownloadingItems()
	SetDownloading(false)
}

func CheckDownloadCancelled() error {
	ctx := ActiveDownloadContext()
	select {
	case <-ctx.Done():
		return ErrDownloadCancelled
	default:
		return nil
	}
}

func SleepWithDownloadContext(delay time.Duration) error {
	if delay <= 0 {
		return CheckDownloadCancelled()
	}

	ctx := ActiveDownloadContext()
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ErrDownloadCancelled
	case <-timer.C:
		return nil
	}
}

func IsDownloadCancelledError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrDownloadCancelled) || errors.Is(err, context.Canceled)
}
