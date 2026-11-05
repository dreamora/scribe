package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// codexProvider uses the authenticated CLI as a text-only backend. Scribe
// owns all KB writes; the child receives its bounded prompt over stdin.
type codexProvider struct {
	model, effort, root string
}

func (c *codexProvider) Name() string { return "codex/" + c.model }

func (c *codexProvider) Generate(ctx context.Context, prompt string) (string, error) {
	if cfg := loadConfig(c.root); cfg != nil {
		if err := checkBudget(c.root, effectiveOutputTokenCeiling(cfg.Sync)); err != nil {
			return "", err
		}
	}
	started := time.Now()
	entry := CostEntry{Timestamp: started.UTC().Format(time.RFC3339), Provider: "codex",
		Model: c.model, Op: opLabelFromContext(ctx), PromptChars: len(prompt), ErrKind: "other"}
	defer func() {
		entry.DurationMS = time.Since(started).Milliseconds()
		appendCostEntry(c.root, entry)
	}()

	dir, err := os.MkdirTemp("", "scribe-codex-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	output := filepath.Join(dir, "response.txt")
	effort := c.effort
	if effort == "" {
		effort = "medium"
	}
	args := []string{"--no-daemon", "exec", "--ignore-user-config", "--ignore-rules",
		"--skip-git-repo-check", "--ephemeral", "--sandbox", "read-only", "--json",
		"--model", c.model, "--output-last-message", output,
		"-c", "model_reasoning_effort=" + strconv.Quote(effort),
		"-c", "approval_policy=\"never\"", "-c", "project_doc_max_bytes=0",
		"-c", "developer_instructions=\"\"", "-c", "web_search=\"disabled\""}
	// Do not inherit MCP servers, plugins, skills, hooks, or agent tools.
	for _, feature := range []string{"shell_tool", "unified_exec", "code_mode", "code_mode_host",
		"view_image", "image_generation", "browser_use", "computer_use", "workspace_dependencies",
		"apps", "plugins", "remote_plugin", "hooks", "memories", "multi_agent", "multi_agent_v2",
		"skill_search", "skill_mcp_dependency_install", "shell_snapshot"} {
		args = append(args, "-c", "features."+feature+"=false")
	}
	args = append(args, "-c", "features.skip_host_skill_discovery=true", "-c", "agents.enabled=false", "-")
	cmd := exec.CommandContext(ctx, "codex", args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(prompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	cmd.WaitDelay = 5 * time.Second
	defer startHeartbeat(ctx, entry.Op)()
	runErr := cmd.Run()
	if isRateLimited(stderr.String()) {
		entry.ErrKind = "rate_limit"
		return "", ErrRateLimit
	}
	if runErr != nil {
		if ctx.Err() != nil {
			entry.ErrKind = "canceled"
			if ctx.Err() == context.DeadlineExceeded {
				entry.ErrKind = "timeout"
			}
			return "", ctx.Err()
		}
		return "", fmt.Errorf("codex exec: %w: %s", runErr, tailLines(stderr.String(), 5))
	}
	// Read only event metadata. Response text may itself discuss errors or
	// rate limits, so never classify failures by searching model output.
	scanner := bufio.NewScanner(&stdout)
	scanner.Buffer(make([]byte, 4096), 16<<20)
	completed := false
	for scanner.Scan() {
		var event struct {
			Type  string `json:"type"`
			Usage struct {
				Input  int64 `json:"input_tokens"`
				Output int64 `json:"output_tokens"`
				Cached int64 `json:"cached_input_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return "", fmt.Errorf("decode codex event: %w", err)
		}
		if event.Type == "turn.failed" || event.Type == "error" {
			return "", fmt.Errorf("codex reported %s", event.Type)
		}
		if event.Type == "turn.completed" {
			completed = true
			entry.InputTokens, entry.OutputTokens, entry.CacheReadTokens = &event.Usage.Input, &event.Usage.Output, &event.Usage.Cached
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read codex events: %w", err)
	}
	if !completed {
		return "", fmt.Errorf("codex did not complete its turn")
	}
	reply, err := os.ReadFile(output)
	if err != nil {
		return "", fmt.Errorf("read codex response: %w", err)
	}
	if strings.TrimSpace(string(reply)) == "" {
		return "", fmt.Errorf("codex returned an empty response")
	}
	entry.OK, entry.ErrKind = true, ""
	return string(reply), nil
}
