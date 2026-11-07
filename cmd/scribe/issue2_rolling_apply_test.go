package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func issue2RepairContent(t *testing.T, root string) map[string]string {
	t.Helper()
	state := issue2KBBytes(t, root)
	for path := range state {
		if path == "output" || path == "output/runs" || strings.HasPrefix(filepath.ToSlash(path), "output/runs/") {
			delete(state, path)
		}
	}
	return state
}

func TestIssue2RollingApplyRepresentative(t *testing.T) {
	root := testKB(t, "domains: [acme]\n")
	path := "projects/acme/learnings.md"
	body := "\n" + issue2RollingBody
	original := "---\ncontributor: Curated author\ncustom_field: retained\nconfidence: high\n---\n" + body
	writeKBFile(t, root, path, original)
	if err := os.Chmod(filepath.Join(root, path), 0o600); err != nil {
		t.Fatal(err)
	}
	writeKBFile(t, root, "wiki/topic.md.md", "Unrelated duplicate.  \n")
	writeKBFile(t, root, "wiki/{{PLACEHOLDER}}.md", "Unrelated placeholder.\n")
	writeKBFile(t, root, "wiki/_sessions_log.json", `{"processed":{"original":"done"}}`)
	before := issue2RepairContent(t, root)
	infoBefore, _ := os.Stat(filepath.Join(root, path))
	out, exit := issue2RunCLI(t, root, "lint", "--fix", "--rolling-only")
	if exit != 0 || !strings.Contains(out, "repaired "+path) || !strings.Contains(out, "scanned 1, repaired 1, unchanged 0, unresolved 0") {
		t.Fatalf("apply did not persist a repair: exit=%d\n%s", exit, out)
	}
	data := []byte(readKBFile(t, root, path))
	raw, err := parseFrontmatterRaw(data)
	span, spanErr := frontmatterSpan(string(data))
	infoAfter, statErr := os.Stat(filepath.Join(root, path))
	if err != nil || spanErr != nil || statErr != nil || len(validateArticleContent(root, data)) != 0 || !bytes.Equal(data[span.BodyStart:], []byte(body)) || raw["contributor"] != "Curated author" || raw["custom_field"] != "retained" || raw["confidence"] != "high" || raw["rolling"] != true || raw["title"] != "Acme Learnings" || stringFromAny(raw["created"]) != "2026-10-01" || stringFromAny(raw["updated"]) != "2026-11-01" {
		t.Fatalf("persisted repair violates metadata/body contract: %v (%v %v %v)", raw, err, spanErr, statErr)
	}
	sources, ok := raw["sources"].([]any)
	if !ok || len(sources) != 1 || sources[0] != "session:issue2-session" || infoAfter.Mode().Perm() != 0o600 || os.SameFile(infoBefore, infoAfter) {
		t.Fatal("apply did not retain source/mode or use inode replacement")
	}
	after := issue2RepairContent(t, root)
	before[path] = after[path]
	if !reflect.DeepEqual(before, after) {
		t.Fatal("rolling apply changed unrelated articles, placeholders or processed log")
	}
	records, _ := filepath.Glob(filepath.Join(root, "output/runs/*.jsonl"))
	if len(records) == 0 {
		t.Fatal("real apply omitted the normal CLI run record")
	}
	t.Logf("real CLI apply output (exit %d):\n%s\npersisted file:\n%s", exit, out, data)
}

func TestIssue2RollingApplyMixedAndStable(t *testing.T) {
	root := testKB(t, "domains: [acme]\nmeta:\n  rolling_targets: [risk-notes]\n")
	for _, path := range []string{"projects/untouched/decisions-log-archive-2025.md", "projects/acme/risk-notes.md"} {
		writeKBFile(t, root, path, issue2RollingBody)
	}
	for _, path := range []string{"patterns/personal/évidence.md", "wiki/_memory.md"} {
		writeKBFile(t, root, path, "---\nrolling: true\n---\n"+issue2RollingBody)
	}
	writeKBFile(t, root, "projects/acme/learnings.md", issue2ValidRolling())
	invalid := "---\nrolling: false\n---\n" + issue2RollingBody
	writeKBFile(t, root, "projects/acme/decisions-log.md", invalid)
	writeKBFile(t, root, "wiki/topic.md.md", "Unrelated duplicate artifact.\n")
	writeKBFile(t, root, "wiki/{{PLACEHOLDER}}.md", "Unrelated placeholder.\n")
	writeKBFile(t, root, "raw/source.md", "---\nrolling: true\n---\nRaw stays unchanged.\n")
	before := issue2RepairContent(t, root)
	out, exit := issue2RunCLI(t, root, "lint", "--fix", "--rolling-only")
	if exit == 0 || !strings.Contains(out, "scanned 6, repaired 4, unchanged 1, unresolved 1") || !strings.Contains(out, "rolling marker conflicts") {
		t.Fatalf("wrong independent mixed outcomes: exit=%d\n%s", exit, out)
	}
	for _, path := range []string{"projects/untouched/decisions-log-archive-2025.md", "projects/acme/risk-notes.md", "patterns/personal/évidence.md", "wiki/_memory.md"} {
		data := []byte(readKBFile(t, root, path))
		span, err := frontmatterSpan(string(data))
		if err != nil || len(validateArticleContent(root, data)) != 0 || !bytes.Equal(data[span.BodyStart:], []byte(issue2RollingBody)) {
			t.Fatalf("sibling not repaired with preserved body: %s (%v)", path, err)
		}
		raw, _ := parseFrontmatterRaw(data)
		if path == "patterns/personal/évidence.md" && (raw["title"] != "Personal Évidence" || raw["domain"] != "personal") || path == "projects/untouched/decisions-log-archive-2025.md" && (raw["title"] != "Untouched Decisions Log Archive 2025" || raw["domain"] != "general") {
			t.Fatalf("wrong title/domain defaults: %s %v", path, raw)
		}
	}
	after := issue2RepairContent(t, root)
	for _, path := range []string{"projects/acme/learnings.md", "projects/acme/decisions-log.md", "wiki/topic.md.md", "wiki/{{PLACEHOLDER}}.md", "raw/source.md"} {
		if before[path] != after[path] {
			t.Fatalf("mixed apply changed an unresolved/valid/unrelated file: %s", path)
		}
	}
	out, exit = issue2RunCLI(t, root, "lint", "--fix", "--rolling-only")
	if exit == 0 || !strings.Contains(out, "scanned 6, repaired 0, unchanged 5, unresolved 1") || !reflect.DeepEqual(after, issue2RepairContent(t, root)) {
		t.Fatalf("repeated mixed apply churned content: exit=%d\n%s", exit, out)
	}
	out, exit = issue2RunCLI(t, root, "lint", "--fix", "--rolling-only", "--dry-run")
	if exit == 0 || !strings.Contains(out, "scanned 6, would repair 0, unchanged 5, unresolved 1") || !reflect.DeepEqual(after, issue2RepairContent(t, root)) {
		t.Fatalf("preview diverges after apply: exit=%d\n%s", exit, out)
	}
}

func TestIssue2RollingApplyWriteFailure(t *testing.T) {
	root := testKB(t, "domains: [acme]\n")
	large := "---\nrolling: true\n---\n" + issue2RollingBody + strings.Repeat("Original large body.  \n", 1000)
	writeKBFile(t, root, "projects/acme/decisions-log.md", large)
	writeKBFile(t, root, "projects/acme/learnings.md", issue2RollingBody)
	before := issue2RepairContent(t, root)
	out, exit := issue2RunCLIEnv(t, root, []string{"ISSUE2_CLI_FILE_LIMIT=4096"}, "lint", "--fix", "--rolling-only")
	if exit == 0 || !strings.Contains(out, "scanned 2, repaired 1, unchanged 0, unresolved 1") || !strings.Contains(out, "write tmp") || !strings.Contains(out, "unresolved projects/acme/decisions-log.md") {
		t.Fatalf("real temporary-write fault did not remain independent: exit=%d\n%s", exit, out)
	}
	if readKBFile(t, root, "projects/acme/decisions-log.md") != large || len(validateFile(root, filepath.Join(root, "projects/acme/learnings.md"))) != 0 {
		t.Fatal("failed write changed original bytes or blocked smaller sibling repair")
	}
	after := issue2RepairContent(t, root)
	before["projects/acme/learnings.md"] = after["projects/acme/learnings.md"]
	if !reflect.DeepEqual(before, after) {
		t.Fatal("atomic write failure leaked a temporary file or changed unrelated bytes")
	}
	t.Logf("caught child-only write-limit output:\n%s", out)
}

func TestIssue2RollingApplyPreservesMode(t *testing.T) {
	root := testKB(t, "domains: [acme]\n")
	path := filepath.Join(root, "projects/acme/learnings.md")
	writeKBFile(t, root, "projects/acme/learnings.md", issue2RollingBody)
	if err := os.Chmod(path, 0o660); err != nil {
		t.Fatal(err)
	}
	out, exit := issue2RunCLIEnv(t, root, []string{"ISSUE2_CLI_PRIVATE_UMASK=1"}, "lint", "--fix", "--rolling-only")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if exit != 0 || info.Mode().Perm() != 0o660 {
		t.Fatalf("atomic replacement changed existing permissions under a private umask: exit=%d mode=%v\n%s", exit, info.Mode().Perm(), out)
	}
}

func TestIssue2RollingApplySelectors(t *testing.T) {
	for _, variant := range []string{"explicit", "changed", "changed-precedence", "root-alias", "ordinary-filter"} {
		t.Run(variant, func(t *testing.T) {
			root := testKB(t, "domains: [acme]\n")
			original := "---\nrolling: true\n---\n" + issue2RollingBody
			writeKBFile(t, root, "projects/acme/learnings.md", original)
			writeKBFile(t, root, "projects/acme/decisions-log.md", original)
			writeKBFile(t, root, "wiki/ordinary.md", "Ordinary article.\n")
			writeKBFile(t, root, "wiki/marked.txt", "---\nrolling: true\n---\n"+issue2RollingBody)
			mustGit(t, root, "init", "-q")
			mustGit(t, root, "add", ".")
			mustGit(t, root, "commit", "-qm", "seed")
			selectedRoot := root
			args := []string{"lint", "--fix", "--rolling-only"}
			count := 1
			switch variant {
			case "explicit":
				args = append(args, "projects/acme/learnings.md", "projects/acme/learnings.md")
			case "changed", "changed-precedence":
				writeKBFile(t, root, "projects/acme/learnings.md", original+"New entry bytes.\n")
				writeKBFile(t, root, "projects/acme/learnings-archive-2025.md", issue2RollingBody)
				args = append(args, "--changed")
				if variant == "changed-precedence" {
					args = append(args, "projects/acme/decisions-log.md")
				}
				count = 2
			case "root-alias":
				selectedRoot = filepath.Join(t.TempDir(), "selected-kb")
				if err := os.Symlink(root, selectedRoot); err != nil {
					t.Fatal(err)
				}
				args = append(args, "projects/acme/learnings.md")
			case "ordinary-filter":
				args = append(args, "wiki/ordinary.md", "wiki/marked.txt")
				count = 0
			}
			before := issue2RepairContent(t, root)
			out, exit := issue2RunCLI(t, selectedRoot, args...)
			if exit != 0 || !strings.Contains(out, fmt.Sprintf("scanned %d, repaired %d, unchanged 0, unresolved 0", count, count)) {
				t.Fatalf("apply selector scope incorrect: exit=%d\n%s", exit, out)
			}
			after := issue2RepairContent(t, root)
			if readKBFile(t, root, "projects/acme/decisions-log.md") != original || before["wiki/ordinary.md"] != after["wiki/ordinary.md"] || before["wiki/marked.txt"] != after["wiki/marked.txt"] {
				t.Fatal("apply selector touched an unselected or unrelated file")
			}
			if count == 0 && !reflect.DeepEqual(before, after) {
				t.Fatal("filtered apply changed content")
			}
		})
	}
}

func TestIssue2RollingApplyUnresolvedMetadata(t *testing.T) {
	for _, tc := range []struct{ name, fields, body, cause string }{
		{"confidence", "confidence: certain\n", issue2RollingBody, "invalid confidence"},
		{"malformed", "tags: [\n", issue2RollingBody, "invalid YAML"},
		{"contradictory", "created: 2026-11-02\n", issue2RollingBody, "before created"},
		{"undated", "", "Real entry without dates.\n", "cannot recover created"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := testKB(t, "domains: [acme]\n")
			original := "---\nrolling: true\n" + tc.fields + "---\n" + tc.body
			writeKBFile(t, root, "patterns/personal/unresolved.md", original)
			writeKBFile(t, root, "projects/acme/learnings.md", issue2RollingBody)
			out, exit := issue2RunCLI(t, root, "lint", "--fix", "--rolling-only")
			if exit == 0 || !strings.Contains(out, tc.cause) || !strings.Contains(out, "scanned 2, repaired 1, unchanged 0, unresolved 1") || readKBFile(t, root, "patterns/personal/unresolved.md") != original || len(validateFile(root, filepath.Join(root, "projects/acme/learnings.md"))) != 0 {
				t.Fatalf("unresolved file changed or blocked valid sibling: exit=%d\n%s", exit, out)
			}
		})
	}
}

func TestIssue2RollingApplyBusy(t *testing.T) {
	root := testKB(t, "domains: [acme]\n")
	writeKBFile(t, root, "projects/acme/learnings.md", issue2RollingBody)
	before := issue2KBBytes(t, root)
	path := lockPathFor(loadConfig(root).LockDir, "sync", root)
	lf, ok, err := acquireOperationLock(path, "sync")
	if err != nil || !ok {
		t.Fatalf("fixture lock failed: %v", err)
	}
	defer releaseLock(lf)
	ownerBefore, _ := os.ReadFile(path)
	for _, dryRun := range []bool{false, true} {
		args := []string{"lint", "--fix", "--rolling-only"}
		if dryRun {
			args = append(args, "--dry-run")
		}
		out, exit := issue2RunCLI(t, root, args...)
		ownerAfter, _ := os.ReadFile(path)
		if exit == 0 || !strings.Contains(out, "busy") || !strings.Contains(out, "sync (pid") || string(ownerBefore) != string(ownerAfter) || !reflect.DeepEqual(before, issue2KBBytes(t, root)) {
			t.Fatalf("busy apply/preview bypassed shared lock or changed content: dry=%v exit=%d\n%s", dryRun, exit, out)
		}
	}
}
