
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func run(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("\nERROR: %s %s\n%s\n",
			name, strings.Join(args, " "), out)
		os.Exit(1)
	}
	return strings.TrimSpace(string(out))
}

func main() {
	fmt.Println("=== GitHub Pull Shark Automator ===")

	// Verify GitHub CLI authentication.
	run("gh", "auth", "status")

	// Work only inside an existing GitHub repository.
	repo := run("gh", "repo", "view",
		"--json", "nameWithOwner",
		"--jq", ".nameWithOwner")

	base := run("gh", "repo", "view",
		"--json", "defaultBranchRef",
		"--jq", ".defaultBranchRef.name")

	current := run("git", "branch", "--show-current")
	status := run("git", "status", "--porcelain")

	if status != "" {
		fmt.Println("Working tree must be clean.")
		os.Exit(1)
	}
	if current != base {
		fmt.Printf("Switch to %s first.\n", base)
		os.Exit(1)
	}

	fmt.Println("Repository:", repo)
	fmt.Println("Base branch:", base)

	// Synchronize before creating changes.
	run("git", "pull", "--ff-only", "origin", base)

	for i := 1; i <= 2; i++ {
		id := fmt.Sprintf("%d-%d",
			time.Now().UnixNano(), i)
		branch := "docs/achievement-" + id
		file := filepath.Join(
			"docs", "achievements", id+".md",
		)

		fmt.Printf("\n[%d/2] Creating PR...\n", i)

		run("git", "switch", "-c", branch)

		content := fmt.Sprintf(
			"# Documentation Update\n\n"+
				"Automated documentation example %d.\n",
			i,
		)

		if err := os.MkdirAll(
			filepath.Dir(file), 0755,
		); err != nil {
			panic(err)
		}
		if err := os.WriteFile(
			file, []byte(content), 0644,
		); err != nil {
			panic(err)
		}

		run("git", "add", file)
		run("git", "commit", "-m",
			"docs: add achievement example")
		run("git", "push", "-u", "origin", branch)

		pr := run("gh", "pr", "create",
			"--base", base,
			"--head", branch,
			"--title", "docs: add example "+id,
			"--body", "Add a documentation example.",
		)

		fmt.Println("PR:", pr)

		// Merge only if repository permissions allow it.
		run("gh", "pr", "merge", pr, "--squash")

		fmt.Println("Merged successfully!")

		run("git", "switch", base)
		run("git", "pull", "--ff-only", "origin", base)

		// Clean up the temporary local branch.
		run("git", "branch", "-d", branch)
	}

	fmt.Println("\n=== COMPLETED ===")
	fmt.Println("2 pull requests merged.")
	fmt.Println("Check GitHub Achievements later.")
}
