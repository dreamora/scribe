package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func issue2WriteScript(path, content string) string {
	return "mkdir -p '" + filepath.ToSlash(filepath.Dir(path)) + "'\ncat >'" + path + "' <<'ROLLING_VARIANT'\n" + content + "ROLLING_VARIANT\n"
}

func TestIssue2ToolsVariantsNormal(t *testing.T) {
	issue2CheckToolsVariants(t, "session-extract.md", "session")
}

func TestIssue2ToolsVariantsLarge(t *testing.T) {
	issue2CheckToolsVariants(t, "session-extract-large.md", "large-session")
}

func issue2CheckToolsVariants(t *testing.T, promptName, label string) {
	t.Helper()
	cases := []struct {
		path, header, title, domain string
	}{
		{"projects/acme/learnings.md", "---\n---\n", "Acme Learnings", "acme"},
		{"projects/acme/decisions-log.md", "---\n---\n", "Acme Decisions Log", "acme"},
		{"projects/acme/risk-notes.md", "---\n---\n", "Acme Risk Notes", "acme"},
		{"projects/acme/learnings-archive-2025.md", "---\n---\n", "Acme Learnings Archive 2025", "acme"},
		{"projects/acme/decisions-log-archive-2025.md", "---\n---\n", "Acme Decisions Log Archive 2025", "acme"},
		{"projects/acme/risk-notes-archive-2025.md", "---\n---\n", "Acme Risk Notes Archive 2025", "acme"},
		{"patterns/personal/field-notes.md", "---\nrolling: true\n---\n", "Personal Field Notes", "personal"},
		{"projects/équipe/retour-expérience.md", "---\n---\n", "Équipe Retour Expérience", "general"},
		{"projects/acme/learnings-archive-2024.md", "", "Acme Learnings Archive 2024", "acme"},
	}
	body := "\n```markdown\n## 2000-01-01 | Example\nsession:example\n```\n" + issue2RollingBody
	var script strings.Builder
	for _, tc := range cases {
		script.WriteString(issue2WriteScript(tc.path, tc.header+body))
	}
	article := "Ordinary article with no rolling marker.\n"
	script.WriteString(issue2WriteScript("projects/acme/ordinary.md", article))
	control := t.TempDir()
	t.Setenv("ISSUE2_CONTROL", control)
	root := issue2ToolsKB(t, script.String()+issue2BarrierScript())
	writeKBFile(t, root, "scribe.yaml", readKBFile(t, root, "scribe.yaml")+"meta:\n  rolling_targets: [risk-notes, retour-expérience]\n")
	untouched := "---\nrolling: true\nconfidence: certain\n---\nUntouched invalid regular file.\n"
	writeKBFile(t, root, "projects/other/decisions-log.md", untouched)
	cmd := &SyncCmd{Model: "sonnet"}
	done := make(chan int, 1)
	go func() {
		mined, _ := cmd.mineSessionBatches(root, []string{"normal-variants"}, 3, 5*time.Second, promptName, label)
		done <- mined
	}()
	issue2WaitForFile(t, filepath.Join(control, "ready"))
	issue2AssertPending(t, root, "normal-variants")
	if readKBFile(t, root, cases[1].path) != cases[1].header+body {
		t.Fatal("child did not write incomplete decisions output before validation")
	}
	writeKBFile(t, control, "release", "continue")
	if mined := <-done; mined != 1 || cmd.fatalSessionErr != nil {
		t.Fatalf("normal variants not accepted: mined=%d fatal=%v", mined, cmd.fatalSessionErr)
	}
	for _, tc := range cases {
		data := []byte(readKBFile(t, root, tc.path))
		if errs := validateArticleContent(root, data); len(errs) != 0 {
			t.Fatalf("%s: invalid output: %v", tc.path, errs)
		}
		raw, err := parseFrontmatterRaw(data)
		if err != nil {
			t.Fatal(err)
		}
		for key, want := range map[string]string{"title": tc.title, "domain": tc.domain, "type": "project", "confidence": "medium", "authority": "contextual", "created": "2026-10-01", "updated": "2026-11-01"} {
			if got := stringFromAny(raw[key]); got != want {
				t.Fatalf("%s: %s=%q, want %q", tc.path, key, got, want)
			}
		}
		sources, sourcesOK := raw["sources"].([]any)
		tags, tagsOK := raw["tags"].([]any)
		related, relatedOK := raw["related"].([]any)
		if raw["rolling"] != true || !sourcesOK || len(sources) != 1 || sources[0] != "session:issue2-session" || !tagsOK || len(tags) != 0 || !relatedOK || len(related) != 0 {
			t.Fatalf("%s: wrong marker, lists, or recovered sources: %v", tc.path, raw)
		}
		span, _ := frontmatterSpan(string(data))
		if !bytes.Equal(data[span.BodyStart:], []byte(body)) {
			t.Fatalf("%s: body bytes changed", tc.path)
		}
		t.Logf("%s: title=%s domain=%s created=%s updated=%s; body preserved", tc.path, raw["title"], raw["domain"], stringFromAny(raw["created"]), stringFromAny(raw["updated"]))
	}
	view, err := loadSessionsLogView(root)
	if err != nil || view.processed["normal-variants"] == nil {
		t.Fatalf("validated normal session not marked processed: %v", err)
	}
	if readKBFile(t, root, "projects/acme/ordinary.md") != article || readKBFile(t, root, "projects/other/decisions-log.md") != untouched {
		t.Fatal("normal repair changed unrelated content")
	}
	t.Logf("complete normal-session result:\n%s", readKBFile(t, root, cases[2].path))
}

func TestIssue2ToolsVariantsRejectAndRestore(t *testing.T) {
	for _, prompt := range []string{"session-extract.md", "session-extract-large.md"} {
		for _, tc := range []struct{ name, content, ending string }{
			{"child-error", "---\nrolling: true\n---\n" + issue2RollingBody, "exit 7\n"},
			{"invalid-confidence", "---\nrolling: true\nconfidence: certain\n---\n" + issue2RollingBody, ""},
			{"false-marker", "---\nrolling: false\n---\n" + issue2RollingBody, ""},
			{"malformed-yaml", "---\nrolling: true\ntags: [\n---\n" + issue2RollingBody, ""},
			{"malformed-marked-nonstandard", "---\nrolling: true\ntags: [\n---\n" + issue2RollingBody, ""},
			{"missing-dates", "---\nrolling: true\n---\nUndated entry.\n", ""},
			{"contradictory-dates", "---\nrolling: true\ncreated: 2026-11-02\n---\n" + issue2RollingBody, ""},
			{"invalid-calendar-date", "---\nrolling: true\ncreated: 2026-02-30\n---\n" + issue2RollingBody, ""},
			{"empty-confidence", "---\nrolling: true\nconfidence: null\n---\n" + issue2RollingBody, ""},
		} {
			t.Run(prompt+"/"+tc.name, func(t *testing.T) {
				logs := captureSlog(t)
				control := t.TempDir()
				t.Setenv("ISSUE2_CONTROL", control)
				changed := "projects/acme/decisions-log.md"
				created := "projects/acme/risk-notes-archive-2025.md"
				invalid := "projects/acme/z-invalid.md"
				if tc.name == "false-marker" || tc.name == "malformed-yaml" {
					invalid = "projects/acme/learnings-archive-2025.md"
				}
				if tc.name == "malformed-marked-nonstandard" {
					invalid = "patterns/personal/field-notes.md"
				}
				script := issue2WriteScript(changed, "---\nrolling: true\n---\n"+issue2RollingBody) + issue2WriteScript(created, "---\n---\n"+issue2RollingBody) + issue2WriteScript(invalid, tc.content) + issue2BarrierScript() + tc.ending
				root := issue2ToolsKB(t, script)
				writeKBFile(t, root, "scribe.yaml", readKBFile(t, root, "scribe.yaml")+"meta:\n  rolling_targets: [risk-notes]\n")
				original := issue2ValidRolling()
				writeKBFile(t, root, changed, original)
				untouched := "---\nrolling: true\n---\nUnrelated invalid content.\n"
				writeKBFile(t, root, "projects/other/learnings.md", untouched)
				cmd := &SyncCmd{Model: "sonnet"}
				done := make(chan int, 1)
				go func() {
					mined, _ := cmd.mineSessionBatches(root, []string{"reject-variant"}, 3, 5*time.Second, prompt, "session")
					done <- mined
				}()
				issue2WaitForFile(t, filepath.Join(control, "ready"))
				if readKBFile(t, root, changed) == original || !fileExists(filepath.Join(root, created)) || readKBFile(t, root, invalid) != tc.content {
					t.Fatal("child did not mutate existing and new rolling variants before failure")
				}
				issue2AssertPending(t, root, "reject-variant")
				writeKBFile(t, control, "release", "finish")
				if mined := <-done; mined != 0 || cmd.fatalSessionErr != nil {
					t.Fatalf("invalid output accepted or failed restoration: mined=%d fatal=%v", mined, cmd.fatalSessionErr)
				}
				if readKBFile(t, root, changed) != original || fileExists(filepath.Join(root, created)) || fileExists(filepath.Join(root, invalid)) || readKBFile(t, root, "projects/other/learnings.md") != untouched {
					t.Fatal("failed session did not restore selected variants while preserving unrelated bytes")
				}
				issue2AssertPending(t, root, "reject-variant")
				if tc.name != "child-error" && strings.Contains(logs.String(), "claude -p:") {
					t.Fatalf("metadata rejection was caused by a child error: %s", logs.String())
				}
			})
		}
	}
}

func TestIssue2ToolsVariantsStopsAtLimits(t *testing.T) {
	for _, prompt := range []string{"session-extract.md", "session-extract-large.md"} {
		for _, limit := range []string{"rate", "daily-budget"} {
			t.Run(prompt+"/"+limit, func(t *testing.T) {
				control := t.TempDir()
				t.Setenv("ISSUE2_CONTROL", control)
				script := "echo run >>\"$ISSUE2_CONTROL/runs\"\nprintf 'mutated' >projects/acme/decisions-log.md\necho 'rate limit hit' >&2\nexit 7\n"
				root := issue2ToolsKB(t, script)
				original := issue2ValidRolling()
				writeKBFile(t, root, "projects/acme/decisions-log.md", original)
				if limit == "daily-budget" {
					t.Setenv("SCRIBE_BYPASS_BUDGET", "")
					resetBudgetCacheForTest()
					t.Cleanup(resetBudgetCacheForTest)
					writeKBFile(t, root, "scribe.yaml", strings.Replace(readKBFile(t, root, "scribe.yaml"), "checkpoint_interval: 100", "checkpoint_interval: 100\n  daily_output_token_ceiling: 1", 1))
					writeBudgetFixture(t, root, time.Now().UTC().Format(time.DateOnly), []CostEntry{{Provider: "anthropic", OutputTokens: ptrI64(1)}})
				}
				cmd := &SyncCmd{Model: "sonnet"}
				mined, limited := cmd.mineSessionBatches(root, []string{"limited", "later"}, 3, 5*time.Second, prompt, "session")
				if mined != 0 || !limited || cmd.fatalSessionErr != nil || readKBFile(t, root, "projects/acme/decisions-log.md") != original {
					t.Fatalf("limit did not stop pending sessions safely: mined=%d limited=%v fatal=%v", mined, limited, cmd.fatalSessionErr)
				}
				if limit == "rate" {
					if readKBFile(t, control, "runs") != "run\n" {
						t.Fatal("a later child ran after rate limiting")
					}
				} else if fileExists(filepath.Join(control, "runs")) {
					t.Fatal("daily-budget exhaustion still launched a child")
				}
				issue2AssertPending(t, root, "limited")
				issue2AssertPending(t, root, "later")
			})
		}
	}
}

func TestIssue2ToolsVariantsPreservesMetadata(t *testing.T) {
	for _, prompt := range []string{"session-extract.md", "session-extract-large.md"} {
		t.Run(prompt, func(t *testing.T) {
			path := "projects/acme/decisions-log-archive-2025.md"
			addition := "\n## 2026-11-01 | New evidence\nA new lesson.  \n"
			root := issue2ToolsKB(t, "cat >>"+path+" <<'APPEND'\n"+addition+"APPEND\n")
			original := strings.Replace(issue2ValidRolling(), "title: Curated lessons\n", "", 1)
			writeKBFile(t, root, path, original)
			cmd := &SyncCmd{Model: "sonnet"}
			mined, _ := cmd.mineSessionBatches(root, []string{"preserve-variants"}, 3, 5*time.Second, prompt, "session")
			data := readKBFile(t, root, path)
			if mined != 1 || strings.Replace(data, "title: Acme Decisions Log Archive 2025\n", "", 1) != original+addition {
				t.Fatalf("repair changed valid existing fields or body bytes: mined=%d data=%s", mined, data)
			}
		})
	}
}

func TestIssue2ToolsVariantsNoReferences(t *testing.T) {
	body := "\n~~~markdown\n## 2000-01-01 | Example\nsession:example\n~~~\n\n## 2026-11-01 | Real entry\nNo explicit reference.  \n"
	path := "decisions/personal/observations.md"
	root := issue2ToolsKB(t, issue2WriteScript(path, "---\nrolling: true\n---\n"+body))
	cmd := &SyncCmd{Model: "sonnet"}
	mined, _ := cmd.mineSessionBatches(root, []string{"no-references"}, 1, 5*time.Second, "session-extract.md", "session")
	data := []byte(readKBFile(t, root, path))
	raw, err := parseFrontmatterRaw(data)
	sources, ok := raw["sources"].([]any)
	span, _ := frontmatterSpan(string(data))
	if mined != 1 || err != nil || !ok || len(sources) != 0 || stringFromAny(raw["created"]) != "2026-11-01" || stringFromAny(raw["updated"]) != "2026-11-01" || !bytes.Equal(data[span.BodyStart:], []byte(body)) {
		t.Fatalf("unsupported provenance invented: mined=%d raw=%v error=%v", mined, raw, err)
	}
}

func TestIssue2ToolsVariantsLargeCheckpoint(t *testing.T) {
	path := "projects/acme/decisions-log-archive-2025.md"
	script := "case \"$prompt\" in\n*'Session ID: bad'*) printf 'failed-only' >" + path + "; exit 7;;\n*'Session ID: good'*)\n" + issue2WriteScript(path, "---\n---\n"+issue2RollingBody) + ";;\nesac\n"
	root := issue2ToolsKB(t, script)
	writeKBFile(t, root, path, issue2ValidRolling())
	writeKBFile(t, root, "scribe.yaml", strings.Replace(readKBFile(t, root, "scribe.yaml"), "checkpoint_interval: 100", "checkpoint_interval: 1", 1))
	t.Setenv("SCRIBE_SKIP_REINDEX", "1")
	mustGit(t, root, "init", "-q")
	mustGit(t, root, "add", ".")
	mustGit(t, root, "commit", "-qm", "seed")
	cmd := &SyncCmd{Model: "sonnet"}
	mined, _ := cmd.mineSessionBatches(root, []string{"bad", "good", "noop"}, 3, 5*time.Second, "session-extract-large.md", "large-session")
	if mined != 2 || strings.TrimSpace(mustGit(t, root, "rev-list", "--count", "HEAD")) != "2" {
		t.Fatal("expected one real checkpoint after the successful large session")
	}
	committed := mustGit(t, root, "show", "HEAD:"+path)
	if failures := validateArticleContent(root, []byte(committed)); len(failures) != 0 || strings.Contains(committed, "failed-only") {
		t.Fatalf("checkpoint contains invalid or failed large output: %v", failures)
	}
	log := mustGit(t, root, "show", "HEAD:wiki/_sessions_log.json")
	if strings.Contains(log, `"bad"`) || !strings.Contains(log, `"good"`) || strings.Contains(log, `"noop"`) {
		t.Fatalf("large checkpoint markers disagree with validated output: %s", log)
	}
}

func TestIssue2ToolsVariantsLargeRestoreFailureStopsSync(t *testing.T) {
	control := t.TempDir()
	t.Setenv("ISSUE2_CONTROL", control)
	script := "echo run >>\"$ISSUE2_CONTROL/runs\"\nprintf 'mutated' >projects/acme/decisions-log.md\nmv projects/acme projects/saved\nprintf 'blocked' >projects/acme\n" + issue2BarrierScript()
	root := issue2ToolsKB(t, script)
	writeKBFile(t, root, "projects/acme/decisions-log.md", issue2ValidRolling())
	head := issue2PrepareSync(t, root)
	executable, err := sessionExecutable()
	if err != nil {
		t.Fatal(err)
	}
	triage := "#!/bin/sh\ncase \" $* \" in\n*' --min-messages 301 '*) printf '%s\\n' '[{\"session_id\":\"fatal\",\"total_score\":100,\"msgs\":600},{\"session_id\":\"later\",\"total_score\":90,\"msgs\":500}]';;\n*) printf '%s\\n' '[]';;\nesac\n"
	if err := os.WriteFile(executable, []byte(triage), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := &SyncCmd{Sessions: true, SessionsMax: 6, SessionSort: "score", Model: "sonnet"}
	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()
	issue2WaitForFile(t, filepath.Join(control, "ready"))
	if readKBFile(t, root, "projects/acme") != "blocked" || readKBFile(t, root, "projects/saved/decisions-log.md") != "mutated" {
		t.Fatal("large fixture did not establish mutation and failed restoration")
	}
	writeKBFile(t, control, "release", "continue")
	err = <-done
	if !errors.Is(err, errRollingRestore) || mustGit(t, root, "rev-parse", "HEAD") != head {
		t.Fatalf("large restoration failure did not stop final commit: %v", err)
	}
	if readKBFile(t, control, "runs") != "run\n" {
		t.Fatal("sync launched a later large session after failed restoration")
	}
	issue2AssertPending(t, root, "fatal")
	issue2AssertPending(t, root, "later")
	if data, err := os.ReadFile(filepath.Join(root, "wiki/_sessions_log.json")); err == nil {
		var log map[string]any
		if err := json.Unmarshal(data, &log); err != nil || log["last_scan"] != nil {
			t.Fatalf("fatal large pass still advanced scan timestamp: %s (%v)", data, err)
		}
	}
	if staged := mustGit(t, root, "diff", "--cached", "--name-only"); strings.TrimSpace(staged) != "" {
		t.Fatalf("sync staged failed large output: %s", staged)
	}
}

func TestIssue2ToolsVariantsUnclassifiedAliasStops(t *testing.T) {
	control := t.TempDir()
	t.Setenv("ISSUE2_CONTROL", control)
	script := "echo run >>\"$ISSUE2_CONTROL/runs\"\nmkdir -p patterns/personal\nln -s ../../raw/backing.txt patterns/personal/field-notes.md\n" + issue2BarrierScript()
	root := issue2ToolsKB(t, script)
	original := "Backing bytes stay unchanged.\n"
	writeKBFile(t, root, "raw/backing.txt", original)
	cmd := &SyncCmd{Model: "sonnet"}
	done := make(chan int, 1)
	go func() {
		mined, _ := cmd.mineSessionBatches(root, []string{"alias", "later"}, 3, 5*time.Second, "session-extract-large.md", "large-session")
		done <- mined
	}()
	issue2WaitForFile(t, filepath.Join(control, "ready"))
	if _, err := os.Readlink(filepath.Join(root, "patterns/personal/field-notes.md")); err != nil {
		t.Fatalf("child did not create the unsupported alias: %v", err)
	}
	writeKBFile(t, control, "release", "finish")
	if mined := <-done; mined != 0 || !errors.Is(cmd.fatalSessionErr, errRollingRestore) {
		t.Fatalf("unclassified alias must stop the run: mined=%d fatal=%v", mined, cmd.fatalSessionErr)
	}
	if readKBFile(t, control, "runs") != "run\n" || readKBFile(t, root, "raw/backing.txt") != original {
		t.Fatal("unsafe inventory launched another child or touched alias backing bytes")
	}
	issue2AssertPending(t, root, "alias")
	issue2AssertPending(t, root, "later")
}
