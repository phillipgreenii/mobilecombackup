package attachments

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
)

// Benchmark the optimized streaming storage vs traditional approach
func BenchmarkAttachmentStorage_StoreFromReader(b *testing.B) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "bench_test")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	storage := NewDirectoryAttachmentStorage(tmpDir, afero.NewOsFs())

	// Test different file sizes
	sizes := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			// Generate test data
			data := make([]byte, size.size)
			for i := range data {
				data[i] = byte(i % 256)
			}

			// Calculate hash
			hasher := sha256.New()
			hasher.Write(data)
			hash := hex.EncodeToString(hasher.Sum(nil))

			metadata := AttachmentInfo{
				Hash:         hash,
				OriginalName: "bench.bin",
				MimeType:     "application/octet-stream",
				Size:         int64(size.size),
				CreatedAt:    time.Now().UTC(),
			}

			b.ResetTimer()
			b.SetBytes(int64(size.size))

			for i := 0; i < b.N; i++ {
				// Generate unique data for each iteration by appending iteration number
				uniqueSuffix := []byte{byte(i & 0xFF), byte((i >> 8) & 0xFF), byte((i >> 16) & 0xFF), byte((i >> 24) & 0xFF)}
				testData := make([]byte, len(data)+len(uniqueSuffix))
				copy(testData, data)
				copy(testData[len(data):], uniqueSuffix)

				// Calculate correct hash for the unique data
				hasher := sha256.New()
				hasher.Write(testData)
				testHash := hex.EncodeToString(hasher.Sum(nil))

				testMetadata := metadata
				testMetadata.Hash = testHash
				testMetadata.Size = int64(len(testData))
				reader := strings.NewReader(string(testData))

				err := storage.StoreFromReader(testHash, reader, testMetadata)
				if err != nil {
					b.Fatalf("Failed to store attachment: %v", err)
				}
			}
		})
	}
}

// Benchmark traditional byte slice storage for comparison
func BenchmarkAttachmentStorage_Store(b *testing.B) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "bench_test")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	storage := NewDirectoryAttachmentStorage(tmpDir, afero.NewOsFs())

	// Test different file sizes
	sizes := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			// Generate test data
			data := make([]byte, size.size)
			for i := range data {
				data[i] = byte(i % 256)
			}

			// Calculate hash
			hasher := sha256.New()
			hasher.Write(data)
			hash := hex.EncodeToString(hasher.Sum(nil))

			metadata := AttachmentInfo{
				Hash:         hash,
				OriginalName: "bench.bin",
				MimeType:     "application/octet-stream",
				Size:         int64(size.size),
				CreatedAt:    time.Now().UTC(),
			}

			b.ResetTimer()
			b.SetBytes(int64(size.size))

			for i := 0; i < b.N; i++ {
				// Generate unique data for each iteration
				uniqueSuffix := []byte{byte(i & 0xFF), byte((i >> 8) & 0xFF), byte((i >> 16) & 0xFF), byte((i >> 24) & 0xFF)}
				testData := make([]byte, len(data)+len(uniqueSuffix))
				copy(testData, data)
				copy(testData[len(data):], uniqueSuffix)

				// Calculate correct hash for the unique data
				hasher := sha256.New()
				hasher.Write(testData)
				testHash := hex.EncodeToString(hasher.Sum(nil))

				testMetadata := metadata
				testMetadata.Hash = testHash
				testMetadata.Size = int64(len(testData))

				err := storage.Store(testHash, testData, testMetadata)
				if err != nil {
					b.Fatalf("Failed to store attachment: %v", err)
				}
			}
		})
	}
}

// Benchmark memory usage comparison
func BenchmarkAttachmentStorage_MemoryUsage(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping memory benchmark in short mode")
	}

	tmpDir, err := os.MkdirTemp("", "memory_bench")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	storage := NewDirectoryAttachmentStorage(tmpDir, afero.NewOsFs())

	// Large file for memory usage testing
	dataSize := 10 * 1024 * 1024 // 10MB
	data := make([]byte, dataSize)
	for i := range data {
		data[i] = byte(i % 256)
	}

	// Calculate hash
	hasher := sha256.New()
	hasher.Write(data)
	hash := hex.EncodeToString(hasher.Sum(nil))

	metadata := AttachmentInfo{
		Hash:         hash,
		OriginalName: "large.bin",
		MimeType:     "application/octet-stream",
		Size:         int64(dataSize),
		CreatedAt:    time.Now().UTC(),
	}

	b.Run("StreamingStorage", func(b *testing.B) {
		b.SetBytes(int64(dataSize))

		for i := 0; i < b.N; i++ {
			// Generate unique data for each iteration
			uniqueSuffix := []byte{byte(i & 0xFF), byte((i >> 8) & 0xFF), byte((i >> 16) & 0xFF), byte((i >> 24) & 0xFF)}
			testData := make([]byte, len(data)+len(uniqueSuffix))
			copy(testData, data)
			copy(testData[len(data):], uniqueSuffix)

			// Calculate correct hash
			hasher := sha256.New()
			hasher.Write(testData)
			testHash := hex.EncodeToString(hasher.Sum(nil))

			testMetadata := metadata
			testMetadata.Hash = testHash
			testMetadata.Size = int64(len(testData))
			reader := strings.NewReader(string(testData))

			err := storage.StoreFromReader(testHash, reader, testMetadata)
			if err != nil {
				b.Fatalf("Failed to store via streaming: %v", err)
			}
		}
	})
}

// Benchmark concurrent storage operations
func BenchmarkAttachmentStorage_Concurrent(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "concurrent_bench")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	storage := NewDirectoryAttachmentStorage(tmpDir, afero.NewOsFs())

	// Medium-sized data for concurrent testing
	dataSize := 100 * 1024 // 100KB
	data := make([]byte, dataSize)
	for i := range data {
		data[i] = byte(i % 256)
	}

	metadata := AttachmentInfo{
		OriginalName: "concurrent.bin",
		MimeType:     "application/octet-stream",
		Size:         int64(dataSize),
		CreatedAt:    time.Now().UTC(),
	}

	b.SetBytes(int64(dataSize))
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// Generate unique data using both counter and timestamp to avoid collisions
			uniqueBytes := []byte{
				byte(i & 0xFF),
				byte((i >> 8) & 0xFF),
				byte((i >> 16) & 0xFF),
				byte((i >> 24) & 0xFF),
			}
			// Add additional entropy with current nanosecond timestamp
			timestamp := time.Now().UnixNano()
			timestampBytes := []byte{
				byte(timestamp & 0xFF),
				byte((timestamp >> 8) & 0xFF),
				byte((timestamp >> 16) & 0xFF),
				byte((timestamp >> 24) & 0xFF),
			}

			testData := make([]byte, len(data)+len(uniqueBytes)+len(timestampBytes))
			copy(testData, data)
			copy(testData[len(data):], uniqueBytes)
			copy(testData[len(data)+len(uniqueBytes):], timestampBytes)

			// Calculate hash for the unique data
			hasher := sha256.New()
			hasher.Write(testData)
			hash := hex.EncodeToString(hasher.Sum(nil))

			testMetadata := metadata
			testMetadata.Hash = hash
			testMetadata.Size = int64(len(testData))
			reader := strings.NewReader(string(testData))

			err := storage.StoreFromReader(hash, reader, testMetadata)
			if err != nil {
				b.Fatalf("Failed concurrent storage: %v", err)
			}
			i++
		}
	})
}

// Benchmark hash verification overhead
func BenchmarkAttachmentStorage_HashVerification(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "hash_bench")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	storage := NewDirectoryAttachmentStorage(tmpDir, afero.NewOsFs())

	dataSize := 1024 * 1024 // 1MB
	data := make([]byte, dataSize)
	for i := range data {
		data[i] = byte(i % 256)
	}

	// Calculate correct hash
	hasher := sha256.New()
	hasher.Write(data)
	correctHash := hex.EncodeToString(hasher.Sum(nil))

	metadata := AttachmentInfo{
		Hash:         correctHash,
		OriginalName: "hash_test.bin",
		MimeType:     "application/octet-stream",
		Size:         int64(dataSize),
		CreatedAt:    time.Now().UTC(),
	}

	b.SetBytes(int64(dataSize))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Generate unique data for each iteration
		uniqueSuffix := []byte{byte(i & 0xFF), byte((i >> 8) & 0xFF), byte((i >> 16) & 0xFF), byte((i >> 24) & 0xFF)}
		testData := make([]byte, len(data)+len(uniqueSuffix))
		copy(testData, data)
		copy(testData[len(data):], uniqueSuffix)

		// Calculate correct hash for verification test
		hasher := sha256.New()
		hasher.Write(testData)
		testHash := hex.EncodeToString(hasher.Sum(nil))

		testMetadata := metadata
		testMetadata.Hash = testHash
		testMetadata.Size = int64(len(testData))
		reader := strings.NewReader(string(testData))

		// This should succeed with hash verification
		err := storage.StoreFromReader(testHash, reader, testMetadata)
		if err != nil {
			b.Fatalf("Benchmark iteration %d failed: %v", i, err)
		}
	}
}
