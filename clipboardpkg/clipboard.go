package clipboard

import (
	"sync"
	"time"
)

var (
	mu         sync.Mutex
	text       string
	clearTimer *time.Timer
	cancelChan chan struct{}
)

func WriteAll(s string) error {
	mu.Lock()
	text = s
	mu.Unlock()
	return nil
}

func ReadAll() (string, error) {
	mu.Lock()
	defer mu.Unlock()
	return text, nil
}

func StartAutoClear(duration int, clearFn func(), cancelCh chan struct{}) {
	if duration <= 0 {
		return
	}

	StopAutoClear()

	cancelChan = make(chan struct{})
	timerDuration := time.Duration(duration) * time.Second

	go func() {
		select {
		case <-cancelChan:
			return
		case <-time.After(timerDuration):
			mu.Lock()
			clearTimer = nil
			cancelChan = nil
			mu.Unlock()
			clearFn()
		}
	}()
}

func StopAutoClear() {
	if cancelChan != nil {
		close(cancelChan)
		cancelChan = nil
	}
}

func GetCancelChan() chan struct{} {
	mu.Lock()
	defer mu.Unlock()
	return cancelChan
}

func Countdown(duration int, updateFn func(int), cancelCh chan struct{}) {
	if duration <= 0 || updateFn == nil {
		return
	}

	StopAutoClear()
	cancelChan = make(chan struct{})
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	remaining := duration
	updateFn(remaining)

	for {
		select {
		case <-cancelCh:
			return
		case <-ticker.C:
			remaining--
			if remaining <= 0 {
				updateFn(0)
				mu.Lock()
				cancelChan = nil
				mu.Unlock()
				return
			}
			updateFn(remaining)
		}
	}
}
