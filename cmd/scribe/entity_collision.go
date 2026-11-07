package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

var entityWriterMu sync.Mutex

func applyEntityActions(root string, env WikiActionEnvelope, opts ApplyOptions) (ApplyResult, error) {
	entityWriterMu.Lock()
	defer entityWriterMu.Unlock()
	return applyWikiActions(root, env, opts)
}

// A create collision needs the existing article as evidence before a safe append.
func applyEntityEnvelope(ctx context.Context, root string, provider llmProviderGenerator, env WikiActionEnvelope) (ApplyResult, error) {
	return applyEntityEnvelopeWithOptions(ctx, root, provider, env, entityWriterApplyOptions())
}

func applyEntityEnvelopeWithOptions(ctx context.Context, root string, provider llmProviderGenerator, env WikiActionEnvelope, opts ApplyOptions) (ApplyResult, error) {
	res, err := applyEntityActions(root, env, opts)
	if err != nil || len(res.Errors) == 0 {
		return res, err
	}
	allowed := make(map[string]bool)
	var packet strings.Builder
	for _, failure := range res.Errors {
		var index int
		if _, err := fmt.Sscanf(failure, "action[%d]", &index); err != nil || index < 0 || index >= len(env.Actions) || !strings.Contains(failure, "file exists and AllowOverwrite=false") {
			return res, nil
		}
		a := env.Actions[index]
		path, err := validateActionPath(root, a.Path)
		if err != nil {
			return res, nil
		}
		existing, err := readRegularFile(path)
		if err != nil {
			return res, nil
		}
		candidate, err := json.Marshal(a)
		if err != nil {
			return res, err
		}
		if packet.Len()+len(existing)+len(candidate) > 128000 {
			return res, nil
		}
		fmt.Fprintf(&packet, "\nCandidate: %s\nExisting article %s:\n%s\n", candidate, a.Path, existing)
		allowed[a.Path] = true
	}
	prompt := "The candidate create actions below collided with existing KB articles. Compare each candidate with its existing article. Return one version-2 JSON envelope containing only append actions for genuinely new, source-grounded material on those exact existing paths. Preserve existing content and provenance. If the material is already covered, return actions: []. Return meta: []. All other original actions and metadata already succeeded and must not be repeated. Use no tools.\n" + packet.String()
	callCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	out, err := generateMaybeJSON(callCtx, provider, prompt)
	if err != nil {
		return res, err
	}
	correction, err := parseEnvelopeV2(out, "create-collision")
	if err != nil {
		return res, err
	}
	if len(correction.Meta) != 0 {
		return res, fmt.Errorf("collision correction must not repeat metadata")
	}
	for _, a := range correction.Actions {
		if a.Op != "append" || !allowed[a.Path] {
			return res, fmt.Errorf("collision correction must only append to rejected create paths")
		}
	}
	corrected, err := applyEntityActions(root, correction, opts)
	res.Applied = append(res.Applied, corrected.Applied...)
	res.Errors = corrected.Errors
	return res, err
}
