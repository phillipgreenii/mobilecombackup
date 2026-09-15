package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	forceFullVerification  bool
	forceQuickVerification bool
	skipTests              bool
	preCommitMode          bool
	verboseOutput          bool
)

var smartVerifyCmd = &cobra.Command{
	Use:   "smart-verify",
	Short: "Context-aware verification that adapts based on changes",
	Long: `Smart verification analyzes your changes and runs appropriate verification steps.

This command:
- Analyzes git changes to understand the scope of modifications
- Selects an appropriate verification strategy (minimal, targeted, progressive, full)
- Runs only the necessary verification steps to maintain quality while saving time
- Always runs full verification for pre-commit to ensure safety

Change Categories:
  docs-only     - Only documentation files changed (skip tests, ~90% time saved)
  test-only     - Only test files changed (run affected tests, ~50% time saved)
  single-package - Changes within one package (targeted tests, ~60% time saved)
  multi-package - Changes across packages (full verification for safety)
  mixed         - Mixed changes (full verification for safety)`,
	RunE: runSmartVerify,
}

func init() {
	// Add flags
	smartVerifyCmd.Flags().BoolVar(&forceFullVerification, "full", false, "Force full verification regardless of changes")
	smartVerifyCmd.Flags().BoolVar(&forceQuickVerification, "quick", false,
		"Force minimal verification (dangerous - use with caution)")
	smartVerifyCmd.Flags().BoolVar(&skipTests, "skip-tests", false, "Skip all test execution (dangerous - use for docs-only work)")
	smartVerifyCmd.Flags().BoolVar(&preCommitMode, "pre-commit", false, "Pre-commit mode (always runs full verification)")
	smartVerifyCmd.Flags().BoolVarP(&verboseOutput, "verbose", "v", false, "Verbose output showing analysis and decisions")

	// Add to root command
	rootCmd.AddCommand(smartVerifyCmd)
}

// ChangeCategory classifies a staged change for verification-strategy selection.
type ChangeCategory string

const (
	DocumentationOnly ChangeCategory = "docs-only"
	TestOnly          ChangeCategory = "test-only"
	SinglePackage     ChangeCategory = "single-package"
	MultiPackage      ChangeCategory = "multi-package"
	Mixed             ChangeCategory = "mixed"
)

type ChangeContext struct {
	Category         ChangeCategory
	StagedFiles      []string
	AffectedPackages []string
	HasCodeChanges   bool
	HasTestChanges   bool
	HasDocChanges    bool
	ChangeCount      int
}

type VerificationStrategy struct {
	Level        string
	RunFormatter bool
	RunLinter    bool
	RunTests     bool
	TestPackages []string
	RunBuild     bool
	Reason       string
	TimeSaved    time.Duration
}

func runSmartVerify(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

	fmt.Println("🔍 Smart Verification - Analyzing changes...")

	// Analyze changes
	context, err := analyzeGitChanges()
	if err != nil {
		return fmt.Errorf("failed to analyze changes: %w", err)
	}

	// Determine strategy
	strategy := determineVerificationStrategy(context)

	// Report decision
	if verboseOutput {
		reportDecision(strategy, context)
	}

	// Execute strategy
	result, err := executeStrategy(strategy)
	if err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	// Report results
	reportResults(result, time.Since(startTime))

	if !result {
		return fmt.Errorf("verification failed")
	}

	return nil
}

// analyzeGitChanges examines git changes and returns context
func analyzeGitChanges() (*ChangeContext, error) {
	// Get staged files
	cmd := exec.Command("git", "diff", "--cached", "--name-only", "--diff-filter=AMDRC")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git command failed: %w", err)
	}

	files := []string{}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			files = append(files, line)
		}
	}

	if len(files) == 0 {
		return &ChangeContext{
			Category:    DocumentationOnly,
			StagedFiles: files,
		}, nil
	}

	context := &ChangeContext{
		StagedFiles:      files,
		AffectedPackages: []string{},
	}

	packages := make(map[string]bool)

	// Analyze each file
	for _, file := range files {
		switch {
		case isDocumentationFile(file):
			context.HasDocChanges = true
		case isTestFile(file):
			context.HasTestChanges = true
			if pkg := getPackageFromFile(file); pkg != "" {
				packages[pkg] = true
			}
		case isGoFile(file):
			context.HasCodeChanges = true
			if pkg := getPackageFromFile(file); pkg != "" {
				packages[pkg] = true
			}
		}
	}

	// Convert packages to slice
	for pkg := range packages {
		context.AffectedPackages = append(context.AffectedPackages, pkg)
	}

	// Determine category
	context.Category = determineCategory(context)
	context.ChangeCount = getChangeCount()

	return context, nil
}

func determineCategory(context *ChangeContext) ChangeCategory {
	hasCode := context.HasCodeChanges
	hasTest := context.HasTestChanges
	hasDoc := context.HasDocChanges

	changeTypes := 0
	if hasCode {
		changeTypes++
	}
	if hasTest {
		changeTypes++
	}
	if hasDoc {
		changeTypes++
	}

	if changeTypes > 1 {
		return Mixed
	}

	if hasDoc && !hasCode && !hasTest {
		return DocumentationOnly
	}

	if hasTest && !hasCode && !hasDoc {
		return TestOnly
	}

	if hasCode && !hasTest && !hasDoc {
		if len(context.AffectedPackages) <= 1 {
			return SinglePackage
		}
		return MultiPackage
	}

	return Mixed
}

// determineVerificationStrategy enumerates the verification-strategy decision
// table (change-category x mode). It is long because it is a flat decision
// table, not nested logic; splitting it would scatter one policy across
// several helpers without reducing its real complexity.
func determineVerificationStrategy(context *ChangeContext) *VerificationStrategy { //nolint:funlen
	// Pre-commit always uses full verification
	if preCommitMode {
		return &VerificationStrategy{
			Level:        "full",
			RunFormatter: true,
			RunLinter:    true,
			RunTests:     true,
			RunBuild:     true,
			Reason:       "Pre-commit safety requires full verification",
			TimeSaved:    0,
		}
	}

	// Handle override flags
	if forceFullVerification {
		return &VerificationStrategy{
			Level:        "full",
			RunFormatter: true,
			RunLinter:    true,
			RunTests:     true,
			RunBuild:     true,
			Reason:       "Full verification forced by --full flag",
			TimeSaved:    0,
		}
	}

	if forceQuickVerification {
		return &VerificationStrategy{
			Level:        "minimal",
			RunFormatter: true,
			RunLinter:    true,
			RunTests:     false,
			RunBuild:     false,
			Reason:       "Quick verification forced by --quick flag (dangerous)",
			TimeSaved:    25 * time.Second,
		}
	}

	if skipTests {
		return &VerificationStrategy{
			Level:        "minimal",
			RunFormatter: true,
			RunLinter:    true,
			RunTests:     false,
			RunBuild:     true,
			Reason:       "Test execution skipped by --skip-tests flag (dangerous)",
			TimeSaved:    20 * time.Second,
		}
	}

	// Strategy based on change category
	switch context.Category {
	case DocumentationOnly:
		return &VerificationStrategy{
			Level:        "minimal",
			RunFormatter: true,
			RunLinter:    true,
			RunTests:     false,
			RunBuild:     false,
			Reason:       "Documentation-only changes detected - skipping tests and build",
			TimeSaved:    25 * time.Second,
		}

	case TestOnly:
		return &VerificationStrategy{
			Level:        "targeted",
			RunFormatter: true,
			RunLinter:    true,
			RunTests:     true,
			TestPackages: context.AffectedPackages,
			RunBuild:     false,
			Reason:       "Test-only changes detected - running affected package tests",
			TimeSaved:    15 * time.Second,
		}

	case SinglePackage:
		if context.ChangeCount >= 5 {
			return fullVerificationStrategy("Change accumulation threshold reached - running full verification")
		}
		return &VerificationStrategy{
			Level:        "targeted",
			RunFormatter: true,
			RunLinter:    true,
			RunTests:     true,
			TestPackages: context.AffectedPackages,
			RunBuild:     true,
			Reason:       "Single package changes detected - running targeted tests",
			TimeSaved:    12 * time.Second,
		}

	case MultiPackage:
		return fullVerificationStrategy("Multi-package changes detected - running full verification for safety")

	case Mixed:
		return fullVerificationStrategy("Mixed changes detected - running full verification for safety")
	}

	return fullVerificationStrategy("Unknown change category, defaulting to full verification")
}

func fullVerificationStrategy(reason string) *VerificationStrategy {
	return &VerificationStrategy{
		Level:        "full",
		RunFormatter: true,
		RunLinter:    true,
		RunTests:     true,
		RunBuild:     true,
		Reason:       reason,
		TimeSaved:    0,
	}
}

// executeStrategy's error return is currently always nil: each step's
// failure is folded into the success bool (and printed) rather than
// propagated, so the caller's single !result check covers every step. The
// error slot is kept for a future failure mode that is not one of these
// steps (e.g. a panic recovery), matching the call site's existing
// `result, err := executeStrategy(...)` handling.
func executeStrategy(strategy *VerificationStrategy) (bool, error) { //nolint:unparam
	success := true

	if strategy.RunFormatter {
		if verboseOutput {
			fmt.Println("🎨 Running formatter...")
		}
		if err := runJustCommand("formatter"); err != nil {
			fmt.Printf("❌ Formatter failed: %v\n", err)
			success = false
		}
	}

	// The nesting here is a flat "if enabled { log; branch on packages; run }"
	// sequence, not compounding conditional logic, so splitting it would not
	// reduce real complexity.
	if strategy.RunTests { //nolint:nestif
		if verboseOutput {
			if len(strategy.TestPackages) > 0 {
				fmt.Printf("🧪 Running tests for packages: %s\n", strings.Join(strategy.TestPackages, ", "))
			} else {
				fmt.Println("🧪 Running all tests...")
			}
		}

		if len(strategy.TestPackages) > 0 {
			// Run targeted tests
			if err := runTargetedTests(strategy.TestPackages); err != nil {
				fmt.Printf("❌ Tests failed: %v\n", err)
				success = false
			}
		} else {
			// Run all tests
			if err := runJustCommand("tests"); err != nil {
				fmt.Printf("❌ Tests failed: %v\n", err)
				success = false
			}
		}
	}

	if strategy.RunLinter {
		if verboseOutput {
			fmt.Println("🔍 Running linter...")
		}
		if err := runJustCommand("linter"); err != nil {
			fmt.Printf("❌ Linter failed: %v\n", err)
			success = false
		}
	}

	if strategy.RunBuild {
		if verboseOutput {
			fmt.Println("🔨 Running build...")
		}
		if err := runJustCommand("build-cli"); err != nil {
			fmt.Printf("❌ Build failed: %v\n", err)
			success = false
		}
	}

	return success, nil
}

func runJustCommand(recipe string) error {
	// recipe is always one of this file's own hardcoded literals ("linter",
	// "build-cli", ...), never external input.
	cmd := exec.Command("just", recipe) //nolint:gosec
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runTargetedTests runs gotestsum against a runtime-computed package list.
// The prior task runner's arbitrary-command passthrough handled this by
// appending the package globs onto a fixed "run -- gotestsum ..." prefix;
// `just` has no equivalent passthrough verb, so tc-5lxy.14 adds a
// parameterized `test-packages` recipe to the justfile (tc-5lxy.8)
// specifically to give this call a named target. The package globs are
// joined into one positional argument, which the recipe's own bash body
// re-splits on whitespace.
func runTargetedTests(packages []string) error {
	pkgArgs := make([]string, 0, len(packages))
	for _, pkg := range packages {
		pkgArgs = append(pkgArgs, fmt.Sprintf("./%s/...", pkg))
	}

	// pkgArgs is built from package names formatted into a fixed
	// "./%s/..." pattern -- never externally supplied shell input -- and
	// exec.Command never invokes a shell, so joining them into a single
	// argument for just to re-split internally is safe.
	cmd := exec.Command("just", "test-packages", strings.Join(pkgArgs, " ")) //nolint:gosec
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func reportDecision(strategy *VerificationStrategy, context *ChangeContext) {
	fmt.Printf("📁 Files changed: %d\n", len(context.StagedFiles))
	fmt.Printf("📂 Change category: %s\n", context.Category)

	if len(context.AffectedPackages) > 0 {
		fmt.Printf("📦 Affected packages: %s\n", strings.Join(context.AffectedPackages, ", "))
	}

	if context.ChangeCount > 1 {
		fmt.Printf("🔄 Change accumulation: %d changes\n", context.ChangeCount)
	}

	fmt.Printf("🎯 Selected strategy: %s\n", strategy.Level)
	fmt.Printf("💡 Reason: %s\n", strategy.Reason)

	steps := []string{}
	if strategy.RunFormatter {
		steps = append(steps, "formatter")
	}
	if strategy.RunTests {
		if len(strategy.TestPackages) > 0 {
			steps = append(steps, fmt.Sprintf("tests (%s)", strings.Join(strategy.TestPackages, ", ")))
		} else {
			steps = append(steps, "tests (all)")
		}
	}
	if strategy.RunLinter {
		steps = append(steps, "linter")
	}
	if strategy.RunBuild {
		steps = append(steps, "build")
	}

	if len(steps) > 0 {
		fmt.Printf("📋 Will execute: %s\n", strings.Join(steps, " → "))
	}

	if strategy.TimeSaved > 0 {
		fmt.Printf("⏱️  Estimated time saved: ~%v\n", strategy.TimeSaved)
	}

	fmt.Println()
}

func reportResults(success bool, duration time.Duration) {
	fmt.Println()

	if success {
		fmt.Println("✅ Smart verification completed successfully!")
	} else {
		fmt.Println("❌ Smart verification failed!")
	}

	fmt.Printf("⏱️  Duration: %v\n", duration.Round(time.Millisecond))
	fmt.Println()
}

// Helper functions
func isDocumentationFile(file string) bool {
	ext := strings.ToLower(filepath.Ext(file))
	docExts := []string{".md", ".markdown", ".yaml", ".yml", ".txt", ".rst"}

	for _, docExt := range docExts {
		if ext == docExt {
			return true
		}
	}

	base := strings.ToLower(filepath.Base(file))
	return base == "readme" || base == "license" || base == "changelog"
}

func isTestFile(file string) bool {
	base := filepath.Base(file)
	return strings.HasSuffix(base, "_test.go") || strings.HasPrefix(base, "test_")
}

func isGoFile(file string) bool {
	return strings.HasSuffix(file, ".go") && !isTestFile(file)
}

func getPackageFromFile(file string) string {
	dir := filepath.Dir(file)
	if dir == "." {
		return ""
	}

	parts := strings.Split(dir, "/")
	if len(parts) >= 2 && (parts[0] == "pkg" || parts[0] == "cmd") {
		return strings.Join(parts[:2], "/")
	}

	return dir
}

func getChangeCount() int {
	cmd := exec.Command("git", "rev-list", "--count", "HEAD", "--since=1.day.ago")
	output, err := cmd.Output()
	if err != nil {
		return 1
	}

	countStr := strings.TrimSpace(string(output))
	var count int
	_, _ = fmt.Sscanf(countStr, "%d", &count)
	if count == 0 {
		count = 1
	}
	return count
}
