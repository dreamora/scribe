package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIssue2ToolsMarkerFailureRestoresBeforeCheckpoint(t *testing.T) {
	for _, prompt := range []string{"session-extract.md", "session-extract-large.md"} {
		t.Run(prompt, func(t *testing.T) {
			control := t.TempDir()
			t.Setenv("ISSUE2_CONTROL", control)
			script := "case \"$prompt\" in\n*'marker-failed'*)\n" + issue2CreateScript() + issue2WriteScript("projects/acme/decisions-log.md", issue2ValidRolling()) + issue2BarrierScript() + ";;\nesac\n"
			root := issue2ToolsKB(t, script)
			original := issue2ValidRolling()
			writeKBFile(t, root, "projects/acme/learnings.md", original)
			damaged := "original damaged marker log\n"
			writeKBFile(t, root, "wiki/_sessions_log.json", damaged)
			writeKBFile(t, root, "scribe.yaml", strings.Replace(readKBFile(t, root, "scribe.yaml"), "checkpoint_interval: 100", "checkpoint_interval: 1", 1))
			t.Setenv("SCRIBE_SKIP_REINDEX", "1")
			mustGit(t, root, "init", "-q")
			mustGit(t, root, "add", ".")
			mustGit(t, root, "commit", "-qm", "seed")
			head := mustGit(t, root, "rev-parse", "HEAD")
			cmd := &SyncCmd{Model: "sonnet"}
			done, finished := make(chan int, 1), make(chan struct{})
			go func() {
				defer close(finished)
				mined, _ := cmd.mineSessionBatches(root, []string{"marker-failed", "later"}, 3, 5*time.Second, prompt, "session")
				done <- mined
			}()
			defer func() {
				writeKBFile(t, control, "release", "cleanup")
				issue2WaitSignal(t, finished)
			}()
			issue2WaitForFile(t, filepath.Join(control, "ready"))
			if readKBFile(t, root, "projects/acme/learnings.md") == original || !fileExists(filepath.Join(root, "projects/acme/decisions-log.md")) {
				t.Fatal("fixture did not mutate existing and new rolling files before marker failure")
			}
			issue2AssertPending(t, root, "marker-failed")
			writeKBFile(t, control, "release", "complete child successfully")
			mined := <-done
			if mined != 0 || cmd.fatalSessionErr != nil {
				t.Errorf("marker persistence failure counted as completion: mined=%d fatal=%v", mined, cmd.fatalSessionErr)
			}
			if readKBFile(t, root, "projects/acme/learnings.md") != original || fileExists(filepath.Join(root, "projects/acme/decisions-log.md")) {
				t.Error("marker persistence failure did not restore/remove session rolling output")
			}
			if got := mustGit(t, root, "rev-parse", "HEAD"); got != head {
				t.Errorf("checkpoint committed output without its marker: HEAD=%s committed log=%q", got, mustGit(t, root, "show", "HEAD:wiki/_sessions_log.json"))
			}
			if readKBFile(t, root, "wiki/_sessions_log.json") != damaged || strings.TrimSpace(mustGit(t, root, "diff", "--cached", "--name-only")) != "" {
				t.Error("marker failure changed damaged-log bytes or staged output")
			}
			issue2AssertPending(t, root, "marker-failed")
			issue2AssertPending(t, root, "later")
		})
	}
}
