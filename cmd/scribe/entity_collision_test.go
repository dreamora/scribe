package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEntityCollisionAppendsWithoutRepeatingSuccess(t *testing.T) {
	root := sessionsTestKB(t, "")
	path := filepath.Join(root, "wiki/existing.md")
	if err := os.WriteFile(path, []byte("Curated original.\n"), 0600); err != nil {
		t.Fatal(err)
	}
	env := WikiActionEnvelope{Version: 2, Actions: []WikiAction{
		{Op: "append", Path: "wiki/accepted.md", Content: "Only once."},
		{Op: "create", Path: "wiki/existing.md", Content: "New grounded insight."},
	}}
	correction, _ := json.Marshal(WikiActionEnvelope{Version: 2, Actions: []WikiAction{{Op: "append", Path: "wiki/existing.md", Content: "\nNew grounded insight."}}})
	p := &stubLLM{DefaultReply: string(correction)}
	res, err := applyEntityEnvelope(context.Background(), root, p, env)
	if err != nil || len(res.Errors) > 0 {
		t.Fatalf("%v %v", err, res.Errors)
	}
	data, _ := os.ReadFile(path)
	if !strings.HasPrefix(string(data), "Curated original.") || !strings.Contains(string(data), "New grounded insight.") {
		t.Fatal("existing content lost")
	}
	accepted, _ := os.ReadFile(filepath.Join(root, "wiki/accepted.md"))
	if strings.Count(string(accepted), "Only once.") != 1 {
		t.Fatal("repeated successful action")
	}
	if calls := p.Calls(); len(calls) != 1 || !strings.Contains(calls[0].Prompt, "Curated original.") {
		t.Fatal("correction missing existing evidence")
	}
}

func TestEntityCollisionCannotRedirectOrOverwrite(t *testing.T) {
	for _, a := range []WikiAction{{Op: "create", Path: "wiki/existing.md", Content: "clobber"}, {Op: "append", Path: "wiki/unrelated.md", Content: "redirect"}} {
		root := sessionsTestKB(t, "")
		path := filepath.Join(root, "wiki/existing.md")
		os.WriteFile(path, []byte("Keep."), 0600)
		env := WikiActionEnvelope{Version: 2, Actions: []WikiAction{{Op: "create", Path: "wiki/existing.md", Content: "Candidate."}}}
		b, _ := json.Marshal(WikiActionEnvelope{Version: 2, Actions: []WikiAction{a}})
		_, err := applyEntityEnvelope(context.Background(), root, &stubLLM{DefaultReply: string(b)}, env)
		if err == nil {
			t.Fatal("unsafe correction accepted")
		}
		data, _ := os.ReadFile(path)
		if string(data) != "Keep." {
			t.Fatal("original overwritten")
		}
	}
}
