package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const issue2RollingBody = "\n## 2026-11-01 | New lesson\nPreserve trailing spaces.  \nSource: session:issue2-session\n\n## 2026-10-01 | Earlier lesson\nEarlier evidence.\n"

func issue2ToolsKB(t *testing.T, script string) string {
	t.Helper()
	root := testKB(t, "domains: [acme]\nccrider_db: "+filepath.Join(t.TempDir(), "none.db")+"\nsync:\n  checkpoint_interval: 100\n")
	t.Setenv("SCRIBE_KB", root)
	bin := t.TempDir()
	program := "#!/bin/sh\nset -eu\nprompt=$(cat)\n" + script + "\nprintf '%s\\n' '{\"type\":\"result\",\"is_error\":false,\"result\":\"done\"}'\n"
	if err := os.WriteFile(filepath.Join(bin, "claude"), []byte(program), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	prior := runClaude
	runClaude = realRunClaude
	t.Cleanup(func() { runClaude = prior })
	resetRunOutcome()
	t.Cleanup(resetRunOutcome)
	return root
}

func issue2CreateScript() string {
	return "mkdir -p projects/acme\ncat >projects/acme/learnings.md <<'ROLLING'\n---\nrolling: true\n---\n" + strings.TrimPrefix(issue2RollingBody, "\n") + "ROLLING\n"
}

func TestIssue2ToolsRollingSuccess(t *testing.T) {
	control := t.TempDir()
	t.Setenv("ISSUE2_CONTROL", control)
	root := issue2ToolsKB(t, issue2CreateScript()+issue2BarrierScript())
	untouched := "---\nrolling: true\n---\nUnrelated existing content.\n"
	writeKBFile(t, root, "projects/unrelated/learnings.md", untouched)
	cmd := &SyncCmd{Model: "sonnet"}
	type result struct {
		mined   int
		limited bool
	}
	done := make(chan result, 1)
	go func() {
		mined, limited := cmd.mineSessionBatches(root, []string{"issue2-session"}, 1, time.Second*5, "session-extract.md", "session")
		done <- result{mined, limited}
	}()
	issue2WaitForFile(t, filepath.Join(control, "ready"))
	issue2AssertPending(t, root, "issue2-session")
	if errs := validateFile(root, filepath.Join(root, "projects/acme/learnings.md")); len(errs) == 0 {
		t.Fatal("fixture must expose incomplete output before validation")
	}
	writeKBFile(t, control, "release", "continue")
	got := <-done
	mined, limited := got.mined, got.limited
	if mined != 1 || limited {
		t.Fatalf("mined=%d limited=%v", mined, limited)
	}
	path := filepath.Join(root, "projects/acme/learnings.md")
	if errs := validateFile(root, path); len(errs) != 0 {
		t.Fatalf("rolling output fails validation: %v", errs)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	span, err := frontmatterSpan(string(data))
	if err != nil || string(data[span.BodyStart:]) != strings.TrimPrefix(issue2RollingBody, "\n") {
		t.Fatalf("repair changed body: %q (%v)", data, err)
	}
	view, err := loadSessionsLogView(root)
	if err != nil || view.processed["issue2-session"] == nil {
		t.Fatalf("validated session not recorded: %v", err)
	}
	if readKBFile(t, root, "projects/unrelated/learnings.md") != untouched {
		t.Fatal("unrelated invalid regular file changed during mining")
	}
}

func issue2BarrierScript() string {
	return "touch \"$ISSUE2_CONTROL/ready\"\nwhile [ ! -f \"$ISSUE2_CONTROL/release\" ]; do sleep 0.01; done\n"
}

func issue2WaitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(4 * time.Second)
	for !fileExists(path) {
		if time.Now().After(deadline) {
			t.Fatalf("fixture did not reach barrier: %s", path)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func issue2AssertPending(t *testing.T, root, sid string) {
	t.Helper()
	view, err := loadSessionsLogView(root)
	if err != nil || view.processed[sid] != nil {
		t.Fatalf("session %s recorded before valid completion: %v", sid, err)
	}
}

func issue2ValidRolling() string {
	return "---\ntitle: Curated lessons\ntype: project\ncreated: 2026-09-01\nupdated: 2026-10-01\ndomain: acme\nconfidence: high\nauthority: canonical\nrolling: true\ntags: [curated]\nrelated: []\nsources: [original-source]\ncontributor: Original author\ncustom_field: preserved\n---\nOriginal body.  \n"
}

func TestIssue2ToolsRollingPreservesValues(t *testing.T) {
	addition := "\n## 2026-11-01 | New evidence\nA lesson.  \n"
	root := issue2ToolsKB(t, "cat >>projects/acme/learnings.md <<'ENTRY'\n"+addition+"ENTRY\n")
	original := issue2ValidRolling()
	writeKBFile(t, root, "projects/acme/learnings.md", original)
	cmd := &SyncCmd{Model: "sonnet"}
	mined, _ := cmd.mineSessionBatches(root, []string{"preserve"}, 1, 5*time.Second, "session-extract.md", "session")
	if mined != 1 || readKBFile(t, root, "projects/acme/learnings.md") != original+addition {
		t.Fatal("metadata repair changed valid fields or body bytes")
	}
}

func TestIssue2ToolsRollingRestoresAfterFailure(t *testing.T) {
	for _, tc := range []struct{ name, ending string }{
		{"child-error", "exit 7\n"},
		{"invalid-metadata", "cat >projects/acme/learnings.md <<'INVALID'\n---\nrolling: true\nconfidence: certain\n---\n## 2026-11-01 | Invalid claim\nINVALID\n"},
		{"rate-limit", "echo 'rate limit hit' >&2\nexit 7\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logs := captureSlog(t)
			control := t.TempDir()
			t.Setenv("ISSUE2_CONTROL", control)
			script := issue2CreateScript() + "cat >projects/acme/decisions-log.md <<'NEW'\n" + issue2ValidRolling() + "NEW\n" + issue2BarrierScript() + tc.ending
			root := issue2ToolsKB(t, script)
			original := issue2ValidRolling()
			writeKBFile(t, root, "projects/acme/learnings.md", original)
			cmd := &SyncCmd{Model: "sonnet"}
			done := make(chan struct{})
			var mined int
			go func() {
				mined, _ = cmd.mineSessionBatches(root, []string{"failed"}, 1, 5*time.Second, "session-extract.md", "session")
				close(done)
			}()
			issue2WaitForFile(t, filepath.Join(control, "ready"))
			if readKBFile(t, root, "projects/acme/learnings.md") == original || !fileExists(filepath.Join(root, "projects/acme/decisions-log.md")) {
				t.Fatal("fixture failed to mutate existing and new files before failure")
			}
			writeKBFile(t, control, "release", "fail now")
			<-done
			if mined != 0 || readKBFile(t, root, "projects/acme/learnings.md") != original || fileExists(filepath.Join(root, "projects/acme/decisions-log.md")) {
				t.Fatal("caught failure did not restore original rolling state")
			}
			issue2AssertPending(t, root, "failed")
			if tc.name == "invalid-metadata" && (!strings.Contains(logs.String(), "invalid confidence") || strings.Contains(logs.String(), "claude -p:")) {
				t.Fatalf("expected a successful child rejected for invalid metadata: %s", logs.String())
			}
		})
	}
}

func TestIssue2ToolsRollingSerializesWorkers(t *testing.T) {
	control := t.TempDir()
	t.Setenv("ISSUE2_CONTROL", control)
	script := "if ! mkdir \"$ISSUE2_CONTROL/active\"; then touch \"$ISSUE2_CONTROL/overlap\"; exit 9; fi\nsleep 0.05\nprintf '\\n## 2026-11-01 | Appended\\nentry\\n' >>projects/acme/learnings.md\nrmdir \"$ISSUE2_CONTROL/active\"\n"
	root := issue2ToolsKB(t, script)
	writeKBFile(t, root, "projects/acme/learnings.md", issue2ValidRolling())
	cmd := &SyncCmd{Model: "sonnet"}
	mined, _ := cmd.mineSessionBatches(root, []string{"first", "second"}, 3, 5*time.Second, "session-extract.md", "session")
	if mined != 2 || fileExists(filepath.Join(control, "overlap")) || strings.Count(readKBFile(t, root, "projects/acme/learnings.md"), "entry\n") != 2 {
		t.Fatal("tools subprocesses overlapped or lost an entry")
	}
}

func TestIssue2ToolsRollingProvenanceRejectsExamples(t *testing.T) {
	for _, tc := range []struct{ name, example string }{
		{"long-delimiter", "````markdown\n```\n## 2000-01-01 | Example\nsession:example\n````\n"},
		{"indented-closing", "```markdown\n    ```\n## 2000-01-01 | Example\nsession:example\n```\n"},
	} {
		t.Run(tc.name, func(t *testing.T) { issue2CheckProvenanceExample(t, tc.example) })
	}
}

func issue2CheckProvenanceExample(t *testing.T, example string) {
	t.Helper()
	root := testKB(t, "domains: [acme]\n")
	body := "\n" + example + issue2RollingBody
	data := []byte("---\nrolling: true\n---\n" + body)
	repaired, err := repairSessionRolling(root, "projects/acme/learnings.md", data)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := parseFrontmatterRaw(repaired)
	if stringFromAny(raw["created"]) != "2026-10-01" || stringFromAny(raw["updated"]) != "2026-11-01" || strings.Contains(stringFromAny(raw["sources"]), "example") {
		t.Fatalf("code example contaminated recovered provenance: %v", raw)
	}
	span, _ := frontmatterSpan(string(repaired))
	if !bytes.Equal(repaired[span.BodyStart:], []byte(body)) {
		t.Fatal("repair changed body")
	}
}

func TestIssue2ToolsRollingIncompleteInventoryStops(t *testing.T) {
	root := testKB(t, "domains: [acme]\n")
	writeKBFile(t, root, "projects/acme/learnings.md", issue2ValidRolling())
	blocked := filepath.Join(root, "solutions")
	if err := os.MkdirAll(blocked, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	err := runToolsRollingSession(root, func() error {
		writeKBFile(t, root, "projects/acme/learnings.md", "changed")
		if err := os.Remove(blocked); err != nil {
			return err
		}
		return os.Symlink(outside, blocked)
	})
	if !errors.Is(err, errRollingRestore) {
		t.Fatalf("incomplete inventory must stop commits, got %v", err)
	}
	if readKBFile(t, root, "projects/acme/learnings.md") != issue2ValidRolling() {
		t.Fatal("known rolling state was not restored")
	}
}

func TestIssue2ToolsRollingRestoresSymlinkOutput(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "new-file", true: "replaced-file"}[existing], func(t *testing.T) {
			outside := filepath.Join(t.TempDir(), "outside.md")
			if err := os.WriteFile(outside, []byte("outside stays unchanged"), 0o644); err != nil {
				t.Fatal(err)
			}
			t.Setenv("ISSUE2_OUTSIDE", outside)
			root := issue2ToolsKB(t, "mkdir -p projects/acme\nrm -f projects/acme/learnings.md\nln -s \"$ISSUE2_OUTSIDE\" projects/acme/learnings.md\n")
			if existing {
				writeKBFile(t, root, "projects/acme/learnings.md", issue2ValidRolling())
			}
			cmd := &SyncCmd{Model: "sonnet"}
			mined, _ := cmd.mineSessionBatches(root, []string{"symlink"}, 1, 5*time.Second, "session-extract.md", "session")
			if mined != 0 || cmd.fatalSessionErr != nil {
				t.Fatalf("unexpected symlink outcome: mined=%d fatal=%v", mined, cmd.fatalSessionErr)
			}
			if existing {
				if readKBFile(t, root, "projects/acme/learnings.md") != issue2ValidRolling() {
					t.Fatal("original not restored")
				}
			} else if _, err := os.Lstat(filepath.Join(root, "projects/acme/learnings.md")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("new symlink remains: %v", err)
			}
			outsideBytes, _ := os.ReadFile(outside)
			if string(outsideBytes) != "outside stays unchanged" {
				t.Fatal("restoration followed symlink")
			}
			issue2AssertPending(t, root, "symlink")
		})
	}
}

func TestIssue2ToolsRollingRejectsExistingSymlink(t *testing.T) {
	for _, tc := range []struct{ name, ending string }{
		{"child-success", ""},
		{"child-error", "exit 7\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			control := t.TempDir()
			t.Setenv("ISSUE2_CONTROL", control)
			root := issue2ToolsKB(t, "touch \"$ISSUE2_CONTROL/child-ran\"\nprintf 'invalid changed bytes' >projects/acme/learnings.md\n"+tc.ending)
			original := issue2ValidRolling()
			writeKBFile(t, root, "projects/acme/backing.txt", original)
			link := filepath.Join(root, "projects/acme/learnings.md")
			if err := os.Symlink("backing.txt", link); err != nil {
				t.Fatal(err)
			}
			cmd := &SyncCmd{Model: "sonnet"}
			mined, _ := cmd.mineSessionBatches(root, []string{"existing-symlink"}, 1, 5*time.Second, "session-extract.md", "session")
			if mined != 0 || fileExists(filepath.Join(control, "child-ran")) {
				t.Fatal("unsafe child ran despite an existing Markdown symlink")
			}
			if readKBFile(t, root, "projects/acme/backing.txt") != original {
				t.Fatal("preflight rejection changed rolling backing bytes")
			}
			if target, err := os.Readlink(link); err != nil || target != "backing.txt" {
				t.Fatalf("preflight rejection changed original link: %q %v", target, err)
			}
			issue2AssertPending(t, root, "existing-symlink")
		})
	}
}

func TestIssue2ToolsRollingRejectsExistingDirectoryAlias(t *testing.T) {
	for _, tc := range []struct{ name, ending string }{
		{"child-success", ""},
		{"child-error", "exit 7\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			control := t.TempDir()
			t.Setenv("ISSUE2_CONTROL", control)
			root := issue2ToolsKB(t, "touch \"$ISSUE2_CONTROL/child-ran\"\nprintf 'invalid changed bytes' >projects/acme/learnings.md\n"+tc.ending)
			original := issue2ValidRolling()
			writeKBFile(t, root, "raw/acme/learnings.md", original)
			if err := os.MkdirAll(filepath.Join(root, "projects"), 0o755); err != nil {
				t.Fatal(err)
			}
			link := filepath.Join(root, "projects/acme")
			if err := os.Symlink("../raw/acme", link); err != nil {
				t.Fatal(err)
			}
			cmd := &SyncCmd{Model: "sonnet"}
			mined, _ := cmd.mineSessionBatches(root, []string{"directory-alias"}, 1, 5*time.Second, "session-extract.md", "session")
			if mined != 0 || cmd.fatalSessionErr != nil || fileExists(filepath.Join(control, "child-ran")) {
				t.Errorf("unsafe child ran despite a pre-existing directory alias: mined=%d fatal=%v", mined, cmd.fatalSessionErr)
			}
			if readKBFile(t, root, "raw/acme/learnings.md") != original {
				t.Error("preflight rejection did not preserve backing bytes")
			}
			if target, err := os.Readlink(link); err != nil || target != "../raw/acme" {
				t.Fatalf("preflight rejection changed the directory link: %q %v", target, err)
			}
			issue2AssertPending(t, root, "directory-alias")
		})
	}
}

func TestIssue2ToolsRollingCheckpoint(t *testing.T) {
	script := "case \"$prompt\" in\n*'Session IDs: bad'*) printf 'failed-only' >projects/acme/learnings.md; exit 7;;\n*'Session IDs: good'*)\n" + issue2CreateScript() + ";;\nesac\n"
	root := issue2ToolsKB(t, script)
	writeKBFile(t, root, "projects/acme/learnings.md", issue2ValidRolling())
	config := readKBFile(t, root, "scribe.yaml")
	writeKBFile(t, root, "scribe.yaml", strings.Replace(config, "checkpoint_interval: 100", "checkpoint_interval: 1", 1))
	t.Setenv("SCRIBE_SKIP_REINDEX", "1")
	mustGit(t, root, "init", "-q")
	mustGit(t, root, "add", ".")
	mustGit(t, root, "commit", "-qm", "seed")
	cmd := &SyncCmd{Model: "sonnet"}
	mined, _ := cmd.mineSessionBatches(root, []string{"bad", "good", "noop"}, 3, 5*time.Second, "session-extract.md", "session")
	if mined != 2 || strings.TrimSpace(mustGit(t, root, "rev-list", "--count", "HEAD")) != "2" {
		t.Fatal("expected one real checkpoint after the successful session")
	}
	committed := mustGit(t, root, "show", "HEAD:projects/acme/learnings.md")
	if failures := validateArticleContent(root, []byte(committed)); len(failures) != 0 || strings.Contains(committed, "failed-only") {
		t.Fatalf("checkpoint contains invalid or failed output: %v", failures)
	}
	log := mustGit(t, root, "show", "HEAD:wiki/_sessions_log.json")
	if strings.Contains(log, `"bad"`) || !strings.Contains(log, `"good"`) || strings.Contains(log, `"noop"`) {
		t.Fatalf("checkpoint processed markers disagree with validated output: %s", log)
	}
}

func TestIssue2ToolsRollingRestoreFailureStopsSync(t *testing.T) {
	control := t.TempDir()
	t.Setenv("ISSUE2_CONTROL", control)
	script := "echo run >>\"$ISSUE2_CONTROL/runs\"\nprintf 'mutated' >projects/acme/learnings.md\nmv projects/acme projects/saved\nprintf 'blocked' >projects/acme\n" + issue2BarrierScript()
	root := issue2ToolsKB(t, script)
	writeKBFile(t, root, "projects/acme/learnings.md", issue2ValidRolling())
	head := issue2PrepareSync(t, root)
	cmd := &SyncCmd{Sessions: true, SessionsMax: 2, SessionSort: "score", SkipLarge: true, Model: "sonnet"}
	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()
	issue2WaitForFile(t, filepath.Join(control, "ready"))
	if readKBFile(t, root, "projects/acme") != "blocked" || readKBFile(t, root, "projects/saved/learnings.md") != "mutated" {
		t.Fatal("fixture did not establish a real mutation and restoration failure")
	}
	writeKBFile(t, control, "release", "continue")
	err := <-done
	if !errors.Is(err, errRollingRestore) || mustGit(t, root, "rev-parse", "HEAD") != head {
		t.Fatalf("sync must stop before final commit: %v", err)
	}
	if readKBFile(t, control, "runs") != "run\n" {
		t.Fatal("sync launched a later session after restoration failed")
	}
	issue2AssertPending(t, root, "fatal")
	issue2AssertPending(t, root, "later")
	if staged := mustGit(t, root, "diff", "--cached", "--name-only"); strings.TrimSpace(staged) != "" {
		t.Fatalf("sync staged partial output after restoration failed: %s", staged)
	}
}

func issue2PrepareSync(t *testing.T, root string) string {
	t.Helper()
	writeKBFile(t, root, "wiki/.keep", "")
	bin := t.TempDir()
	writeKBFile(t, bin, "ccrider", "#!/bin/sh\nexit 0\n")
	triage := "#!/bin/sh\nprintf '%s\\n' '[{\"session_id\":\"fatal\",\"total_score\":100,\"msgs\":10},{\"session_id\":\"later\",\"total_score\":90,\"msgs\":10}]'\n"
	writeKBFile(t, bin, "scribe", triage)
	for _, name := range []string{"ccrider", "scribe"} {
		if err := os.Chmod(filepath.Join(bin, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	prior := sessionExecutable
	sessionExecutable = func() (string, error) { return filepath.Join(bin, "scribe"), nil }
	t.Cleanup(func() { sessionExecutable = prior })
	t.Setenv("SCRIBE_SKIP_REINDEX", "1")
	t.Setenv("SCRIBE_NO_CONFIG_BACKFILL", "1")
	mustGit(t, root, "init", "-q")
	mustGit(t, root, "add", ".")
	mustGit(t, root, "commit", "-qm", "seed")
	return mustGit(t, root, "rev-parse", "HEAD")
}

func TestIssue2ToolsRollingRestoreRejectsAliasedParent(t *testing.T) {
	control := t.TempDir()
	t.Setenv("ISSUE2_CONTROL", control)
	script := "echo run >>\"$ISSUE2_CONTROL/runs\"\nprintf 'mutated' >projects/acme/learnings.md\n" + issue2BarrierScript() + "exit 7\n"
	root := issue2ToolsKB(t, script)
	writeKBFile(t, root, "projects/acme/learnings.md", issue2ValidRolling())
	other := strings.ReplaceAll(issue2ValidRolling(), "Original body.", "Other project's original body.")
	writeKBFile(t, root, "projects/other/learnings.md", other)
	head := issue2PrepareSync(t, root)
	cmd := &SyncCmd{Sessions: true, SessionsMax: 2, SessionSort: "score", SkipLarge: true, Model: "sonnet"}
	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()
	issue2WaitForFile(t, filepath.Join(control, "ready"))
	if readKBFile(t, root, "projects/acme/learnings.md") != "mutated" {
		t.Fatal("child did not mutate the rolling file before fault injection")
	}
	if err := os.Rename(filepath.Join(root, "projects/acme"), filepath.Join(root, "projects/saved")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("other", filepath.Join(root, "projects/acme")); err != nil {
		t.Fatal(err)
	}
	writeKBFile(t, control, "release", "continue")
	err := <-done
	if readKBFile(t, root, "projects/other/learnings.md") != other {
		t.Fatal("restoration followed a symlink ancestor into an untouched project")
	}
	if !errors.Is(err, errRollingRestore) || mustGit(t, root, "rev-parse", "HEAD") != head {
		t.Fatalf("sync must stop before final commit: %v", err)
	}
	if readKBFile(t, control, "runs") != "run\n" {
		t.Fatal("sync launched a later session after restoration failed")
	}
	issue2AssertPending(t, root, "fatal")
	issue2AssertPending(t, root, "later")
	if staged := mustGit(t, root, "diff", "--cached", "--name-only"); strings.TrimSpace(staged) != "" {
		t.Fatalf("sync staged partial output after restoration failed: %s", staged)
	}
}

func TestIssue2ToolsRollingRestoreRejectsAliasedParentRemoval(t *testing.T) {
	root := testKB(t, "domains: [acme]\n")
	other := issue2ValidRolling()
	writeKBFile(t, root, "projects/other/learnings.md", other)
	before, err := captureRollingState(root)
	if err != nil {
		t.Fatal(err)
	}
	writeKBFile(t, root, "projects/acme/learnings.md", "new session output")
	after, err := captureRollingState(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(root, "projects/acme"), filepath.Join(root, "projects/saved")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("other", filepath.Join(root, "projects/acme")); err != nil {
		t.Fatal(err)
	}
	err = restoreRollingState(root, before, after)
	data, readErr := os.ReadFile(filepath.Join(root, "projects/other/learnings.md"))
	if err == nil || readErr != nil || string(data) != other {
		t.Fatalf("restoration removal touched an aliased project: restore=%v read=%v data=%q", err, readErr, data)
	}
}

func TestIssue2ToolsRollingRestoresThroughRootAlias(t *testing.T) {
	control := t.TempDir()
	t.Setenv("ISSUE2_CONTROL", control)
	root := issue2ToolsKB(t, "printf 'mutated' >projects/acme/learnings.md\n"+issue2BarrierScript()+"exit 7\n")
	original := issue2ValidRolling()
	writeKBFile(t, root, "projects/acme/learnings.md", original)
	alias := filepath.Join(t.TempDir(), "selected-kb")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SCRIBE_KB", alias)
	cmd := &SyncCmd{Model: "sonnet"}
	done := make(chan int, 1)
	go func() {
		mined, _ := cmd.mineSessionBatches(alias, []string{"root-alias"}, 1, 5*time.Second, "session-extract.md", "session")
		done <- mined
	}()
	issue2WaitForFile(t, filepath.Join(control, "ready"))
	if readKBFile(t, root, "projects/acme/learnings.md") != "mutated" {
		t.Fatal("fixture did not mutate through the selected KB-root alias")
	}
	writeKBFile(t, control, "release", "fail now")
	mined := <-done
	if mined != 0 || cmd.fatalSessionErr != nil || readKBFile(t, root, "projects/acme/learnings.md") != original {
		t.Fatalf("stable configured KB-root alias prevented restoration: mined=%d fatal=%v", mined, cmd.fatalSessionErr)
	}
	issue2AssertPending(t, alias, "root-alias")
}
