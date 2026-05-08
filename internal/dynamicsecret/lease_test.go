package dynamicsecret

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestLeaseManagerCreate(t *testing.T) {
	lm := NewLeaseManager()
	defer lm.Close()

	secret := &Secret{
		LeaseID:       "lease-1",
		LeaseDuration: time.Hour,
		Renewable:     true,
		Data:          map[string]any{"key": "value"},
		CreatedAt:     time.Now().UTC(),
		EngineType:    EngineTypeMock,
	}

	lease, err := lm.Create(secret)
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if lease == nil {
		t.Fatal("Create returned nil lease")
	}

	if lease.ID == "" {
		t.Error("lease.ID is empty")
	}
	if lease.Secret != secret {
		t.Error("lease.Secret does not match input secret")
	}
	if lease.Revoked {
		t.Error("new lease should not be revoked")
	}
	if !lease.CreatedAt.Before(time.Now().Add(time.Minute)) {
		t.Error("lease.CreatedAt should be recent")
	}
	if !lease.ExpiresAt.After(time.Now()) {
		t.Error("lease.ExpiresAt should be in the future")
	}
}

func TestLeaseManagerGet(t *testing.T) {
	lm := NewLeaseManager()
	defer lm.Close()

	secret := &Secret{LeaseID: "lease-1", LeaseDuration: time.Hour}
	lease, err := lm.Create(secret)
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if lease == nil {
		t.Fatal("Create returned nil lease")
	}

	got, ok := lm.Get(lease.ID)
	if !ok {
		t.Fatalf("Get(%q) = false, want true", lease.ID)
	}
	if got.ID != lease.ID {
		t.Errorf("Get ID = %q, want %q", got.ID, lease.ID)
	}

	_, ok = lm.Get("non-existent")
	if ok {
		t.Error("Get(non-existent) = true, want false")
	}
}

func TestLeaseManagerIsAlive(t *testing.T) {
	lm := NewLeaseManager()
	defer lm.Close()

	secret := &Secret{LeaseID: "lease-1", LeaseDuration: time.Hour}
	lease, err := lm.Create(secret)
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if lease == nil {
		t.Fatal("Create returned nil lease")
	}

	if !lm.IsAlive(lease.ID) {
		t.Error("IsAlive = false for fresh lease, want true")
	}

	if lm.IsAlive("non-existent") {
		t.Error("IsAlive = true for missing lease, want false")
	}

	if err := lm.Revoke(lease.ID); err != nil {
		t.Fatalf("Revoke error: %v", err)
	}

	if lm.IsAlive(lease.ID) {
		t.Error("IsAlive = true for revoked lease, want false")
	}
}

func TestLeaseManagerRevoke(t *testing.T) {
	lm := NewLeaseManager()
	defer lm.Close()

	secret := &Secret{LeaseID: "lease-1", LeaseDuration: time.Hour}
	lease, err := lm.Create(secret)
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if lease == nil {
		t.Fatal("Create returned nil lease")
	}

	if err := lm.Revoke(lease.ID); err != nil {
		t.Fatalf("Revoke error: %v", err)
	}

	got, _ := lm.Get(lease.ID)
	if got != nil && !got.Revoked {
		t.Error("revoked lease should have Revoked=true")
	}

	if err := lm.Revoke("non-existent"); err == nil {
		t.Error("Revoke(non-existent) = nil, want error")
	}
}

func TestLeaseManagerListActive(t *testing.T) {
	lm := NewLeaseManager()
	defer lm.Close()

	secret1 := &Secret{LeaseID: "lease-1", LeaseDuration: time.Hour}
	secret2 := &Secret{LeaseID: "lease-2", LeaseDuration: time.Hour}

	l1, err := lm.Create(secret1)
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if l1 == nil {
		t.Fatal("Create returned nil lease")
	}
	_, err = lm.Create(secret2)
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}

	if err := lm.Revoke(l1.ID); err != nil {
		t.Fatalf("Revoke error: %v", err)
	}

	active := lm.ListActive()
	if len(active) != 1 {
		t.Fatalf("ListActive() = %d, want 1", len(active))
	}
	if active[0].ID == l1.ID {
		t.Error("ListActive should not include revoked lease")
	}
}

func TestLeaseManagerExpiredNotAlive(t *testing.T) {
	lm := NewLeaseManager()
	defer lm.Close()

	secret := &Secret{LeaseID: "lease-1", LeaseDuration: time.Millisecond}
	lease, err := lm.Create(secret)
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if lease == nil {
		t.Fatal("Create returned nil lease")
	}

	time.Sleep(10 * time.Millisecond)

	if lm.IsAlive(lease.ID) {
		t.Error("IsAlive = true for expired lease, want false")
	}

	active := lm.ListActive()
	for _, l := range active {
		if l.ID == lease.ID {
			t.Error("ListActive should not include expired lease")
		}
	}
}

func TestLeaseManagerConcurrentAccess(t *testing.T) {
	lm := NewLeaseManager()
	defer lm.Close()

	var wg sync.WaitGroup
	numWorkers := 10
	numOps := 50

	// Concurrent creates
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				secret := &Secret{LeaseID: "lease", LeaseDuration: time.Hour}
				lm.Create(secret)
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				lm.ListActive()
				lm.IsAlive("any-id")
			}
		}(i)
	}

	wg.Wait()

	// Verify some leases exist
	active := lm.ListActive()
	if len(active) == 0 {
		t.Error("expected some active leases after concurrent creates")
	}
}

func TestLeaseManagerStartCleanup(t *testing.T) {
	lm := NewLeaseManager()
	defer lm.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lm.StartCleanup(ctx, 50*time.Millisecond)

	secret := &Secret{LeaseID: "lease-1", LeaseDuration: time.Millisecond}
	lease, err := lm.Create(secret)
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if lease == nil {
		t.Fatal("Create returned nil lease")
	}

	if !lm.IsAlive(lease.ID) {
		t.Fatal("lease should be alive initially")
	}

	time.Sleep(200 * time.Millisecond)

	if lm.IsAlive(lease.ID) {
		t.Error("expired lease should be cleaned up")
	}
}

func TestLeaseManagerClose(t *testing.T) {
	lm := NewLeaseManager()

	if err := lm.Close(); err != nil {
		t.Errorf("Close error: %v", err)
	}

	// Close should be idempotent
	if err := lm.Close(); err != nil {
		t.Errorf("second Close error: %v", err)
	}
}

func TestLeaseFields(t *testing.T) {
	now := time.Now().UTC()
	lease := Lease{
		ID:        "test-id",
		Secret:    &Secret{LeaseID: "test-id"},
		ExpiresAt: now.Add(time.Hour),
		Revoked:   false,
		CreatedAt: now,
	}

	if lease.ID != "test-id" {
		t.Errorf("ID = %q, want test-id", lease.ID)
	}
	if lease.Secret == nil {
		t.Error("Secret is nil")
	}
	if !lease.ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Error("ExpiresAt mismatch")
	}
	if lease.Revoked {
		t.Error("Revoked = true, want false")
	}
	if !lease.CreatedAt.Equal(now) {
		t.Error("CreatedAt mismatch")
	}
}
