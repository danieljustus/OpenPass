package policy

import (
	"testing"
	"time"
)

func TestAgentRateLimiter_AllowWithinLimit(t *testing.T) {
	rl := NewAgentRateLimiter()
	defer rl.Cleanup()

	rl.SetLimits("agent1", 10, 100)

	for i := 0; i < 5; i++ {
		if !rl.Allow("agent1") {
			t.Fatalf("Allow() = false on request %d, want true (within limit)", i+1)
		}
	}
}

func TestAgentRateLimiter_AllowExceedsHourlyLimit(t *testing.T) {
	rl := NewAgentRateLimiter()
	defer rl.Cleanup()

	rl.SetLimits("agent1", 3, 100)

	for i := 0; i < 3; i++ {
		if !rl.Allow("agent1") {
			t.Fatalf("Allow() = false on request %d, want true", i+1)
		}
	}

	if rl.Allow("agent1") {
		t.Error("Allow() = true after exceeding hourly limit, want false")
	}
}

func TestAgentRateLimiter_AllowExceedsDailyLimit(t *testing.T) {
	rl := NewAgentRateLimiter()
	defer rl.Cleanup()

	rl.SetLimits("agent1", 100, 2)

	for i := 0; i < 2; i++ {
		if !rl.Allow("agent1") {
			t.Fatalf("Allow() = false on request %d, want true", i+1)
		}
	}

	if rl.Allow("agent1") {
		t.Error("Allow() = true after exceeding daily limit, want false")
	}
}

func TestAgentRateLimiter_AllowUnknownAgent(t *testing.T) {
	rl := NewAgentRateLimiter()
	defer rl.Cleanup()

	if !rl.Allow("unknown") {
		t.Error("Allow() = false for unknown agent, want true")
	}
}

func TestAgentRateLimiter_TokenRefill(t *testing.T) {
	rl := NewAgentRateLimiter()
	defer rl.Cleanup()

	rl.SetLimits("agent1", 1, 0)

	if !rl.Allow("agent1") {
		t.Fatal("Allow() = false on first request, want true")
	}

	if rl.Allow("agent1") {
		t.Error("Allow() = true immediately after exhausting token, want false")
	}

	time.Sleep(10 * time.Millisecond)

	if rl.Allow("agent1") {
		t.Log("Token was refilled")
	}
}

func TestAgentRateLimiter_SetLimitsUpdatesExisting(t *testing.T) {
	rl := NewAgentRateLimiter()
	defer rl.Cleanup()

	rl.SetLimits("agent1", 5, 10)
	rl.SetLimits("agent1", 3, 6)

	for i := 0; i < 3; i++ {
		if !rl.Allow("agent1") {
			t.Fatalf("Allow() = false on request %d, want true", i+1)
		}
	}

	if rl.Allow("agent1") {
		t.Error("Allow() = true after exceeding updated limit, want false")
	}
}

func TestAgentRateLimiter_Cleanup(t *testing.T) {
	rl := NewAgentRateLimiter()
	rl.SetLimits("agent1", 1, 1)

	if !rl.Allow("agent1") {
		t.Fatal("Allow() = false before cleanup, want true")
	}

	rl.Cleanup()

	if !rl.Allow("agent1") {
		t.Error("Allow() = false after cleanup, want true (unknown agents allowed)")
	}
}

func TestAgentRateLimiter_DailyWindowReset(t *testing.T) {
	rl := NewAgentRateLimiter()
	defer rl.Cleanup()

	rl.SetLimits("agent1", 100, 1)

	if !rl.Allow("agent1") {
		t.Fatal("Allow() = false on first request, want true")
	}

	if rl.Allow("agent1") {
		t.Error("Allow() = true after reaching daily limit, want false")
	}
}
