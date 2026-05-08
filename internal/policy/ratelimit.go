package policy

import (
	"sync"
	"time"
)

type bucket struct {
	tokens           float64
	capacity         float64
	refillRate       float64
	lastRefill       time.Time
	dailyCount       int
	dailyWindowStart time.Time
	maxPerDay        int
	mu               sync.Mutex
}

type AgentRateLimiter struct {
	buckets map[string]*bucket
	mu      sync.RWMutex
}

func NewAgentRateLimiter() *AgentRateLimiter {
	return &AgentRateLimiter{
		buckets: make(map[string]*bucket),
	}
}

func (rl *AgentRateLimiter) Allow(agentID string) bool {
	rl.mu.RLock()
	b, ok := rl.buckets[agentID]
	rl.mu.RUnlock()

	if !ok {
		return true
	}

	return b.allow()
}

func (rl *AgentRateLimiter) SetLimits(agentID string, hour, day int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	refillRate := float64(hour) / 3600.0

	rl.buckets[agentID] = &bucket{
		tokens:           float64(hour),
		capacity:         float64(hour),
		refillRate:       refillRate,
		lastRefill:       time.Now(),
		maxPerDay:        day,
		dailyWindowStart: time.Now(),
	}
}

func (rl *AgentRateLimiter) Cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.buckets = make(map[string]*bucket)
}

func (b *bucket) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()

	elapsed := now.Sub(b.lastRefill).Seconds()
	if elapsed > 0 {
		b.tokens = min(b.capacity, b.tokens+elapsed*b.refillRate)
		b.lastRefill = now
	}

	if b.maxPerDay > 0 {
		if now.Sub(b.dailyWindowStart) >= 24*time.Hour {
			b.dailyCount = 0
			b.dailyWindowStart = now
		}
		if b.dailyCount >= b.maxPerDay {
			return false
		}
	}

	if b.tokens < 1 {
		return false
	}

	b.tokens--
	if b.maxPerDay > 0 {
		b.dailyCount++
	}
	return true
}
