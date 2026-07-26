package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func createTestRepoWithHistory(t *testing.T, defaultBranch string) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "calavera-calver-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@example.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\nOutput: %s", args, err, out)
		}
	}

	runGit("init", "-b", defaultBranch)
	runGit("config", "user.name", "Test")
	runGit("config", "user.email", "test@example.com")

	// Commit 1
	dummyFile := filepath.Join(dir, "README.md")
	os.WriteFile(dummyFile, []byte("commit 1"), 0644)
	runGit("add", ".")
	runGit("commit", "-m", "commit 1")

	// Commit 2
	os.WriteFile(dummyFile, []byte("commit 2"), 0644)
	runGit("add", ".")
	runGit("commit", "-m", "commit 2")

	return dir
}

func TestGetCalVerOnDefaultBranchNoTag(t *testing.T) {
	repoDir := createTestRepoWithHistory(t, "main")
	defer os.RemoveAll(repoDir)

	fixedTime := time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)
	res, err := GetCalVer(repoDir, fixedTime)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedVersion := "2026.07.2"
	if res.Version != expectedVersion {
		t.Errorf("expected version %q, got %q", expectedVersion, res.Version)
	}
	if res.DefaultCount != 2 {
		t.Errorf("expected DefaultCount 2, got %d", res.DefaultCount)
	}
	if !res.IsDefault {
		t.Errorf("expected IsDefault true, got false")
	}
}

func TestGetCalVerWithExplicitCalVerTag(t *testing.T) {
	repoDir := createTestRepoWithHistory(t, "main")
	defer os.RemoveAll(repoDir)

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\nOutput: %s", args, err, out)
		}
	}

	// Main branch is tagged with 2026.07.12
	runGit("tag", "2026.07.12")

	// Checkout feature branch calver-logic
	runGit("checkout", "-b", "calver-logic")

	// Add 1 commit on feature branch
	dummyFile := filepath.Join(repoDir, "feature.txt")
	os.WriteFile(dummyFile, []byte("new feature"), 0644)
	runGit("add", ".")
	runGit("commit", "-m", "feature commit 1")

	fixedTime := time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)
	res, err := GetCalVer(repoDir, fixedTime)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be 2026.07.12-calver-logic.1
	expectedVersion := "2026.07.12-calver-logic.1"
	if res.Version != expectedVersion {
		t.Errorf("expected version %q, got %q", expectedVersion, res.Version)
	}
	if res.DefaultCount != 12 {
		t.Errorf("expected DefaultCount 12, got %d", res.DefaultCount)
	}
	if res.BranchCount != 1 {
		t.Errorf("expected BranchCount 1, got %d", res.BranchCount)
	}
}

func TestGetCalVerOnFeatureBranch(t *testing.T) {
	repoDir := createTestRepoWithHistory(t, "main")
	defer os.RemoveAll(repoDir)

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\nOutput: %s", args, err, out)
		}
	}

	// Create tag on main
	runGit("tag", "v0.5.0") // main tag has baseline 0

	// Create and checkout feature branch
	runGit("checkout", "-b", "feature/login")

	// Add 3 commits on feature branch
	dummyFile := filepath.Join(repoDir, "feature.txt")
	for i := 1; i <= 3; i++ {
		os.WriteFile(dummyFile, []byte(string(rune('0'+i))), 0644)
		runGit("add", ".")
		runGit("commit", "-m", "feature commit")
	}

	fixedTime := time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)
	res, err := GetCalVer(repoDir, fixedTime)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedVersion := "2026.07.0-feature-login.3"
	if res.Version != expectedVersion {
		t.Errorf("expected version %q, got %q", expectedVersion, res.Version)
	}
	if res.DefaultCount != 0 {
		t.Errorf("expected DefaultCount 0, got %d", res.DefaultCount)
	}
	if res.BranchCount != 3 {
		t.Errorf("expected BranchCount 3, got %d", res.BranchCount)
	}
}

func TestParseTagBaseline(t *testing.T) {
	tests := []struct {
		tag      string
		expected int
	}{
		{"2026.07.12", 12},
		{"v2026.07.12", 12},
		{"2026.07.0", 0},
		{"v1.0.5", 5},
		{"v12", 12},
		{"no-numbers", -1},
	}

	for _, tt := range tests {
		got := parseTagBaseline(tt.tag)
		if got != tt.expected {
			t.Errorf("parseTagBaseline(%q) = %d, expected %d", tt.tag, got, tt.expected)
		}
	}
}
