package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func createTestRepo(t *testing.T, initialBranch string) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "calavera-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	cmd := exec.Command("git", "init", "-b", initialBranch)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		// Fallback for older git versions that don't support -b
		cmdInit := exec.Command("git", "init")
		cmdInit.Dir = dir
		if err := cmdInit.Run(); err != nil {
			t.Fatalf("git init failed: %v", err)
		}
		cmdCheckout := exec.Command("git", "checkout", "-b", initialBranch)
		cmdCheckout.Dir = dir
		_ = cmdCheckout.Run()
	}

	// Make an initial commit so HEAD reference exists
	dummyFile := filepath.Join(dir, "README.md")
	if err := os.WriteFile(dummyFile, []byte("# Test Repo"), 0644); err != nil {
		t.Fatalf("failed to write dummy file: %v", err)
	}

	cmdAdd := exec.Command("git", "add", ".")
	cmdAdd.Dir = dir
	_ = cmdAdd.Run()

	cmdCommit := exec.Command("git", "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", "initial commit")
	cmdCommit.Dir = dir
	if err := cmdCommit.Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	return dir
}

func TestGetDefaultBranch(t *testing.T) {
	testCases := []string{"main", "master", "develop"}

	for _, expectedBranch := range testCases {
		t.Run("Branch_"+expectedBranch, func(t *testing.T) {
			repoDir := createTestRepo(t, expectedBranch)
			defer os.RemoveAll(repoDir)

			branch, err := GetDefaultBranch(repoDir)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if branch != expectedBranch {
				t.Errorf("expected branch %q, got %q", expectedBranch, branch)
			}
		})
	}
}

func TestNonGitDir(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "calavera-nongit-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	_, err = GetDefaultBranch(tempDir)
	if err == nil {
		t.Fatalf("expected error for non-git repository, got nil")
	}
}
