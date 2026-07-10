package attachments

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/phillipgreenii/mobilecombackup/pkg/security"
)

// AttachmentReferencesProvider provides attachment references from messages
type AttachmentReferencesProvider interface {
	GetAllAttachmentRefs(ctx context.Context) (map[string]bool, error)
}

// OrphanRemovalResult represents the result of orphan attachment removal
type OrphanRemovalResult struct {
	AttachmentsScanned int             `json:"attachments_scanned"`
	OrphansFound       int             `json:"orphans_found"`
	OrphansRemoved     int             `json:"orphans_removed"`
	BytesFreed         int64           `json:"bytes_freed"`
	RemovalFailures    int             `json:"removal_failures"`
	FailedRemovals     []FailedRemoval `json:"failed_removals,omitempty"`
}

// FailedRemoval represents a single failed removal
type FailedRemoval struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

// OrphanRemover handles finding and removing orphaned attachments
type OrphanRemover struct {
	refProvider   AttachmentReferencesProvider
	attachmentMgr *AttachmentManager
	pathValidator *security.PathValidator
	dryRun        bool
}

// NewOrphanRemover creates a new orphan remover
func NewOrphanRemover(refProvider AttachmentReferencesProvider, attachmentMgr *AttachmentManager, dryRun bool) *OrphanRemover {
	return &OrphanRemover{
		refProvider:   refProvider,
		attachmentMgr: attachmentMgr,
		pathValidator: security.NewPathValidator(attachmentMgr.GetRepoPath()),
		dryRun:        dryRun,
	}
}

// RemoveOrphans finds and removes orphaned attachments
func (r *OrphanRemover) RemoveOrphans(ctx context.Context) (*OrphanRemovalResult, error) {
	// Find orphaned attachments
	orphanedAttachments, totalCount, err := r.findOrphanedAttachments(ctx)
	if err != nil {
		return nil, err
	}

	// Create initial result
	result := &OrphanRemovalResult{
		AttachmentsScanned: totalCount,
		OrphansFound:       len(orphanedAttachments),
		OrphansRemoved:     0,
		BytesFreed:         0,
		RemovalFailures:    0,
		FailedRemovals:     []FailedRemoval{},
	}

	if r.dryRun {
		return r.handleDryRun(result, orphanedAttachments), nil
	}

	return r.executeRemoval(result, orphanedAttachments), nil
}

// findOrphanedAttachments finds and returns orphaned attachments
func (r *OrphanRemover) findOrphanedAttachments(ctx context.Context) ([]*Attachment, int, error) {
	// Get all attachment references from SMS messages
	referencedHashes, err := r.refProvider.GetAllAttachmentRefs(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get attachment references: %w", err)
	}

	// Find orphaned attachments
	orphanedAttachments, err := r.attachmentMgr.FindOrphanedAttachments(referencedHashes)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find orphaned attachments: %w", err)
	}

	// Count total attachments scanned
	totalCount := 0
	err = r.attachmentMgr.StreamAttachments(func(*Attachment) error {
		totalCount++
		return nil
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count attachments: %w", err)
	}

	return orphanedAttachments, totalCount, nil
}

// handleDryRun calculates what would be removed without actually removing
func (r *OrphanRemover) handleDryRun(result *OrphanRemovalResult, orphanedAttachments []*Attachment) *OrphanRemovalResult {
	// Calculate potential bytes freed
	for _, attachment := range orphanedAttachments {
		result.BytesFreed += attachment.Size
	}
	result.OrphansRemoved = len(orphanedAttachments) // Would be removed
	return result
}

// executeRemoval actually removes the orphaned attachments
func (r *OrphanRemover) executeRemoval(result *OrphanRemovalResult, orphanedAttachments []*Attachment) *OrphanRemovalResult {
	repoPath := r.attachmentMgr.GetRepoPath()
	emptyDirs := make(map[string]bool) // Track directories that might become empty

	for _, attachment := range orphanedAttachments {
		r.removeOrphanedAttachment(attachment, repoPath, result, emptyDirs)
	}

	// Clean up empty directories
	for dir := range emptyDirs {
		// Validate directory path before cleanup
		safeDirPath, err := r.pathValidator.JoinAndValidate(dir)
		if err != nil {
			// Skip cleanup for invalid directory paths
			continue
		}
		r.cleanupEmptyDirectory(safeDirPath)
	}

	return result
}

// removeOrphanedAttachment removes a single orphaned attachment
func (r *OrphanRemover) removeOrphanedAttachment(
	attachment *Attachment,
	repoPath string,
	result *OrphanRemovalResult,
	emptyDirs map[string]bool,
) {
	// Construct the full path for attachment removal
	var fullPath string

	if filepath.IsAbs(attachment.Path) {
		fullPath = attachment.Path
	} else {
		fullPath = filepath.Join(repoPath, attachment.Path)
	}

	// Validate that the final path is within the repository bounds
	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		result.RemovalFailures++
		result.FailedRemovals = append(result.FailedRemovals, FailedRemoval{
			Path:  attachment.Path,
			Error: fmt.Sprintf("failed to resolve repo path: %v", err),
		})
		return
	}

	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		result.RemovalFailures++
		result.FailedRemovals = append(result.FailedRemovals, FailedRemoval{
			Path:  attachment.Path,
			Error: fmt.Sprintf("failed to resolve attachment path: %v", err),
		})
		return
	}

	// Ensure the path is within repository boundaries
	if !strings.HasPrefix(absFullPath+string(filepath.Separator), absRepoPath+string(filepath.Separator)) {
		result.RemovalFailures++
		result.FailedRemovals = append(result.FailedRemovals, FailedRemoval{
			Path:  attachment.Path,
			Error: fmt.Sprintf("path %s is outside repository %s", absFullPath, absRepoPath),
		})
		return
	}

	// Track the directory for potential cleanup
	dir := filepath.Dir(attachment.Path)
	emptyDirs[dir] = true

	// Attempt to remove the file
	if err := os.Remove(absFullPath); err != nil {
		result.RemovalFailures++
		result.FailedRemovals = append(result.FailedRemovals, FailedRemoval{
			Path:  attachment.Path,
			Error: err.Error(),
		})
	} else {
		result.OrphansRemoved++
		result.BytesFreed += attachment.Size
	}
}

// cleanupEmptyDirectory removes a directory if it's empty
func (r *OrphanRemover) cleanupEmptyDirectory(dirPath string) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return // Can't read directory, skip cleanup
	}

	if len(entries) == 0 {
		// Directory is empty, try to remove it
		_ = os.Remove(dirPath)
		// Note: We ignore errors here as cleanup is best-effort
	}
}
