package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// CalVerResult contains the components of the generated CalVer.
type CalVerResult struct {
	YearMonth    string // yyyy.mm
	DefaultCount int    // commit count on default branch (or tag baseline + commits since tag)
	IsDefault    bool   // true if currently on default branch
	BranchName   string // raw branch name
	CleanBranch  string // sanitized branch name for versioning
	BranchCount  int    // commit count on feature branch since branching off default branch
	Version      string // full formatted CalVer string
}

// GetCalVer computes the calendar version for the git repository at repoPath.
// If now is zero, time.Now() in UTC is used.
func GetCalVer(repoPath string, now time.Time) (*CalVerResult, error) {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	if !isGitRepository(absPath) {
		return nil, fmt.Errorf("%w: %s", ErrNotGitRepository, absPath)
	}

	if now.IsZero() {
		now = time.Now()
	}
	yearMonth := now.Format("2006.01")

	defaultBranch, err := GetDefaultBranch(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to determine default branch: %w", err)
	}

	currentBranch, err := getCurrentBranch(absPath)
	if err != nil {
		currentBranch = "detached"
	}

	isDefault := (currentBranch == defaultBranch)

	res := &CalVerResult{
		YearMonth:   yearMonth,
		IsDefault:   isDefault,
		BranchName:  currentBranch,
		CleanBranch: sanitizeBranchName(currentBranch),
	}

	if isDefault {
		// Calculate commit count on default branch (factoring in tag baseline)
		count, err := getCommitCountSinceTag(absPath, defaultBranch)
		if err != nil {
			return nil, fmt.Errorf("failed to count commits on default branch: %w", err)
		}
		res.DefaultCount = count
		res.Version = fmt.Sprintf("%s.%d", yearMonth, count)
	} else {
		// Non-default branch logic:
		// 1. Find merge-base between defaultBranch and current branch/HEAD
		mergeBase, err := getMergeBase(absPath, defaultBranch, "HEAD")
		if err != nil {
			mergeBase = defaultBranch
		}

		// 2. Count default branch commits up to merge-base (factoring in tag baseline)
		defaultCount, err := getCommitCountSinceTag(absPath, mergeBase)
		if err != nil {
			return nil, fmt.Errorf("failed to count commits up to merge-base: %w", err)
		}
		res.DefaultCount = defaultCount

		// 3. Count commits on current feature branch since merge-base
		branchCount, err := getCommitCountBetween(absPath, mergeBase, "HEAD")
		if err != nil {
			return nil, fmt.Errorf("failed to count feature branch commits: %w", err)
		}
		res.BranchCount = branchCount

		res.Version = fmt.Sprintf("%s.%d-%s.%d", yearMonth, defaultCount, res.CleanBranch, branchCount)
	}

	return res, nil
}

func sanitizeBranchName(name string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9\-]+`)
	sanitized := reg.ReplaceAllString(name, "-")
	sanitized = strings.Trim(sanitized, "-")
	if sanitized == "" {
		return "branch"
	}
	return sanitized
}

func getLatestTag(dir, ref string) (string, error) {
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0", ref)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func getCommitCountSinceTag(dir, ref string) (int, error) {
	tag, err := getLatestTag(dir, ref)
	if err != nil || tag == "" {
		return getCommitCount(dir, ref)
	}

	revRange := fmt.Sprintf("%s..%s", tag, ref)
	countSinceTag, err := getCommitCount(dir, revRange)
	if err != nil {
		return 0, err
	}

	baseCount := parseTagBaseline(tag)
	if baseCount < 0 {
		return getCommitCount(dir, ref)
	}

	return baseCount + countSinceTag, nil
}

// parseTagBaseline extracts the patch/commit count integer from tags like "2026.07.12", "v2026.07.12", or "v1.2.3".
func parseTagBaseline(tag string) int {
	cleanTag := strings.TrimPrefix(strings.TrimPrefix(tag, "v"), "V")

	// Split by non-alphanumeric separators (e.g. dots, hyphens)
	parts := regexp.MustCompile(`[.\-_]+`).Split(cleanTag, -1)

	// Search backwards for the last pure numeric segment (e.g. "12" in "2026.07.12")
	for i := len(parts) - 1; i >= 0; i-- {
		if val, err := strconv.Atoi(parts[i]); err == nil {
			return val
		}
	}

	return -1
}

func getCommitCountBetween(dir, fromRef, toRef string) (int, error) {
	revRange := fmt.Sprintf("%s..%s", fromRef, toRef)
	return getCommitCount(dir, revRange)
}

func getCommitCount(dir, revRange string) (int, error) {
	cmd := exec.Command("git", "rev-list", "--count", revRange)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("git rev-list error: %w", err)
	}
	str := strings.TrimSpace(out.String())
	count, err := strconv.Atoi(str)
	if err != nil {
		return 0, fmt.Errorf("invalid commit count output '%s': %w", str, err)
	}
	return count, nil
}

func getMergeBase(dir, ref1, ref2 string) (string, error) {
	cmd := exec.Command("git", "merge-base", ref1, ref2)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}
