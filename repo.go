package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrNotGitRepository is returned when the given path is not inside a Git repository.
var ErrNotGitRepository = errors.New("not a git repository")

// GetDefaultBranch attempts to determine the default branch of the Git repository at repoPath.
func GetDefaultBranch(repoPath string) (string, error) {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Verify the directory exists
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("cannot access path: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", absPath)
	}

	// Verify it's a git repo
	if !isGitRepository(absPath) {
		return "", fmt.Errorf("%w: %s", ErrNotGitRepository, absPath)
	}

	// Strategy 1: Check remote origin symbolic ref (origin/HEAD)
	if branch, err := getRemoteHeadBranch(absPath, "origin"); err == nil && branch != "" {
		return branch, nil
	}

	// Strategy 2: Check any remote's HEAD
	if branch, err := getAnyRemoteHeadBranch(absPath); err == nil && branch != "" {
		return branch, nil
	}

	// Strategy 3: Check common default branch names present in local branches
	if branch, err := getFallbackLocalBranch(absPath); err == nil && branch != "" {
		return branch, nil
	}

	// Strategy 4: Fallback to current branch symbolic ref (HEAD)
	if branch, err := getCurrentBranch(absPath); err == nil && branch != "" {
		return branch, nil
	}

	return "", errors.New("unable to determine default branch")
}

func isGitRepository(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return false
	}
	return strings.TrimSpace(out.String()) == "true"
}

func getRemoteHeadBranch(dir, remote string) (string, error) {
	ref := fmt.Sprintf("refs/remotes/%s/HEAD", remote)
	cmd := exec.Command("git", "symbolic-ref", "--short", ref)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	fullRef := strings.TrimSpace(out.String()) // e.g. "origin/main"
	prefix := remote + "/"
	if strings.HasPrefix(fullRef, prefix) {
		return strings.TrimPrefix(fullRef, prefix), nil
	}
	return fullRef, nil
}

func getAnyRemoteHeadBranch(dir string) (string, error) {
	cmd := exec.Command("git", "symbolic-ref", "--short", "--glob=refs/remotes/*/HEAD")
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) > 0 && lines[0] != "" {
		parts := strings.SplitN(lines[0], "/", 2)
		if len(parts) == 2 {
			return parts[1], nil
		}
		return lines[0], nil
	}
	return "", errors.New("no remote HEAD ref found")
}

func getCurrentBranch(dir string) (string, error) {
	cmd := exec.Command("git", "symbolic-ref", "--short", "HEAD")
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	branch := strings.TrimSpace(out.String())
	if branch != "" {
		return branch, nil
	}
	return "", errors.New("detached HEAD or unknown branch")
}

func getFallbackLocalBranch(dir string) (string, error) {
	cmd := exec.Command("git", "branch", "--format=%(refname:short)")
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	branches := strings.Split(strings.TrimSpace(out.String()), "\n")
	branchMap := make(map[string]bool)
	for _, b := range branches {
		branchMap[strings.TrimSpace(b)] = true
	}

	for _, candidate := range []string{"main", "master", "trunk", "development"} {
		if branchMap[candidate] {
			return candidate, nil
		}
	}
	if len(branches) > 0 && branches[0] != "" {
		return branches[0], nil
	}
	return "", errors.New("no local branches found")
}
