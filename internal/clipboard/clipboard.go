// Package clipboard provides timer-based clipboard clearing utilities.
package clipboard

import "time"

// StartAutoClear clears the clipboard after the configured timeout unless canceled.
func StartAutoClear(duration int, clearFn func(), cancelCh <-chan struct{}) {
	if duration <= 0 || clearFn == nil {
		return
	}

	timer := time.NewTimer(time.Duration(duration) * time.Second)
	defer timer.Stop()

	select {
	case <-timer.C:
		clearFn()
	case <-cancelCh:
	}
}

// Countdown reports the remaining seconds until zero unless canceled.
func Countdown(duration int, updateFn func(int), cancelCh <-chan struct{}) {
	if duration <= 0 || updateFn == nil {
		return
	}

	updateFn(duration)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	remaining := duration
	for remaining > 0 {
		select {
		case <-cancelCh:
			return
		case <-ticker.C:
			remaining--
			updateFn(remaining)
		}
	}
}
