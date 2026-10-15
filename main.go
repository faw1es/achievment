
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func run(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Printf("\n[ERROR] %s %s\n%s\n",
			name, strings.Join(args, " "), output)
		os.Exit(1)
	}

	return strings.TrimSpace(string(output))
}

func syncBranch(base string) {
	fmt.Println("[SYNC] Fetching origin/" + base)

	run("git", "fetch", "origin", base)

	// Never force-reset or overwrite local commits.
	cmd := exec.Command(
		"git", "merge", "--ff-only", "origin/"+base,
	)

	output, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Println("\n[SYNC ERROR]")
		fmt.Println(strings.TrimSpace(string(output)))
		fmt.Println(
			"Local and remote history may have diverged.",
		)
		fmt.Println(
			"Inspect git status and git log before continuing.",
		)
		os.Exit(1)
	}

	fmt.Println("[OK] Branch synchronized")
}

func main() {
	count := flag.Int(
		"count", 1,
		"Number of pull requests to create",
	)
	flag.Parse()

	if *count < 1 || *count > 2 {
		fmt.Println("Count must be 1 or 2")
		os.Exit(1)
	}

	fmt.Println("=== GitHub Achievement CLI ===")

	// Check authentication.
	run("gh", "auth", "status")

	repo := run(
		"gh", "repo", "view",
		"--json", "nameWithOwner",
		"--jq", ".nameWithOwner",
	)

	base := run(
		"gh", "repo", "view",
		"--json", "defaultBranchRef",
		"--jq", ".defaultBranchRef.name",
	)

	current := run(
		"git", "branch", "--show-current",
	)

	status := run(
		"git", "status", "--porcelain",
	)

	if status != "" {
		fmt.Println(
			"[STOP] Working tree is not clean:",
		)
		fmt.Println(status)
		os.Exit(1)
	}

	if current != base {
		fmt.Printf(
			"[STOP] Switch to %s first\n", base,
		)
		os.Exit(1)
	}

	fmt.Println("Repository:", repo)
	fmt.Println("Base branch:", base)

	syncBranch(base)

	for i := 1; i <= *count; i++ {
		id := fmt.Sprintf(
			"%d-%d", time.Now().UnixNano(), i,
		)

		branch := "docs/achievement-" + id

		file := filepath.Join(
			"docs", "achievements", id+".md",
		)

		fmt.Printf(
			"\n[%d/%d] Creating pull request\n",
			i, *count,
		)

		run("git", "switch", "-c", branch)

		err := os.MkdirAll(
			filepath.Dir(file), 0755,
		)
		if err != nil {
			panic(err)
		}

		content := fmt.Sprintf(
			"# Documentation Example\n\n"+
				"Example documentation update %d.\n",
			i,
		)

		err = os.WriteFile(
			file, []byte(content), 0644,
		)
		if err != nil {
			panic(err)
		}

		run("git", "add", file)

		run(
			"git", "commit",
			"-m", "docs: add documentation example",
		)

		run(
			"git", "push", "-u", "origin", branch,
		)

		pr := run(
			"gh", "pr", "create",
			"--base", base,
			"--head", branch,
			"--title", "docs: update "+id,
			"--body", "Add a documentation example.",
		)

		fmt.Println("[PR]", pr)

		// Standard merge, preserving commit history.
		run(
			"gh", "pr", "merge", pr, "--merge",
		)

		fmt.Println("[OK] PR merged")

		run("git", "switch", base)
		syncBranch(base)

		// Delete local temporary branch after merge.
		run("git", "branch", "-d", branch)

		fmt.Println("[OK] Completed PR", i)
	}

	fmt.Println("\n=======================")
	fmt.Println("AUTOMATION COMPLETED")
	fmt.Println("=======================")
	fmt.Printf("Merged PRs this run: %d\n", *count)
	fmt.Println("Check GitHub Achievements.")
}
