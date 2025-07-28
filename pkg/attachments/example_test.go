package attachments_test

import (
	"fmt"
	"log"

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