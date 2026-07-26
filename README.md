# Calavera

A lightweight CLI tool written in Go to generate Calendar Versions (CalVer) and detect default branch metadata from a local Git repository.

## CalVer Version Format

- **Default Branch**: `yyyy.mm.[commitCounts]`
  - `yyyy.mm`: Current year and month (e.g., `2026.07`).
  - `[commitCounts]`: Number of commits on the default branch since the latest Git tag (or total commits if no tag exists).
  - *Example*: `2026.07.3`

- **Feature / Non-Default Branch**: `yyyy.mm.[commitCounts]-[branchName].[commitCount]`
  - `[commitCounts]`: Number of commits on the default branch up to the merge-base (since the latest tag on default branch).
  - `[branchName]`: Sanitized name of the feature branch (e.g. `feature/login` -> `feature-login`).
  - `.[commitCount]`: Number of commits on the feature branch since branching off the default branch.
  - *Example*: `2026.07.1-feature-login.3`

---

## Usage

```bash
# Build the binary
go build -o calavera

# Output CalVer for current repository
./calavera

# Output CalVer for a specific repository directory
./calavera /path/to/repo

# Output raw CalVer string only (ideal for CI/CD and scripts)
./calavera -q

# Query default branch name directly
./calavera default-branch
./calavera default-branch -q /path/to/repo

# Override date string (e.g., for testing)
./calavera --date 2026.07
```

## Running Tests

```bash
go test -v ./...
```
