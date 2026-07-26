package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "help" || os.Args[1] == "-h" || os.Args[1] == "--help") {
		printUsage()
		os.Exit(0)
	}

	subcommand := "calver"
	argOffset := 1

	if len(os.Args) > 1 {
		firstArg := os.Args[1]
		if firstArg == "calver" || firstArg == "default-branch" {
			subcommand = firstArg
			argOffset = 2
		}
	}

	fs := flag.NewFlagSet(subcommand, flag.ExitOnError)
	quietFlag := fs.Bool("quiet", false, "Print only the version/branch string without additional info")
	fs.BoolVar(quietFlag, "q", false, "Print only the version/branch string without additional info (shorthand)")
	dateFlag := fs.String("date", "", "Override date in YYYY.MM format (e.g. 2026.07)")

	if err := fs.Parse(os.Args[argOffset:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	repoPath := "."
	if fs.NArg() > 0 {
		repoPath = fs.Arg(0)
	}

	switch subcommand {
	case "default-branch":
		branch, err := GetDefaultBranch(repoPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if *quietFlag {
			fmt.Println(branch)
		} else {
			fmt.Printf("Default branch for '%s': %s\n", repoPath, branch)
		}

	case "calver":
		var now time.Time
		if *dateFlag != "" {
			parsed, err := time.Parse("2006.01", *dateFlag)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: invalid date format '%s', expected YYYY.MM\n", *dateFlag)
				os.Exit(1)
			}
			now = parsed
		} else {
			now = time.Now()
		}

		res, err := GetCalVer(repoPath, now)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if *quietFlag {
			fmt.Println(res.Version)
		} else {
			if res.IsDefault {
				fmt.Printf("CalVer (%s, default branch '%s'): %s\n", repoPath, res.BranchName, res.Version)
			} else {
				fmt.Printf("CalVer (%s, feature branch '%s'): %s\n", repoPath, res.BranchName, res.Version)
			}
		}
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Calavera - Calendar Versioning & Git Repository Utility\n\n")
	fmt.Fprintf(os.Stderr, "Usage:\n")
	fmt.Fprintf(os.Stderr, "  calavera [calver] [options] [path_to_repo]\n")
	fmt.Fprintf(os.Stderr, "  calavera default-branch [options] [path_to_repo]\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  calver          Output Calendar Version (yyyy.mm.[commitCounts] or yyyy.mm.[commitCounts]-[branch].[branchCommitCount]) [default]\n")
	fmt.Fprintf(os.Stderr, "  default-branch  Output the default branch name of the repository\n\n")
	fmt.Fprintf(os.Stderr, "Options:\n")
	fmt.Fprintf(os.Stderr, "  -q, --quiet     Print raw output string only\n")
	fmt.Fprintf(os.Stderr, "  --date string   Override date format (YYYY.MM)\n")
}
