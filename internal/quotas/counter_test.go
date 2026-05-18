package quotas

import (
	"os"
	"sync"
	"testing"
)

func TestIncrementAndCheck(t *testing.T) {
	vaultDir := t.TempDir()
	c, err := New(vaultDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer c.Close()

	ok, current := c.Check("tool_a", 10)
	if !ok {
		t.Errorf("Check before increment: ok=%v, want true", ok)
	}
	if current != 0 {
		t.Errorf("Check before increment: current=%d, want 0", current)
	}

	got, err := c.Increment("tool_a")
	if err != nil {
		t.Fatalf("Increment: %v", err)
	}
	if got != 1 {
		t.Errorf("Increment: got %d, want 1", got)
	}

	ok, current = c.Check("tool_a", 10)
	if !ok {
		t.Errorf("Check after increment: ok=%v, want true", ok)
	}
	if current != 1 {
		t.Errorf("Check after increment: current=%d, want 1", current)
	}

	for i := 2; i <= 5; i++ {
		got, err = c.Increment("tool_a")
		if err != nil {
			t.Fatalf("Increment %d: %v", i, err)
		}
		if got != i {
			t.Errorf("Increment %d: got %d, want %d", i, got, i)
		}
	}

	ok, _ = c.Check("tool_a", 5)
	if ok {
		t.Errorf("Check with limit=5 after 5 increments: ok=true, want false")
	}
	ok, _ = c.Check("tool_a", 6)
	if !ok {
		t.Errorf("Check with limit=6 after 5 increments: ok=false, want true")
	}
}

func TestSeparateToolCounters(t *testing.T) {
	vaultDir := t.TempDir()
	c, err := New(vaultDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer c.Close()

	_, err = c.Increment("tool_a")
	if err != nil {
		t.Fatalf("Increment tool_a: %v", err)
	}
	_, err = c.Increment("tool_b")
	if err != nil {
		t.Fatalf("Increment tool_b: %v", err)
	}
	_, err = c.Increment("tool_a")
	if err != nil {
		t.Fatalf("Increment tool_a again: %v", err)
	}

	_, ca := c.Check("tool_a", 100)
	_, cb := c.Check("tool_b", 100)
	if ca != 2 {
		t.Errorf("tool_a count: %d, want 2", ca)
	}
	if cb != 1 {
		t.Errorf("tool_b count: %d, want 1", cb)
	}
}

func TestReset(t *testing.T) {
	vaultDir := t.TempDir()
	c, err := New(vaultDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer c.Close()

	for i := 0; i < 5; i++ {
		_, _ = c.Increment("tool_a")
	}

	c.Reset()

	_, current := c.Check("tool_a", 100)
	if current != 0 {
		t.Errorf("After reset: current=%d, want 0", current)
	}
}

func TestCheckZeroOrNegativeLimit(t *testing.T) {
	vaultDir := t.TempDir()
	c, err := New(vaultDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer c.Close()

	if ok, _ := c.Check("tool_a", 0); ok {
		t.Error("Check with limit=0: ok=true, want false")
	}
	if ok, _ := c.Check("tool_a", -1); ok {
		t.Error("Check with limit=-1: ok=true, want false")
	}
}

func TestConcurrentIncrements(t *testing.T) {
	vaultDir := t.TempDir()
	c, err := New(vaultDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer c.Close()

	var wg sync.WaitGroup
	n := 20
	wg.Add(n)

	for range n {
		go func() {
			defer wg.Done()
			_, err := c.Increment("conc")
			if err != nil {
				t.Errorf("concurrent increment: %v", err)
			}
		}()
	}

	wg.Wait()

	_, current := c.Check("conc", 100)
	if current != n {
		t.Errorf("%d concurrent increments: current=%d, want %d", n, current, n)
	}
}

func TestConcurrentMultipleTools(t *testing.T) {
	vaultDir := t.TempDir()
	c, err := New(vaultDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer c.Close()

	tools := []string{"alpha", "beta", "gamma"}
	var wg sync.WaitGroup

	for _, tool := range tools {
		wg.Add(1)
		go func(tn string) {
			defer wg.Done()
			for range 5 {
				_, err := c.Increment(tn)
				if err != nil {
					t.Errorf("Increment %s: %v", tn, err)
				}
			}
		}(tool)
	}

	wg.Wait()

	for _, tool := range tools {
		_, count := c.Check(tool, 100)
		if count != 5 {
			t.Errorf("Tool %s: count=%d, want 5", tool, count)
		}
	}
}

func TestPersistenceAcrossNewInstance(t *testing.T) {
	vaultDir := t.TempDir()

	c1, err := New(vaultDir)
	if err != nil {
		t.Fatalf("New c1: %v", err)
	}
	_, err = c1.Increment("persist")
	if err != nil {
		t.Fatalf("c1.Increment: %v", err)
	}
	c1.Close()

	c2, err := New(vaultDir)
	if err != nil {
		t.Fatalf("New c2: %v", err)
	}
	defer c2.Close()

	_, current := c2.Check("persist", 100)
	if current != 1 {
		t.Errorf("After reopen: current=%d, want 1", current)
	}

	got, err := c2.Increment("persist")
	if err != nil {
		t.Fatalf("c2.Increment: %v", err)
	}
	if got != 2 {
		t.Errorf("c2.Increment: %d, want 2", got)
	}
}

func TestNewQuotaFileCreated(t *testing.T) {
	vaultDir := t.TempDir()

	c, err := New(vaultDir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer c.Close()

	_, err = c.Increment("first")
	if err != nil {
		t.Fatalf("Increment: %v", err)
	}

	if _, err := os.Stat(c.filePath); err != nil {
		t.Errorf("quota file not created at %s: %v", c.filePath, err)
	}
}
