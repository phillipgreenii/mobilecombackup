package attachments_test

import (
	"fmt"
	"log"
	"time"

	"github.com/phillipgreen/mobilecombackup/pkg/attachments"
)

// Example demonstrates basic usage of the AttachmentManager
func ExampleAttachmentManager() {
	// Create a manager for a repository
	manager := attachments.NewAttachmentManager("/path/to/repository")

	// Get attachment by hash
	hash := "3ceb5c413ee02895bf1f357a8c2cc2bec824f4d8aad13aeab69303f341c8b781"
	attachment, err := manager.GetAttachment(hash)
	if err != nil {
		log.Fatal(err)
	}

	if attachment.Exists {
		fmt.Printf("Attachment %s exists, size: %d bytes\n", attachment.Hash, attachment.Size)
	} else {
		fmt.Printf("Attachment %s not found\n", attachment.Hash)
	}
}

// Example demonstrates reading attachment content
func ExampleAttachmentManager_ReadAttachment() {
	manager := attachments.NewAttachmentManager("/path/to/repository")

	hash := "3ceb5c413ee02895bf1f357a8c2cc2bec824f4d8aad13aeab69303f341c8b781"
	content, err := manager.ReadAttachment(hash)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Read %d bytes of content\n", len(content))
	// Process the attachment content as needed
}

// Example demonstrates verifying attachment integrity
func ExampleAttachmentManager_VerifyAttachment() {
	manager := attachments.NewAttachmentManager("/path/to/repository")

	hash := "3ceb5c413ee02895bf1f357a8c2cc2bec824f4d8aad13aeab69303f341c8b781"
	verified, err := manager.VerifyAttachment(hash)
	if err != nil {
		log.Fatal(err)
	}

	if verified {
		fmt.Println("Attachment verification passed")
	} else {
		fmt.Println("Attachment verification failed - content doesn't match hash")
	}
}

// Example demonstrates listing all attachments
func ExampleAttachmentManager_ListAttachments() {
	manager := attachments.NewAttachmentManager("/path/to/repository")

	attachments, err := manager.ListAttachments()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d attachments:\n", len(attachments))
	for _, attachment := range attachments {
		fmt.Printf("  %s (%d bytes)\n", attachment.Hash, attachment.Size)
	}
}

// Example demonstrates streaming attachments for memory efficiency
func ExampleAttachmentManager_StreamAttachments() {
	manager := attachments.NewAttachmentManager("/path/to/repository")

	var totalSize int64
	var count int

	err := manager.StreamAttachments(func(attachment *attachments.Attachment) error {
		count++
		totalSize += attachment.Size

		// Process each attachment without loading all into memory
		fmt.Printf("Processing attachment %s...\n", attachment.Hash[:8])
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Processed %d attachments, total size: %d bytes\n", count, totalSize)
}

// Example demonstrates finding orphaned attachments
func ExampleAttachmentManager_FindOrphanedAttachments() {
	manager := attachments.NewAttachmentManager("/path/to/repository")

	// Get referenced hashes from SMS reader (example)
	referencedHashes := map[string]bool{
		"3ceb5c413ee02895bf1f357a8c2cc2bec824f4d8aad13aeab69303f341c8b781": true,
		"26fdc315fadc05db9f8f3236fc30b9f0ca044e56ec1e9450ccd5fdab900e9e46": true,
	}

	orphaned, err := manager.FindOrphanedAttachments(referencedHashes)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d orphaned attachments:\n", len(orphaned))
	for _, attachment := range orphaned {
		fmt.Printf("  %s (%d bytes) - not referenced by any message\n",
			attachment.Hash, attachment.Size)
	}
}

// Example demonstrates validating attachment directory structure
func ExampleAttachmentManager_ValidateAttachmentStructure() {
	manager := attachments.NewAttachmentManager("/path/to/repository")

	err := manager.ValidateAttachmentStructure()
	if err != nil {
		fmt.Printf("Validation failed: %v\n", err)
		return
	}

	fmt.Println("Attachment directory structure validation passed")
}

// Example demonstrates getting attachment statistics
func ExampleAttachmentManager_GetAttachmentStats() {
	manager := attachments.NewAttachmentManager("/path/to/repository")

	// Get referenced hashes from SMS reader
	referencedHashes := map[string]bool{
		"3ceb5c413ee02895bf1f357a8c2cc2bec824f4d8aad13aeab69303f341c8b781": true,
		"26fdc315fadc05db9f8f3236fc30b9f0ca044e56ec1e9450ccd5fdab900e9e46": true,
	}

	stats, err := manager.GetAttachmentStats(referencedHashes)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Attachment Statistics:\n")
	fmt.Printf("  Total: %d attachments (%d bytes)\n", stats.TotalCount, stats.TotalSize)
	fmt.Printf("  Orphaned: %d attachments\n", stats.OrphanedCount)
	fmt.Printf("  Corrupted: %d attachments\n", stats.CorruptedCount)
}

// Example demonstrates checking if attachment exists
func ExampleAttachmentManager_AttachmentExists() {
	manager := attachments.NewAttachmentManager("/path/to/repository")

	hashes := []string{
		"3ceb5c413ee02895bf1f357a8c2cc2bec824f4d8aad13aeab69303f341c8b781",
		"26fdc315fadc05db9f8f3236fc30b9f0ca044e56ec1e9450ccd5fdab900e9e46",
		"nonexistent1234567890abcdef1234567890abcdef1234567890abcdef123456",
	}

	for _, hash := range hashes {
		exists, err := manager.AttachmentExists(hash)
		if err != nil {
			fmt.Printf("Error checking %s: %v\n", hash[:8], err)
			continue
		}

		if exists {
			fmt.Printf("Attachment %s... exists\n", hash[:8])
		} else {
			fmt.Printf("Attachment %s... not found\n", hash[:8])
		}
	}
}

// Example demonstrates getting attachment path
func ExampleAttachmentManager_GetAttachmentPath() {
	manager := attachments.NewAttachmentManager("/path/to/repository")

	hash := "3ceb5c413ee02895bf1f357a8c2cc2bec824f4d8aad13aeab69303f341c8b781"
	path := manager.GetAttachmentPath(hash)

	fmt.Printf("Attachment path for hash %s:\n", hash[:8])
	fmt.Printf("  %s\n", path)

	// Path structure explanation
	fmt.Printf("Path structure: attachments/[first-2-chars]/[full-hash]\n")
	fmt.Printf("  First 2 chars: %s\n", hash[:2])
	fmt.Printf("  Full hash: %s\n", hash)
}

// Example demonstrates error handling patterns
func ExampleAttachmentManager_errorHandling() {
	manager := attachments.NewAttachmentManager("/path/to/repository")

	// Example 1: Invalid hash format
	_, err := manager.GetAttachment("invalid-hash")
	if err != nil {
		fmt.Printf("Invalid hash error: %v\n", err)
	}

	// Example 2: Non-existent attachment
	hash := "3ceb5c413ee02895bf1f357a8c2cc2bec824f4d8aad13aeab69303f341c8b781"
	attachment, err := manager.GetAttachment(hash)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else if !attachment.Exists {
		fmt.Printf("Attachment %s not found (but no error)\n", hash[:8])
	}

	// Example 3: Read error handling
	content, err := manager.ReadAttachment(hash)
	if err != nil {
		fmt.Printf("Read error: %v\n", err)
	} else {
		fmt.Printf("Read %d bytes successfully\n", len(content))
	}
}

// Example demonstrates working with attachment metadata
func ExampleAttachment() {
	manager := attachments.NewAttachmentManager("/path/to/repository")

	hash := "3ceb5c413ee02895bf1f357a8c2cc2bec824f4d8aad13aeab69303f341c8b781"
	attachment, err := manager.GetAttachment(hash)
	if err != nil {
		log.Fatal(err)
	}

	// Access attachment metadata
	fmt.Printf("Attachment Details:\n")
	fmt.Printf("  Hash: %s\n", attachment.Hash)
	fmt.Printf("  Path: %s\n", attachment.Path)
	fmt.Printf("  Size: %d bytes\n", attachment.Size)
	fmt.Printf("  Exists: %t\n", attachment.Exists)

	// Convert size to human readable format
	var sizeStr string
	if attachment.Size < 1024 {
		sizeStr = fmt.Sprintf("%d B", attachment.Size)
	} else if attachment.Size < 1024*1024 {
		sizeStr = fmt.Sprintf("%.1f KB", float64(attachment.Size)/1024)
	} else {
		sizeStr = fmt.Sprintf("%.1f MB", float64(attachment.Size)/(1024*1024))
	}
	fmt.Printf("  Human size: %s\n", sizeStr)
}

// Example demonstrates using the new DirectoryAttachmentStorage
func ExampleDirectoryAttachmentStorage() {
	// Create storage instance
	storage := attachments.NewDirectoryAttachmentStorage("/path/to/repository")

	// Store a new attachment
	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	data := []byte("Hello, world!")
	metadata := attachments.AttachmentInfo{
		Hash:         hash,
		OriginalName: "hello.txt",
		MimeType:     "text/plain",
		Size:         int64(len(data)),
		CreatedAt:    time.Now().UTC(),
		SourceMMS:    "mms-12345",
	}

	err := storage.Store(hash, data, metadata)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Stored attachment: %s\n", hash[:8])
	fmt.Println("New directory structure created with metadata")

	// Output:
	// Stored attachment: e3b0c442
	// New directory structure created with metadata
}

// Example demonstrates retrieving attachment metadata
func ExampleDirectoryAttachmentStorage_GetMetadata() {
	storage := attachments.NewDirectoryAttachmentStorage("/path/to/repository")

	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	metadata, err := storage.GetMetadata(hash)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Attachment Metadata:\n")
	fmt.Printf("  Hash: %s\n", metadata.Hash[:8])
	fmt.Printf("  Original Name: %s\n", metadata.OriginalName)
	fmt.Printf("  MIME Type: %s\n", metadata.MimeType)
	fmt.Printf("  Size: %d bytes\n", metadata.Size)
	fmt.Printf("  Created: %s\n", metadata.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Source MMS: %s\n", metadata.SourceMMS)
}

// Example demonstrates MIME type to file extension mapping
func ExampleGetFileExtension() {
	mimeTypes := []string{
		"image/jpeg",
		"image/png",
		"application/pdf",
		"video/mp4",
		"audio/mpeg",
		"application/unknown",
	}

	fmt.Println("MIME Type to Extension Mapping:")
	for _, mimeType := range mimeTypes {
		ext := attachments.GetFileExtension(mimeType)
		fmt.Printf("  %s -> .%s\n", mimeType, ext)
	}

	// Output:
	// MIME Type to Extension Mapping:
	//   image/jpeg -> .jpg
	//   image/png -> .png
	//   application/pdf -> .pdf
	//   video/mp4 -> .mp4
	//   audio/mpeg -> .mp3
	//   application/unknown -> .bin
}

// Example demonstrates filename generation
func ExampleGenerateFilename() {
	testCases := []struct {
		originalName string
		mimeType     string
	}{
		{"photo.jpg", "image/jpeg"},
		{"", "image/png"},
		{"document.pdf", "application/pdf"},
		{"null", "video/mp4"},
		{"readme.txt", ""},
	}

	fmt.Println("Filename Generation:")
	for _, tc := range testCases {
		filename := attachments.GenerateFilename(tc.originalName, tc.mimeType)
		fmt.Printf("  Original: '%s', MIME: '%s' -> '%s'\n",
			tc.originalName, tc.mimeType, filename)
	}

	// Output:
	// Filename Generation:
	//   Original: 'photo.jpg', MIME: 'image/jpeg' -> 'photo.jpg'
	//   Original: '', MIME: 'image/png' -> 'attachment.png'
	//   Original: 'document.pdf', MIME: 'application/pdf' -> 'document.pdf'
	//   Original: 'null', MIME: 'video/mp4' -> 'attachment.mp4'
	//   Original: 'readme.txt', MIME: '' -> 'readme.txt'
}

// Example demonstrates migration from legacy format
func ExampleMigrationManager() {
	// Create migration manager
	migrationManager := attachments.NewMigrationManager("/path/to/repository")

	// Check current migration status
	status, err := migrationManager.GetMigrationStatus()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Migration Status:\n")
	fmt.Printf("  Total attachments: %d\n", status["total_count"])
	fmt.Printf("  Legacy format: %d\n", status["legacy_count"])
	fmt.Printf("  New format: %d\n", status["new_count"])
	fmt.Printf("  Migration complete: %t\n", status["migrated"])

	// Perform migration if needed
	if !status["migrated"].(bool) {
		summary, err := migrationManager.MigrateAllAttachments()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("\nMigration Results:\n")
		fmt.Printf("  Found: %d attachments\n", summary.TotalFound)
		fmt.Printf("  Migrated: %d attachments\n", summary.Migrated)
		fmt.Printf("  Failed: %d attachments\n", summary.Failed)
		fmt.Printf("  Skipped: %d attachments\n", summary.Skipped)
	}
}

// Example demonstrates dry run migration
func ExampleMigrationManager_dryRun() {
	migrationManager := attachments.NewMigrationManager("/path/to/repository")

	// Enable dry run mode
	migrationManager.SetDryRun(true)

	// Run migration simulation
	summary, err := migrationManager.MigrateAllAttachments()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Dry Run Migration Results:\n")
	fmt.Printf("  Would migrate: %d attachments\n", summary.Migrated)
	fmt.Printf("  Would fail: %d attachments\n", summary.Failed)
	fmt.Printf("  Would skip: %d attachments\n", summary.Skipped)
	fmt.Println("No actual changes made to filesystem")
}

// Example demonstrates attachment format detection
func ExampleAttachmentManager_formatDetection() {
	manager := attachments.NewAttachmentManager("/path/to/repository")

	hash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	// Check which format this attachment uses
	if manager.IsNewFormat(hash) {
		fmt.Printf("Attachment %s uses new directory format\n", hash[:8])

		// Access metadata
		storage := attachments.NewDirectoryAttachmentStorage("/path/to/repository")
		metadata, err := storage.GetMetadata(hash)
		if err == nil {
			fmt.Printf("  Original name: %s\n", metadata.OriginalName)
			fmt.Printf("  MIME type: %s\n", metadata.MimeType)
		}
	} else if manager.IsLegacyFormat(hash) {
		fmt.Printf("Attachment %s uses legacy file format\n", hash[:8])
	} else {
		fmt.Printf("Attachment %s not found\n", hash[:8])
	}
}
