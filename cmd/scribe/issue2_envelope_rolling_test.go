package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type issue2EnvelopeProvider struct {
	generate func(context.Context, string) (string, error)
}

func (p *issue2EnvelopeProvider) Name() string { return "issue2-fixture" }
func (p *issue2EnvelopeProvider) Generate(ctx context.Context, prompt string) (string, error) {
	return p.generate(ctx, prompt)
}

func issue2EnvelopeKB(t *testing.T, ids ...string) (string, string) {
	t.Helper()
	db, dbPath := newCcriderDB(t)
	for _, sid := range ids {
		row := insertFixtureSession(t, db, sid, "/work/fixture/projects/acme", 6, "", "", "durable decisions")
		for range 3 {
			insertFixtureMessage(t, db, row, "user", strings.Repeat("Explain durable receipts and idempotence before acknowledging a task. ", 4), false)
			insertFixtureMessage(t, db, row, "assistant", "Persist a receipt so retries cannot duplicate effects.", false)
		}
	}
	root := testKB(t, "domains: [acme]\nccrider_db: "+dbPath+"\nsession_mine:\n  mode: envelope\n  provider: anthropic\nsync:\n  checkpoint_interval: 100\n")
	writeKBFile(t, root, "wiki/.keep", "")
	t.Setenv("SCRIBE_KB", root)
	t.Setenv("SCRIBE_SKIP_REINDEX", "1")
	resetRunOutcome()
	t.Cleanup(resetRunOutcome)
	return root, dbPath
}

func issue2WaitSignal(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("fixture did not reach expected barrier")
	}
}

func issue2Release(ch chan struct{}) {
	select {
	case <-ch:
	default:
		close(ch)
	}
}

func issue2ProviderWait(ctx context.Context, ch <-chan struct{}) error {
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return fmt.Errorf("fixture barrier timed out")
	}
}

func issue2EnvelopeJSON(t *testing.T, env WikiActionEnvelope) string {
	t.Helper()
	env.Version = 2
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestIssue2EnvelopeRepresentative(t *testing.T) {
	db, dbPath := newCcriderDB(t)
	row := insertFixtureSession(t, db, "representative", "/work/acme", 2, "", "", "a decision")
	insertFixtureMessage(t, db, row, "user", "Explain the durable receipt before acknowledging a task.", false)
	insertFixtureMessage(t, db, row, "assistant", "Persist a receipt before acknowledging so retries cannot duplicate effects.", false)
	root := sessionsTestKB(t, dbPath)
	t.Setenv("SCRIBE_KB", root)
	body := strings.TrimPrefix(issue2RollingBody, "\n")
	p := &stubLLM{DefaultReply: issue2EnvelopeJSON(t, WikiActionEnvelope{Meta: []MetaAction{
		{Op: "rolling_memory_append", Domain: "general", Target: "learnings", Content: body},
		{Op: "sessions_log_append", SessionID: "forged"},
	}})}
	limited, err := mineSessionEnvelope(context.Background(), root, dbPath, "representative", p, loadConfig(root).SessionMine, "session-mine", "session-mine")
	if err != nil || limited {
		t.Fatalf("envelope failed: limited=%v error=%v", limited, err)
	}
	data := []byte(readKBFile(t, root, "projects/general/learnings.md"))
	raw, err := parseFrontmatterRaw(data)
	if err != nil || stringFromAny(raw["created"]) != "2026-10-01" || stringFromAny(raw["updated"]) != "2026-11-01" {
		t.Fatalf("rolling dates do not come from entry evidence: %v (%v)", raw, err)
	}
	if failures := validateArticleContent(root, data); len(failures) != 0 {
		t.Fatalf("envelope output invalid: %v", failures)
	}
	span, _ := frontmatterSpan(string(data))
	if !bytes.Equal(data[span.BodyStart:], []byte("\n"+body)) {
		t.Fatalf("metadata repair changed appended body: %q", data[span.BodyStart:])
	}
	sources, ok := raw["sources"].([]any)
	if !ok || len(sources) != 1 || sources[0] != "session:issue2-session" {
		t.Fatalf("source evidence not recovered: %v", raw["sources"])
	}
	view, _ := loadSessionsLogView(root)
	if view.processed["representative"] == nil || view.processed["forged"] != nil {
		t.Fatalf("wrong session completion markers: %v", view.processed)
	}
	t.Logf("complete envelope result:\n%s", data)
}

func TestIssue2EnvelopeParallelIsolationCheckpoint(t *testing.T) {
	root, _ := issue2EnvelopeKB(t, "s3-failed", "s3-good", "s3-third")
	path := "projects/acme/learnings.md"
	writeKBFile(t, root, path, issue2ValidRolling())
	writeKBFile(t, root, "wiki/collision.md", "Curated article.\n")
	writeKBFile(t, root, "scribe.yaml", strings.Replace(readKBFile(t, root, "scribe.yaml"), "checkpoint_interval: 100", "checkpoint_interval: 1", 1))
	mustGit(t, root, "init", "-q")
	mustGit(t, root, "add", ".")
	mustGit(t, root, "commit", "-qm", "seed")
	head := mustGit(t, root, "rev-parse", "HEAD")
	failed := issue2EnvelopeJSON(t, WikiActionEnvelope{Actions: []WikiAction{{Op: "create", Path: "wiki/collision.md", Content: "New grounded candidate."}}, Meta: []MetaAction{{Op: "rolling_memory_append", Domain: "acme", Target: "learnings", Content: "## 2026-11-01 | Failed\nfailed-only session:s3-failed"}}})
	good := issue2EnvelopeJSON(t, WikiActionEnvelope{Meta: []MetaAction{{Op: "rolling_memory_append", Domain: "acme", Target: "learnings", Content: "## 2026-11-01 | Good\nsuccessful-only session:s3-good"}}})
	badCorrection := issue2EnvelopeJSON(t, WikiActionEnvelope{Actions: []WikiAction{{Op: "append", Path: "wiki/unrelated.md", Content: "Forbidden redirect."}}})
	entered := make(chan string, 2)
	badGeneration, goodGeneration, goodReturned := make(chan struct{}), make(chan struct{}), make(chan struct{})
	mutation, failApply, thirdEntered, thirdRelease := make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})
	p := &issue2EnvelopeProvider{generate: func(ctx context.Context, prompt string) (string, error) {
		switch {
		case strings.HasPrefix(prompt, "The candidate create actions"):
			close(mutation)
			if err := issue2ProviderWait(ctx, failApply); err != nil {
				return "", err
			}
			return badCorrection, nil
		case strings.Contains(prompt, "s3-failed"):
			entered <- "failed"
			if err := issue2ProviderWait(ctx, badGeneration); err != nil {
				return "", err
			}
			return failed, nil
		case strings.Contains(prompt, "s3-good"):
			entered <- "good"
			if err := issue2ProviderWait(ctx, goodGeneration); err != nil {
				return "", err
			}
			close(goodReturned)
			return good, nil
		case strings.Contains(prompt, "s3-third"):
			if err := issue2ProviderWait(ctx, goodReturned); err != nil {
				return "", err
			}
			close(thirdEntered)
			if err := issue2ProviderWait(ctx, thirdRelease); err != nil {
				return "", err
			}
			return `{"version":2,"actions":[]}`, nil
		}
		return "", fmt.Errorf("unexpected fixture prompt")
	}}
	installStubLLM(t, p)
	cmd := &SyncCmd{Model: "sonnet"}
	done, finished := make(chan int, 1), make(chan struct{})
	go func() {
		defer close(finished)
		mined, _ := cmd.mineSessionBatches(root, []string{"s3-failed", "s3-good", "s3-third"}, 2, time.Minute, "session-extract.md", "session")
		done <- mined
	}()
	defer func() {
		for _, ch := range []chan struct{}{badGeneration, goodGeneration, failApply, thirdRelease} {
			issue2Release(ch)
		}
		issue2WaitSignal(t, finished)
	}()
	seen := make(map[string]bool)
	for range 2 {
		select {
		case sid := <-entered:
			seen[sid] = true
		case <-time.After(5 * time.Second):
			t.Fatal("two primary generations did not overlap")
		}
	}
	if !seen["failed"] || !seen["good"] {
		t.Fatalf("unexpected primary generations: %v", seen)
	}
	close(badGeneration)
	issue2WaitSignal(t, mutation)
	if !strings.Contains(readKBFile(t, root, path), "failed-only") {
		t.Fatal("failed session did not actually mutate the shared rolling file")
	}
	issue2AssertPending(t, root, "s3-failed")
	close(goodGeneration)
	issue2WaitSignal(t, thirdEntered)
	issue2AssertPending(t, root, "s3-good")
	if strings.Contains(readKBFile(t, root, path), "successful-only") || mustGit(t, root, "rev-parse", "HEAD") != head {
		t.Fatal("sibling apply or checkpoint overlapped the unvalidated failed session")
	}
	close(failApply)
	deadline := time.Now().Add(5 * time.Second)
	for mustGit(t, root, "rev-parse", "HEAD") == head {
		if time.Now().After(deadline) {
			t.Fatal("validated sibling did not reach real checkpoint")
		}
		time.Sleep(10 * time.Millisecond)
	}
	committed := mustGit(t, root, "show", "HEAD:"+path)
	log := mustGit(t, root, "show", "HEAD:wiki/_sessions_log.json")
	if !strings.Contains(committed, "successful-only") || strings.Contains(committed, "failed-only") || len(validateArticleContent(root, []byte(committed))) != 0 || !strings.Contains(log, `"s3-good"`) || strings.Contains(log, `"s3-failed"`) || strings.Contains(log, `"s3-third"`) {
		t.Fatalf("checkpoint contains failed/unvalidated output or mismatched markers:\n%s\n%s", committed, log)
	}
	close(thirdRelease)
	if mined := <-done; mined != 2 || cmd.fatalSessionErr != nil {
		t.Fatalf("unexpected batch completion: mined=%d fatal=%v", mined, cmd.fatalSessionErr)
	}
	issue2AssertPending(t, root, "s3-failed")
}

func issue2CodexFixture(t *testing.T, root, sid string) *ScribeConfig {
	t.Helper()
	rollouts := t.TempDir()
	issue2WriteCodexSession(t, rollouts, sid)
	cfg := loadConfig(root)
	cfg.CodexSessionsDir = rollouts
	cfg.Codex.Mine = true
	cfg.Codex.MinScore = 0
	cfg.Codex.SessionsMax = 3
	return cfg
}

func issue2WriteCodexSession(t *testing.T, rollouts, sid string) {
	t.Helper()
	path := writeCodexRollout(t, rollouts, "2026", "11", "06", sid, "/work/acme")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString("{\"type\":\"response_item\",\"payload\":{\"type\":\"message\",\"role\":\"user\",\"content\":[{\"type\":\"input_text\",\"text\":\"Explain architecture decisions, tradeoffs, patterns and durable receipts for retries.\"}]}}\n")
	if closeErr := f.Close(); err != nil || closeErr != nil {
		t.Fatalf("write rollout: %v %v", err, closeErr)
	}
}

func issue2RunEnvelopeCaller(t *testing.T, caller, root string, env WikiActionEnvelope) (int, *stubLLM, error) {
	t.Helper()
	p := &stubLLM{DefaultReply: issue2EnvelopeJSON(t, env)}
	installStubLLM(t, p)
	cmd := &SyncCmd{}
	if caller == "codex" {
		mined, err := cmd.mineCodexSessions(root, issue2CodexFixture(t, root, "variant-session"))
		return mined, p, err
	}
	prompt, label := "session-extract.md", "session"
	if caller == "large" {
		prompt, label = "session-extract-large.md", "large-session"
	}
	mined, _ := cmd.mineSessionBatches(root, []string{"variant-session"}, 2, time.Minute, prompt, label)
	return mined, p, cmd.fatalSessionErr
}

func TestIssue2EnvelopeVariants(t *testing.T) {
	for _, caller := range []string{"normal", "large", "codex"} {
		t.Run(caller, func(t *testing.T) {
			root, _ := issue2EnvelopeKB(t, "variant-session")
			writeKBFile(t, root, "scribe.yaml", readKBFile(t, root, "scribe.yaml")+"meta:\n  rolling_targets: [risk-notes]\n")
			body := "\n````markdown\n```\n## 2000-01-01 | Example\nsession:example\n````\n" + issue2RollingBody
			curated := strings.Replace(issue2ValidRolling(), "title: Curated lessons\n", "", 1)
			untouched := "---\nrolling: true\n---\nUntouched invalid file.\n"
			writeKBFile(t, root, "projects/other/learnings.md", untouched)
			writeKBFile(t, root, "projects/acme/risk-notes-archive-2025.md", curated)
			env := WikiActionEnvelope{Actions: []WikiAction{
				{Op: "create", Path: "projects/acme/decisions-log-archive-2025.md", Content: body},
				{Op: "create", Path: "projects/acme/risk-notes.md", Content: "---\n---\n" + body},
				{Op: "create", Path: "patterns/personal/évidence.md", Content: "---\nrolling: true\n---\n" + body},
				{Op: "append", Path: "projects/acme/risk-notes-archive-2025.md", Content: body},
				{Op: "create", Path: "tools/ordinary.md", Content: "---\ntitle: Ordinary\ntype: invented\ndomain: invented\n---\nOrdinary body.\n"},
			}}
			mined, p, err := issue2RunEnvelopeCaller(t, caller, root, env)
			if mined != 1 || err != nil || len(p.Calls()) != 1 {
				t.Fatalf("variant caller failed: mined=%d error=%v calls=%d", mined, err, len(p.Calls()))
			}
			if caller == "large" && !strings.HasPrefix(p.Calls()[0].Prompt, "You are extracting knowledge from one coding-agent session for the project") {
				t.Fatal("large dispatch did not use its envelope prompt family")
			}
			for _, tc := range []struct{ path, title, domain string }{
				{"projects/acme/decisions-log-archive-2025.md", "Acme Decisions Log Archive 2025", "acme"},
				{"projects/acme/risk-notes.md", "Acme Risk Notes", "acme"},
				{"patterns/personal/évidence.md", "Personal Évidence", "personal"},
			} {
				data := []byte(readKBFile(t, root, tc.path))
				raw, _ := parseFrontmatterRaw(data)
				span, _ := frontmatterSpan(string(data))
				sources, ok := raw["sources"].([]any)
				if len(validateArticleContent(root, data)) != 0 || raw["title"] != tc.title || raw["domain"] != tc.domain || stringFromAny(raw["created"]) != "2026-10-01" || stringFromAny(raw["updated"]) != "2026-11-01" || !ok || len(sources) != 1 || sources[0] != "session:issue2-session" || !bytes.Equal(data[span.BodyStart:], []byte(body)) {
					t.Fatalf("%s violates rolling metadata/body policy: %v", tc.path, raw)
				}
			}
			data := readKBFile(t, root, "projects/acme/risk-notes-archive-2025.md")
			if strings.Replace(data, "title: Acme Risk Notes Archive 2025\n", "", 1) != curated+body || readKBFile(t, root, "projects/other/learnings.md") != untouched {
				t.Fatal("repair changed curated fields/body or untouched invalid file")
			}
			ordinary, _ := parseFrontmatterRaw([]byte(readKBFile(t, root, "tools/ordinary.md")))
			if ordinary["type"] != "tool" || ordinary["domain"] != "general" {
				t.Fatal("session rolling policy disabled ordinary-article normalization")
			}
		})
	}
}

func TestIssue2EnvelopeRejectsMetadata(t *testing.T) {
	for _, caller := range []string{"normal", "large", "codex"} {
		for _, tc := range []struct{ name, header, body string }{
			{"false-marker", "rolling: false\n", issue2RollingBody},
			{"malformed", "rolling: true\ntags: [\n", issue2RollingBody},
			{"invalid-type", "rolling: true\ntype: invented\n", issue2RollingBody},
			{"invalid-domain", "rolling: true\ndomain: invented\n", issue2RollingBody},
			{"invalid-confidence", "rolling: true\nconfidence: certain\n", issue2RollingBody},
			{"contradictory", "rolling: true\ncreated: 2026-11-02\n", issue2RollingBody},
			{"unresolved-dates", "rolling: true\n", "No dated entries.\n"},
		} {
			t.Run(caller+"/"+tc.name, func(t *testing.T) {
				root, _ := issue2EnvelopeKB(t, "variant-session")
				path := "patterns/personal/notes.md"
				if tc.name == "false-marker" {
					path = "projects/acme/decisions-log.md"
				}
				env := WikiActionEnvelope{Actions: []WikiAction{{Op: "create", Path: path, Content: "---\n" + tc.header + "---\n" + tc.body}}, Meta: []MetaAction{{Op: "rolling_memory_append", Domain: "acme", Target: "learnings", Content: strings.TrimPrefix(issue2RollingBody, "\n")}}}
				mined, p, err := issue2RunEnvelopeCaller(t, caller, root, env)
				if mined != 0 || err != nil || len(p.Calls()) != 1 || fileExists(filepath.Join(root, path)) || fileExists(filepath.Join(root, "projects/acme/learnings.md")) {
					t.Fatalf("invalid session survived or was counted: mined=%d error=%v calls=%d", mined, err, len(p.Calls()))
				}
				if loadProcessedCodexIDs(root)["variant-session"] {
					t.Fatal("invalid session entered Codex log")
				}
				issue2AssertPending(t, root, "variant-session")
			})
		}
	}
}

func TestIssue2EnvelopeCodexCaller(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		t.Run(fmt.Sprintf("invalid=%v", invalid), func(t *testing.T) {
			root, _ := issue2EnvelopeKB(t)
			cfg := issue2CodexFixture(t, root, "codex-rolling")
			env := WikiActionEnvelope{Meta: []MetaAction{{Op: "rolling_memory_append", Domain: "acme", Target: "decisions-log", Content: strings.TrimPrefix(issue2RollingBody, "\n")}, {Op: "sessions_log_append", SessionID: "forged"}}}
			if invalid {
				env.Actions = []WikiAction{{Op: "create", Path: "patterns/personal/notes.md", Content: "---\nrolling: true\nconfidence: certain\n---\n" + issue2RollingBody}}
			}
			p := &stubLLM{DefaultReply: issue2EnvelopeJSON(t, env)}
			installStubLLM(t, p)
			cmd := &SyncCmd{}
			mined, err := cmd.mineCodexSessions(root, cfg)
			if err != nil || len(p.Calls()) != 1 {
				t.Fatalf("Codex fixture did not reach envelope: calls=%d error=%v", len(p.Calls()), err)
			}
			processed := loadProcessedCodexIDs(root)
			if invalid {
				if mined != 0 || processed["codex-rolling"] || fileExists(filepath.Join(root, "patterns/personal/notes.md")) || fileExists(filepath.Join(root, "projects/acme/decisions-log.md")) {
					t.Fatal("Codex invalid rolling output survived or was marked processed")
				}
			} else {
				data := []byte(readKBFile(t, root, "projects/acme/decisions-log.md"))
				raw, _ := parseFrontmatterRaw(data)
				if mined != 1 || !processed["codex-rolling"] || stringFromAny(raw["created"]) != "2026-10-01" || stringFromAny(raw["updated"]) != "2026-11-01" || len(validateArticleContent(root, data)) != 0 {
					t.Fatalf("Codex completion lacks validated entry provenance: mined=%d log=%v metadata=%v", mined, processed, raw)
				}
			}
			view, _ := loadSessionsLogView(root)
			if len(view.processed) != 0 || processed["forged"] {
				t.Fatal("Codex envelope touched another caller's processed log")
			}
		})
	}
}

func TestIssue2EnvelopeMarkerFailureRestores(t *testing.T) {
	for _, caller := range []string{"ccrider", "codex"} {
		t.Run(caller, func(t *testing.T) {
			root, _ := issue2EnvelopeKB(t, "marker-failure")
			path := "projects/acme/decisions-log.md"
			writeKBFile(t, root, path, issue2ValidRolling())
			writeKBFile(t, root, "wiki/collision.md", "Curated article.\n")
			logPath := "wiki/_sessions_log.json"
			if caller == "codex" {
				logPath = "wiki/_codex_sessions_log.json"
			}
			writeKBFile(t, root, logPath, "not-json\n")
			cfg := loadConfig(root)
			if caller == "codex" {
				cfg = issue2CodexFixture(t, root, "marker-failure")
			}
			initial := issue2EnvelopeJSON(t, WikiActionEnvelope{Actions: []WikiAction{{Op: "create", Path: "wiki/collision.md", Content: "A source-grounded addition."}}, Meta: []MetaAction{{Op: "rolling_memory_append", Domain: "acme", Target: "decisions-log", Content: "## 2026-11-01 | Pending\nmarker-failure-output"}}})
			mutation, release := make(chan struct{}), make(chan struct{})
			p := &issue2EnvelopeProvider{generate: func(ctx context.Context, prompt string) (string, error) {
				if strings.HasPrefix(prompt, "The candidate create actions") {
					close(mutation)
					if err := issue2ProviderWait(ctx, release); err != nil {
						return "", err
					}
					return `{"version":2,"actions":[]}`, nil
				}
				return initial, nil
			}}
			installStubLLM(t, p)
			cmd := &SyncCmd{}
			done, finished := make(chan int, 1), make(chan struct{})
			go func() {
				defer close(finished)
				if caller == "codex" {
					mined, _ := cmd.mineCodexSessions(root, cfg)
					done <- mined
				} else {
					mined, _ := cmd.mineSessionBatches(root, []string{"marker-failure"}, 1, time.Minute, "session-extract.md", "session")
					done <- mined
				}
			}()
			defer func() { issue2Release(release); issue2WaitSignal(t, finished) }()
			issue2WaitSignal(t, mutation)
			if !strings.Contains(readKBFile(t, root, path), "marker-failure-output") {
				t.Fatal("fixture did not mutate before marker-write failure")
			}
			close(release)
			if mined := <-done; mined != 0 || cmd.fatalSessionErr != nil || readKBFile(t, root, path) != issue2ValidRolling() || readKBFile(t, root, logPath) != "not-json\n" {
				t.Fatalf("failed marker write claimed success or retained rolling changes: mined=%d fatal=%v", mined, cmd.fatalSessionErr)
			}
		})
	}
}

func TestIssue2EnvelopeFatalStopsSync(t *testing.T) {
	for _, caller := range []string{"normal", "large", "codex"} {
		t.Run(caller, func(t *testing.T) {
			root, _ := issue2EnvelopeKB(t, "fatal", "later")
			path := "projects/acme/learnings.md"
			writeKBFile(t, root, path, issue2ValidRolling())
			writeKBFile(t, root, "wiki/collision.md", "Curated article.\n")
			cfg := issue2CodexFixture(t, root, "z-fatal")
			issue2WriteCodexSession(t, cfg.CodexSessionsDir, "a-later")
			writeKBFile(t, root, "scribe.yaml", readKBFile(t, root, "scribe.yaml")+"codex_sessions_dir: "+cfg.CodexSessionsDir+"\ncodex:\n  mine: true\n  sessions_max: 3\n  min_score: 1\n")
			head := issue2PrepareSync(t, root)
			if caller != "normal" {
				executable, _ := sessionExecutable()
				triage := "#!/bin/sh\nprintf '%s\\n' '[]'\n"
				if caller == "large" {
					triage = "#!/bin/sh\ncase \" $* \" in\n*' --min-messages 301 '*) printf '%s\\n' '[{\"session_id\":\"fatal\",\"total_score\":100,\"msgs\":600},{\"session_id\":\"later\",\"total_score\":90,\"msgs\":500}]';;\n*) printf '%s\\n' '[]';;\nesac\n"
				}
				if err := os.WriteFile(executable, []byte(triage), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			initial := issue2EnvelopeJSON(t, WikiActionEnvelope{Actions: []WikiAction{{Op: "create", Path: "wiki/collision.md", Content: "New grounded candidate."}}, Meta: []MetaAction{{Op: "rolling_memory_append", Domain: "acme", Target: "learnings", Content: "## 2026-11-01 | Pending\nfatal-output"}}})
			mutation, release := make(chan struct{}), make(chan struct{})
			calls := make(chan string, 8)
			p := &issue2EnvelopeProvider{generate: func(ctx context.Context, prompt string) (string, error) {
				if strings.HasPrefix(prompt, "The candidate create actions") {
					close(mutation)
					if err := issue2ProviderWait(ctx, release); err != nil {
						return "", err
					}
					return `{"version":2,"actions":[{"op":"append","path":"wiki/forbidden.md","content":"redirect"}]}`, nil
				}
				calls <- opLabelFromContext(ctx)
				if strings.Contains(prompt, "fatal") {
					return initial, nil
				}
				if err := issue2ProviderWait(ctx, release); err != nil {
					return "", err
				}
				return `{"version":2,"actions":[]}`, nil
			}}
			installStubLLM(t, p)
			cmd := &SyncCmd{Sessions: true, SessionsMax: 6, SessionSort: "score", Parallel: 1, Model: "sonnet"}
			done, finished := make(chan error, 1), make(chan struct{})
			go func() { defer close(finished); done <- cmd.Run() }()
			defer func() { issue2Release(release); issue2WaitSignal(t, finished) }()
			issue2WaitSignal(t, mutation)
			if !strings.Contains(readKBFile(t, root, path), "fatal-output") {
				t.Fatal("fixture did not mutate rolling output before the restoration fault")
			}
			if err := os.Rename(filepath.Join(root, "projects/acme"), filepath.Join(root, "projects/saved")); err != nil {
				t.Fatal(err)
			}
			writeKBFile(t, root, "projects/acme", "blocked parent")
			close(release)
			err := <-done
			if !errors.Is(err, errRollingRestore) || mustGit(t, root, "rev-parse", "HEAD") != head || strings.TrimSpace(mustGit(t, root, "diff", "--cached", "--name-only")) != "" {
				t.Fatalf("fatal envelope restoration did not stop staging/final commit: %v", err)
			}
			issue2AssertPending(t, root, "fatal")
			issue2AssertPending(t, root, "later")
			if loadProcessedCodexIDs(root)["z-fatal"] || loadProcessedCodexIDs(root)["a-later"] {
				t.Fatal("fatal envelope still recorded Codex success")
			}
			close(calls)
			var codexCalls int
			for op := range calls {
				if op == "codex-session" {
					codexCalls++
				}
			}
			if caller == "codex" && codexCalls != 1 || caller != "codex" && codexCalls != 0 {
				t.Fatalf("fatal stop continued serial Codex mining: %d calls", codexCalls)
			}
		})
	}
}

func TestIssue2EnvelopeSkipAndLimits(t *testing.T) {
	for _, caller := range []string{"normal", "large", "codex"} {
		for _, outcome := range []string{"no-actions", "rate", "budget", "retry-budget"} {
			t.Run(caller+"/"+outcome, func(t *testing.T) {
				root, _ := issue2EnvelopeKB(t, "variant-session")
				p := &stubLLM{DefaultReply: `{"version":2,"actions":[]}`}
				if outcome == "rate" {
					p.DefaultErr = ErrRateLimit
				}
				if outcome == "budget" || outcome == "retry-budget" {
					p.DefaultErr = ErrDailyBudgetExhausted
				}
				if outcome == "retry-budget" {
					p.Rules = []*stubRule{{Reply: "invalid envelope", Once: true}}
				}
				installStubLLM(t, p)
				cmd := &SyncCmd{}
				mined, limited := 0, false
				var err error
				if caller == "codex" {
					mined, err = cmd.mineCodexSessions(root, issue2CodexFixture(t, root, "variant-session"))
					limited = err != nil
				} else {
					prompt, label := "session-extract.md", "session"
					if caller == "large" {
						prompt, label = "session-extract-large.md", "large-session"
					}
					mined, limited = cmd.mineSessionBatches(root, []string{"variant-session"}, 1, time.Minute, prompt, label)
				}
				view, _ := loadSessionsLogView(root)
				processed := view.processed["variant-session"] != nil || loadProcessedCodexIDs(root)["variant-session"]
				if outcome == "no-actions" {
					if mined != 1 || limited || !processed {
						t.Fatalf("empty envelope must complete: mined=%d limited=%v error=%v", mined, limited, err)
					}
				} else if mined != 0 || !limited || processed {
					t.Fatalf("limit must remain pending and stop: mined=%d limited=%v error=%v", mined, limited, err)
				}
			})
		}
	}
	for _, outcome := range []string{"ccrider-empty", "codex-empty", "codex-low-score"} {
		t.Run(outcome, func(t *testing.T) {
			root, _ := issue2EnvelopeKB(t)
			p := &stubLLM{DefaultErr: fmt.Errorf("a skipped transcript must not call a model")}
			installStubLLM(t, p)
			cmd := &SyncCmd{}
			if outcome == "ccrider-empty" {
				mined, limited := cmd.mineSessionBatches(root, []string{"empty"}, 1, time.Minute, "session-extract.md", "session")
				view, _ := loadSessionsLogView(root)
				if mined != 1 || limited || view.processed["empty"] == nil {
					t.Fatal("empty ccrider transcript did not retain skip completion")
				}
			} else {
				cfg := issue2CodexFixture(t, root, "skip")
				if outcome == "codex-empty" {
					path := writeCodexRollout(t, cfg.CodexSessionsDir, "2026", "11", "06", "skip", "/work/acme")
					if _, err := os.Stat(path); err != nil {
						t.Fatal(err)
					}
				} else {
					cfg.Codex.MinScore = 1000000
				}
				mined, err := cmd.mineCodexSessions(root, cfg)
				if mined != 0 || err != nil || !loadProcessedCodexIDs(root)["skip"] {
					t.Fatalf("Codex skip failed: mined=%d error=%v", mined, err)
				}
			}
			if len(p.Calls()) != 0 {
				t.Fatal("skip path called the provider")
			}
		})
	}
}
