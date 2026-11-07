package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestIssue2CLIProcessHelper(t *testing.T) {
	if os.Getenv("ISSUE2_CLI_PROCESS") != "1" {
		return
	}
	if value := os.Getenv("ISSUE2_CLI_FILE_LIMIT"); value != "" {
		limit, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		signal.Ignore(syscall.SIGXFSZ)
		if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &syscall.Rlimit{Cur: limit, Max: limit}); err != nil {
			t.Fatal(err)
		}
	}
	if os.Getenv("ISSUE2_CLI_PRIVATE_UMASK") == "1" {
		syscall.Umask(0o077)
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{"scribe"}, os.Args[i+1:]...)
			main()
			os.Exit(0)
		}
	}
	t.Fatal("missing CLI arguments")
}

func issue2RunCLI(t *testing.T, root string, args ...string) (string, int) {
	t.Helper()
	return issue2RunCLIEnv(t, root, nil, args...)
}

func issue2RunCLIEnv(t *testing.T, root string, env []string, args ...string) (string, int) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	argv := append([]string{"-test.run=^TestIssue2CLIProcessHelper$", "--", "-C", root}, args...)
	cmd := exec.CommandContext(ctx, exe, argv...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "ISSUE2_CLI_PROCESS=1")
	cmd.Env = append(cmd.Env, env...)
	guard := t.TempDir()
	for _, tool := range []string{"claude", "codex", "qmd", "ccrider"} {
		if err := os.WriteFile(filepath.Join(guard, tool), []byte("#!/bin/sh\necho forbidden-external-tool >&2\nexit 97\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	cmd.Env = append(cmd.Env, "PATH="+guard+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("CLI did not complete within bound: %v\n%s", ctx.Err(), out)
	}
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return string(out), exit.ExitCode()
		}
		t.Fatal(err)
	}
	return string(out), 0
}

func issue2KBBytes(t *testing.T, root string) map[string]string {
	t.Helper()
	state := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == ".git" {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			state[rel] = "directory"
		} else if entry.Type()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			state[rel] = "symlink:" + target
		} else {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			state[rel] = fmt.Sprintf("%x", sha256.Sum256(data))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func TestIssue2RollingPreviewRepresentative(t *testing.T) {
	root := testKB(t, "domains: [acme]\n")
	writeKBFile(t, root, "projects/acme/learnings.md", "---\nrolling: true\n---\n"+issue2RollingBody)
	writeKBFile(t, root, "wiki/topic.md.md", "Unrelated duplicate artifact stays untouched.\n")
	before := issue2KBBytes(t, root)
	out, exit := issue2RunCLI(t, root, "lint", "--fix", "--rolling-only", "--dry-run")
	if exit != 0 || !strings.Contains(out, "would repair projects/acme/learnings.md") || !strings.Contains(out, "scanned 1, would repair 1, unchanged 0, unresolved 0") || strings.Contains(out, "duplicates:") {
		t.Fatalf("preview did not report the rolling proposal: exit=%d\n%s", exit, out)
	}
	if !reflect.DeepEqual(before, issue2KBBytes(t, root)) {
		t.Fatal("preview changed KB bytes, file set, or run records")
	}
	t.Logf("real CLI preview output (exit %d):\n%s", exit, out)
}

func TestIssue2RollingPreviewFullScope(t *testing.T) {
	root := testKB(t, "domains: [acme]\nmeta:\n  rolling_targets: [risk-notes]\n")
	for _, path := range []string{"projects/untouched/decisions-log-archive-2025.md", "projects/acme/risk-notes.md"} {
		writeKBFile(t, root, path, issue2RollingBody)
	}
	for _, path := range []string{"patterns/personal/évidence.md", "wiki/_memory.md"} {
		writeKBFile(t, root, path, "---\nrolling: true\n---\n"+issue2RollingBody)
	}
	writeKBFile(t, root, "projects/acme/learnings.md", issue2ValidRolling())
	writeKBFile(t, root, "projects/acme/decisions-log.md", "---\nrolling: false\n---\n"+issue2RollingBody)
	writeKBFile(t, root, "patterns/personal/broken.md", "---\nrolling: true\ntags: [\n---\n"+issue2RollingBody)
	for _, path := range []string{"raw/outside.md", "wiki/ordinary.md", "wiki/topic.md.md", "projects/acme/not-an-archive-2025.md"} {
		writeKBFile(t, root, path, "Unrelated article.\n")
	}
	before := issue2KBBytes(t, root)
	out, exit := issue2RunCLI(t, root, "lint", "--fix", "--rolling-only", "--dry-run")
	if exit == 0 || !strings.Contains(out, "scanned 7, would repair 4, unchanged 1, unresolved 2") || !strings.Contains(out, "would repair wiki/_memory.md") || !strings.Contains(out, "would repair projects/untouched/decisions-log-archive-2025.md") || !strings.Contains(out, "unresolved patterns/personal/broken.md") || !strings.Contains(out, "rolling marker conflicts") {
		t.Fatalf("incomplete rolling selection or wrong mixed results: exit=%d\n%s", exit, out)
	}
	if !reflect.DeepEqual(before, issue2KBBytes(t, root)) {
		t.Fatal("mixed preview changed KB content")
	}
}

func TestIssue2RollingPreviewSelectors(t *testing.T) {
	root := testKB(t, "domains: [acme]\n")
	for _, path := range []string{"projects/acme/learnings.md", "projects/acme/decisions-log.md"} {
		writeKBFile(t, root, path, "---\nrolling: true\n---\n"+issue2RollingBody)
	}
	writeKBFile(t, root, "wiki/ordinary.md", "Ordinary article.\n")
	writeKBFile(t, root, "wiki/marked.txt", "---\nrolling: true\n---\n"+issue2RollingBody)
	mustGit(t, root, "init", "-q")
	mustGit(t, root, "add", ".")
	mustGit(t, root, "commit", "-qm", "seed")
	writeKBFile(t, root, "projects/acme/learnings.md", "---\nrolling: true\n---\n"+issue2RollingBody+"New bytes.\n")
	writeKBFile(t, root, "projects/acme/learnings-archive-2025.md", issue2RollingBody)
	before := issue2KBBytes(t, root)
	for _, tc := range []struct {
		name  string
		args  []string
		count string
	}{
		{"explicit", []string{"projects/acme/decisions-log.md", "projects/acme/decisions-log.md"}, "scanned 1, would repair 1, unchanged 0, unresolved 0"},
		{"ordinary-filter", []string{"wiki/ordinary.md"}, "scanned 0, would repair 0, unchanged 0, unresolved 0"},
		{"markdown-filter", []string{"wiki/marked.txt"}, "scanned 0, would repair 0, unchanged 0, unresolved 0"},
		{"changed", []string{"--changed"}, "scanned 2, would repair 2, unchanged 0, unresolved 0"},
		{"changed-precedence", []string{"--changed", "projects/acme/decisions-log.md"}, "scanned 2, would repair 2, unchanged 0, unresolved 0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"lint", "--fix", "--rolling-only", "--dry-run"}, tc.args...)
			out, exit := issue2RunCLI(t, root, args...)
			if exit != 0 || !strings.Contains(out, tc.count) {
				t.Fatalf("wrong selector scope: exit=%d\n%s", exit, out)
			}
			if !reflect.DeepEqual(before, issue2KBBytes(t, root)) {
				t.Fatal("selector preview changed content")
			}
		})
	}
}

func TestIssue2RollingPreviewLockOwnership(t *testing.T) {
	root := testKB(t, "domains: [acme]\n")
	writeKBFile(t, root, "projects/acme/learnings.md", issue2RollingBody)
	before := issue2KBBytes(t, root)
	lockDir := loadConfig(root).LockDir
	path := lockPathFor(lockDir, "sync", root)
	lf, ok, err := acquireOperationLock(path, "rolling-only preview")
	if err != nil || !ok {
		t.Fatalf("operation lock failed: %v", err)
	}
	defer func() { releaseLock(lf) }()
	info, _ := os.Stat(path)
	ownerBefore, _ := os.ReadFile(path)
	alias := filepath.Join(t.TempDir(), "selected-kb")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	out, exit := issue2RunCLI(t, alias, "lint", "--fix", "--rolling-only", "--dry-run")
	ownerAfter, _ := os.ReadFile(path)
	if exit == 0 || !strings.Contains(out, "rolling-only preview") || string(ownerBefore) != string(ownerAfter) {
		t.Fatalf("root alias bypassed lock or contender changed holder details: exit=%d\n%s", exit, out)
	}
	other := t.TempDir()
	writeKBFile(t, other, "scribe.yaml", "lock_dir: "+lockDir+"\ndomains: [acme]\n")
	writeKBFile(t, other, "projects/acme/learnings.md", issue2RollingBody)
	otherBefore := issue2KBBytes(t, other)
	out, exit = issue2RunCLI(t, other, "lint", "--fix", "--rolling-only", "--dry-run")
	if exit != 0 || !reflect.DeepEqual(otherBefore, issue2KBBytes(t, other)) {
		t.Fatalf("unrelated KB contended on the same lock directory: exit=%d\n%s", exit, out)
	}
	releaseLock(lf)
	lf = nil
	lf, ok, err = acquireLock(path)
	if err != nil || !ok {
		t.Fatalf("unlabeled existing lock caller failed: %v", err)
	}
	out, exit = issue2RunCLI(t, root, "lint", "--fix", "--rolling-only", "--dry-run")
	if exit == 0 || !strings.Contains(out, "holder details unavailable") || strings.Contains(out, "rolling-only preview (pid") {
		t.Fatalf("busy result reused a released holder's identity: exit=%d\n%s", exit, out)
	}
	releaseLock(lf)
	lf = nil
	out, exit = issue2RunCLI(t, root, "lint", "--fix", "--rolling-only", "--dry-run")
	afterInfo, _ := os.Stat(path)
	if exit != 0 || !os.SameFile(info, afterInfo) || !reflect.DeepEqual(before, issue2KBBytes(t, root)) {
		t.Fatalf("preview failed after release, replaced lock inode, or wrote content: exit=%d\n%s", exit, out)
	}
}

func TestIssue2RollingPreviewUnresolvedMetadata(t *testing.T) {
	for _, tc := range []struct{ name, fields, body, cause string }{
		{"confidence", "confidence: certain\n", issue2RollingBody, "invalid confidence"},
		{"no-dates", "", "No dated entries.\n", "cannot recover created"},
		{"code-only-dates", "", "```markdown\n## 2000-01-01 | Example\nsession:example\n```\nUndated real entry.\n", "cannot recover created"},
		{"contradictory", "created: 2026-11-02\n", issue2RollingBody, "before created"},
		{"invalid-calendar", "created: 2026-02-30\n", issue2RollingBody, "valid calendar date"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := testKB(t, "domains: [acme]\n")
			writeKBFile(t, root, "projects/acme/learnings.md", "---\nrolling: true\n"+tc.fields+"---\n"+tc.body)
			before := issue2KBBytes(t, root)
			out, exit := issue2RunCLI(t, root, "lint", "--fix", "--rolling-only", "--dry-run")
			if exit == 0 || !strings.Contains(out, tc.cause) || !strings.Contains(out, "scanned 1, would repair 0, unchanged 0, unresolved 1") || !reflect.DeepEqual(before, issue2KBBytes(t, root)) {
				t.Fatalf("unresolved metadata was hidden or changed: exit=%d\n%s", exit, out)
			}
		})
	}
}

func TestIssue2RollingPreviewHelpAndOrdinaryLint(t *testing.T) {
	root := testKB(t, "domains: [acme]\n")
	writeKBFile(t, root, "wiki/ordinary.md", "---\ntitle: Ordinary\ntype: project\n---\nPreserve preview whitespace.  \n")
	before := issue2KBBytes(t, root)
	out, exit := issue2RunCLI(t, root, "lint", "--help")
	if exit != 0 || !strings.Contains(out, "--rolling-only") || !strings.Contains(out, "unresolved") || !strings.Contains(strings.Join(strings.Fields(out), " "), "--dry-run previews") {
		t.Fatalf("rolling preview help is missing its boundary: exit=%d\n%s", exit, out)
	}
	out, exit = issue2RunCLI(t, root, "lint", "--fix", "--dry-run", "wiki/ordinary.md")
	if exit != 0 || !strings.Contains(out, "would fix 1 file(s)") || strings.Contains(out, "would repair") || !reflect.DeepEqual(before, issue2KBBytes(t, root)) {
		t.Fatalf("ordinary lint repair preview changed behavior: exit=%d\n%s", exit, out)
	}
}

func TestIssue2RollingPreviewContainment(t *testing.T) {
	for _, variant := range []string{"outside", "leaf-alias", "directory-alias", "root-alias"} {
		t.Run(variant, func(t *testing.T) {
			root := testKB(t, "domains: [acme]\n")
			outside := testKB(t, "domains: [acme]\n")
			writeKBFile(t, outside, "learnings.md", "---\nrolling: true\n---\n"+issue2RollingBody)
			selectedRoot := root
			args := []string{"lint", "--fix", "--rolling-only", "--dry-run"}
			switch variant {
			case "outside":
				args = append(args, filepath.Join(outside, "learnings.md"))
			case "leaf-alias":
				writeKBFile(t, root, "projects/acme/.keep", "")
				if err := os.Symlink(filepath.Join(outside, "learnings.md"), filepath.Join(root, "projects/acme/learnings.md")); err != nil {
					t.Fatal(err)
				}
			case "directory-alias":
				writeKBFile(t, root, "projects/.keep", "")
				if err := os.Symlink(outside, filepath.Join(root, "projects/acme")); err != nil {
					t.Fatal(err)
				}
			case "root-alias":
				writeKBFile(t, root, "projects/acme/learnings.md", issue2RollingBody)
				selectedRoot = filepath.Join(t.TempDir(), "selected-kb")
				if err := os.Symlink(root, selectedRoot); err != nil {
					t.Fatal(err)
				}
				args = append(args, "projects/acme/learnings.md")
			}
			before, outsideBefore := issue2KBBytes(t, root), issue2KBBytes(t, outside)
			out, exit := issue2RunCLI(t, selectedRoot, args...)
			if variant == "root-alias" {
				if exit != 0 || !strings.Contains(out, "would repair projects/acme/learnings.md") {
					t.Fatalf("stable selected root alias rejected: exit=%d\n%s", exit, out)
				}
			} else if exit == 0 || !strings.Contains(out, "unresolved") {
				t.Fatalf("unsafe selector or alias ignored: exit=%d\n%s", exit, out)
			}
			if !reflect.DeepEqual(before, issue2KBBytes(t, root)) || !reflect.DeepEqual(outsideBefore, issue2KBBytes(t, outside)) {
				t.Fatal("preview crossed containment or changed bytes")
			}
		})
	}
}

func TestIssue2RollingPreviewFlagsAndBusy(t *testing.T) {
	root := testKB(t, "domains: [acme]\n")
	writeKBFile(t, root, "projects/acme/learnings.md", issue2RollingBody)
	writeKBFile(t, root, "wiki/topic.md.md", "Duplicate artifact.\n")
	before := issue2KBBytes(t, root)
	for _, tc := range []struct {
		name    string
		args    []string
		message string
	}{
		{"requires-fix", []string{"--rolling-only", "--dry-run"}, "requires --fix"},
		{"no-duplicate-scan", []string{"--fix", "--rolling-only", "--dry-run", "--duplicates"}, "cannot be combined"},
		{"no-model-operation", []string{"--fix", "--rolling-only", "--dry-run", "--contradictions"}, "cannot be combined"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, exit := issue2RunCLI(t, root, append([]string{"lint"}, tc.args...)...)
			if exit == 0 || !strings.Contains(out, tc.message) || strings.Contains(out, "forbidden-external-tool") || !reflect.DeepEqual(before, issue2KBBytes(t, root)) {
				t.Fatalf("rolling-only flags escaped containment: exit=%d\n%s", exit, out)
			}
		})
	}
	lockPath := lockPathFor(loadConfig(root).LockDir, "sync", root)
	lf, ok, err := acquireLock(lockPath)
	if err != nil || !ok {
		t.Fatalf("fixture lock unavailable: %v", err)
	}
	defer releaseLock(lf)
	if _, err := fmt.Fprintf(lf, "{\"operation\":\"sync\",\"pid\":%d}\n", os.Getpid()); err != nil {
		t.Fatal(err)
	}
	out, exit := issue2RunCLI(t, root, "lint", "--fix", "--rolling-only", "--dry-run")
	if exit == 0 || !strings.Contains(out, "busy") || !strings.Contains(out, "sync") || !strings.Contains(out, fmt.Sprint(os.Getpid())) || !reflect.DeepEqual(before, issue2KBBytes(t, root)) {
		t.Fatalf("busy preview did not identify active sync without writes: exit=%d\n%s", exit, out)
	}
}
