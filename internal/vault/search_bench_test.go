package vault

import (
	"fmt"
	"testing"

	"filippo.io/age"
)

func BenchmarkList_100Entries(b *testing.B) {
	benchmarkList(b, 100)
}

func BenchmarkList_1kEntries(b *testing.B) {
	benchmarkList(b, 1000)
}

func BenchmarkList_10kEntries(b *testing.B) {
	benchmarkList(b, 10000)
}

func BenchmarkFind_100Entries_PathOnly(b *testing.B) {
	benchmarkFindPathOnly(b, 100)
}

func BenchmarkFind_1kEntries_PathOnly(b *testing.B) {
	benchmarkFindPathOnly(b, 1000)
}

func BenchmarkFind_10kEntries_PathOnly(b *testing.B) {
	benchmarkFindPathOnly(b, 10000)
}

func BenchmarkFind_100Entries_FieldSearch(b *testing.B) {
	benchmarkFindFieldSearch(b, 100)
}

func BenchmarkFind_1kEntries_FieldSearch(b *testing.B) {
	benchmarkFindFieldSearch(b, 1000)
}

func BenchmarkFind_10kEntries_FieldSearch(b *testing.B) {
	benchmarkFindFieldSearch(b, 10000)
}

func BenchmarkFind_50kEntries_PathOnly(b *testing.B) {
	benchmarkFindPathOnly(b, 50000)
}

func BenchmarkFind_50kEntries_FieldSearch(b *testing.B) {
	benchmarkFindFieldSearch(b, 50000)
}

func BenchmarkFind_10kEntries_Concurrent(b *testing.B) {
	benchmarkFindConcurrent(b, 10000, 4)
}

func BenchmarkFind_1kEntries_Concurrent(b *testing.B) {
	benchmarkFindConcurrent(b, 1000, 4)
}

func benchmarkList(b *testing.B, numEntries int) {
	vaultDir := b.TempDir()
	identity := generateTestIdentity(b)
	createTestEntries(b, vaultDir, identity, numEntries)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		paths, err := List(vaultDir, "")
		if err != nil {
			b.Fatalf("List failed: %v", err)
		}
		if len(paths) != numEntries {
			b.Fatalf("expected %d entries, got %d", numEntries, len(paths))
		}
	}
}

func benchmarkFindPathOnly(b *testing.B, numEntries int) {
	vaultDir := b.TempDir()
	identity := generateTestIdentity(b)
	createTestEntries(b, vaultDir, identity, numEntries)
	rememberSearchIdentity(identity)

	// Find a path number that exists in all vault sizes (use 50 for 100-entry vaults)
	pathQuery := fmt.Sprintf("entry-%05d", min(50, numEntries-1))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Query that matches path - uses fast path (no decryption)
		matches, err := Find(vaultDir, pathQuery)
		if err != nil {
			b.Fatalf("Find failed: %v", err)
		}
		if len(matches) != 1 {
			b.Fatalf("expected 1 match, got %d", len(matches))
		}
	}
}

func benchmarkFindFieldSearch(b *testing.B, numEntries int) {
	vaultDir := b.TempDir()
	identity := generateTestIdentity(b)
	createTestEntries(b, vaultDir, identity, numEntries)
	rememberSearchIdentity(identity)

	fieldQuery := fmt.Sprintf("secret-password-%05d", min(50, numEntries-1))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		matches, err := Find(vaultDir, fieldQuery)
		if err != nil {
			b.Fatalf("Find failed: %v", err)
		}
		if len(matches) != 1 {
			b.Fatalf("expected 1 match, got %d", len(matches))
		}
	}
}

func benchmarkFindConcurrent(b *testing.B, numEntries int, maxWorkers int) {
	vaultDir := b.TempDir()
	identity := generateTestIdentity(b)
	createTestEntries(b, vaultDir, identity, numEntries)
	rememberSearchIdentity(identity)

	fieldQuery := fmt.Sprintf("secret-password-%05d", min(50, numEntries-1))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		matches, err := FindConcurrent(vaultDir, fieldQuery, maxWorkers)
		if err != nil {
			b.Fatalf("FindConcurrent failed: %v", err)
		}
		if len(matches) != 1 {
			b.Fatalf("expected 1 match, got %d", len(matches))
		}
	}
}

func createTestEntries(b *testing.B, vaultDir string, identity *age.X25519Identity, count int) {
	b.Helper()
	for i := 0; i < count; i++ {
		path := fmt.Sprintf("service-%d/entry-%05d", i/100, i)
		data := map[string]interface{}{
			"username": fmt.Sprintf("user-%05d", i),
			"password": fmt.Sprintf("secret-password-%05d", i),
			"url":      fmt.Sprintf("https://service-%d.example.com", i),
		}
		if err := WriteEntry(vaultDir, path, &Entry{Data: data}, identity); err != nil {
			b.Fatalf("WriteEntry(%s) failed: %v", path, err)
		}
	}
}

func generateTestIdentity(b *testing.B) *age.X25519Identity {
	b.Helper()
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		b.Fatalf("GenerateX25519Identity failed: %v", err)
	}
	return identity
}
