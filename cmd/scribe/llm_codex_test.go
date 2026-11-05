package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

var codexLiveTest = flag.Bool("codex-live-test", false, "Call the authenticated Codex CLI")

func TestCodexLiveParallel(t *testing.T) {
	if !*codexLiveTest {
		t.Skip("pass -codex-live-test to call the authenticated Codex CLI")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	provider := &codexProvider{model: "gpt-6.1-terra", effort: "medium"}
	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() {
			reply, err := provider.Generate(ctx, "Return exactly the JSON object {\"ok\":true}.")
			if err != nil {
				t.Errorf("live Codex call: %v", err)
				return
			}
			if strings.TrimSpace(reply) != `{"ok":true}` {
				t.Errorf("unexpected response: %q", reply)
			}
		})
	}
	wg.Wait()
}

func TestCodexProviderCLI(t *testing.T) {
	for _, mode := range []string{"ok", "failure", "event-error", "incomplete", "empty", "timeout"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			script := `#!/bin/sh
printf '%s\n' "$@" > "$CODEX_TEST_CAPTURE/args"
while [ "$#" -gt 0 ]; do
  if [ "$1" = '--output-last-message' ]; then shift; output="$1"; fi
  shift
done
cat > "$CODEX_TEST_CAPTURE/prompt"
case "$CODEX_TEST_MODE" in
  failure) echo 'authentication failed' >&2; exit 1 ;;
  event-error) echo '{"type":"turn.failed"}'; exit 0 ;;
  incomplete) echo '{"type":"thread.started"}'; exit 0 ;;
  timeout) exec sleep 10 ;;
  empty) : > "$output" ;;
  *) printf '%s' 'A document about rate limits and errors.' > "$output" ;;
esac
echo '{"type":"turn.completed","usage":{"input_tokens":12,"output_tokens":8,"cached_input_tokens":3}}'
`
			if err := os.WriteFile(filepath.Join(dir, "codex"), []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("CODEX_TEST_CAPTURE", dir)
			t.Setenv("CODEX_TEST_MODE", mode)
			ctx := context.Background()
			if mode == "timeout" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()
			}
			provider := &codexProvider{model: "gpt-6.1-terra", effort: "medium"}
			reply, err := provider.Generate(ctx, "private source text")
			if mode != "ok" {
				if err == nil {
					t.Fatal("expected failed generation")
				}
				if mode == "timeout" && !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("expected deadline error, got %v", err)
				}
				return
			}
			if err != nil || reply != "A document about rate limits and errors." {
				t.Fatalf("reply=%q err=%v", reply, err)
			}
			args, _ := os.ReadFile(filepath.Join(dir, "args"))
			for _, want := range []string{"--model\ngpt-6.1-terra", "model_reasoning_effort=\"medium\"", "--ephemeral", "--ignore-user-config", "--sandbox\nread-only", "features.shell_tool=false", "features.plugins=false"} {
				if !strings.Contains(string(args), want) {
					t.Errorf("missing argument %q", want)
				}
			}
			if strings.Contains(string(args), "private source text") {
				t.Fatal("prompt leaked to argv")
			}
			input, _ := os.ReadFile(filepath.Join(dir, "prompt"))
			if string(input) != "private source text" {
				t.Fatalf("stdin=%q", input)
			}
		})
	}
}

func TestCodexRoutingAndParallelism(t *testing.T) {
	root := t.TempDir()
	config := "llm:\n  codex_reasoning_effort: medium\nextract:\n  provider: codex\n  model: gpt-6.1-terra\nsession_mine:\n  provider: codex\n  model: gpt-6.1-terra\nsync:\n  parallel_extractions: 5\n"
	if err := os.WriteFile(filepath.Join(root, "scribe.yaml"), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := loadConfig(root)
	if cfg.Extract.Mode != "envelope" || cfg.SessionMine.Mode != "envelope" {
		t.Fatal("Codex operations must use Go-owned envelope writes")
	}
	for _, model := range []string{cfg.Extract.Model, cfg.SessionMine.Model} {
		provider, ok := realNewLLMProvider("codex", model, "", root).(*codexProvider)
		if !ok || provider.model != "gpt-6.1-terra" || provider.effort != "medium" {
			t.Fatalf("unexpected provider: %#v", provider)
		}
	}
	for _, tc := range []struct{ cli, configured, want int }{
		{0, 5, 5}, {2, 5, 2}, {5, 3, 5}, {9, 5, 5}, {-1, 5, 1}, {0, 0, 3},
	} {
		cfg.Sync.ParallelExtractions = tc.configured
		if got := (&SyncCmd{Parallel: tc.cli}).parallelism(cfg); got != tc.want {
			t.Errorf("cli=%d config=%d: got %d want %d", tc.cli, tc.configured, got, tc.want)
		}
	}
}
