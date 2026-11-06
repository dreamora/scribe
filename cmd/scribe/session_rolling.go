package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var errRollingRestore = errors.New("rolling-file restoration failed; stop sync")

type rollingFileState struct {
	data []byte
	mode fs.FileMode
}

type rollingSnapshot map[string]rollingFileState

func captureRollingState(root string) (rollingSnapshot, error) {
	state := make(rollingSnapshot)
	for _, dir := range wikiDirs {
		base := filepath.Join(root, dir)
		info, err := os.Lstat(base)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return state, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return state, fmt.Errorf("content directory is a symlink: %s", dir)
		}
		err = filepath.WalkDir(base, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".md") && entry.Type()&os.ModeSymlink == 0) {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if info.Mode()&os.ModeSymlink != 0 {
				target, err := os.Readlink(path)
				if err != nil {
					return err
				}
				rel, err := filepath.Rel(root, path)
				if err == nil {
					state[rel] = rollingFileState{data: []byte(target), mode: info.Mode()}
				}
				return err
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, path)
			if err == nil {
				state[rel] = rollingFileState{data: data, mode: info.Mode().Perm()}
			}
			return err
		})
		if err != nil {
			return state, err
		}
	}
	return state, nil
}

func isSessionRollingFile(root, rel string, data []byte) bool {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) >= 3 && parts[0] == "projects" {
		stem := strings.TrimSuffix(parts[len(parts)-1], ".md")
		if i := strings.LastIndex(stem, "-archive-"); i >= 0 {
			if _, err := time.Parse("2006", stem[i+9:]); err == nil {
				stem = stem[:i]
			}
		}
		if allowedRollingTargetsForRoot(root)[stem] || validRollingTargets[stem] {
			return true
		}
	}
	raw, err := parseFrontmatterRaw(data)
	if err == nil {
		return raw["rolling"] == true
	}
	first, header, hasNL := strings.Cut(string(data), "\n")
	if !hasNL || !isBareFence(first) {
		return false
	}
	for line := range strings.SplitSeq(header, "\n") {
		if isBareFence(line) || isLooseFence(line) {
			break
		}
		if rollingDeclarationRE.MatchString(line) {
			return true
		}
	}
	return false
}

var rollingDeclarationRE = regexp.MustCompile(`^(?:rolling|"rolling"|'rolling')[ \t]*:`)

func changedRollingFiles(root string, before, after rollingSnapshot) []string {
	paths := make(map[string]bool)
	for rel, prior := range before {
		current, exists := after[rel]
		if (!exists || prior.mode != current.mode || !bytes.Equal(prior.data, current.data)) && (isSessionRollingFile(root, rel, prior.data) || isSessionRollingFile(root, rel, current.data)) {
			paths[rel] = true
		}
	}
	for rel, current := range after {
		if _, exists := before[rel]; !exists && isSessionRollingFile(root, rel, current.data) {
			paths[rel] = true
		}
	}
	result := make([]string, 0, len(paths))
	for rel := range paths {
		result = append(result, rel)
	}
	sort.Strings(result)
	return result
}

func restoreRollingState(root string, before, after rollingSnapshot) error {
	var failures []error
	for _, rel := range changedRollingFiles(root, before, after) {
		path, err := rollingRestorePath(root, rel)
		if err == nil {
			if original, exists := before[rel]; exists {
				if original.mode&os.ModeSymlink != 0 {
					err = errors.New("cannot restore a changed pre-existing symlink")
				} else {
					err = writeFileAtomic(path, original.data, original.mode)
				}
			} else {
				err = os.Remove(path)
				if errors.Is(err, os.ErrNotExist) {
					err = nil
				}
			}
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", rel, err))
		}
	}
	return errors.Join(failures...)
}

func rollingRestorePath(root, rel string) (string, error) {
	if _, err := validateActionPath(root, filepath.Join(filepath.Dir(rel), "rolling-restore-check.md")); err != nil {
		return "", err
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	path := filepath.Join(absRoot, rel)
	for ancestor := filepath.Dir(path); ; ancestor = filepath.Dir(ancestor) {
		if ancestor == absRoot {
			return path, nil
		}
		info, err := os.Lstat(ancestor)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return "", fmt.Errorf("restore ancestor is a symlink: %s", ancestor)
			}
			if !info.IsDir() {
				return "", fmt.Errorf("restore ancestor is not a directory: %s", ancestor)
			}
		}
	}
}

func runToolsRollingSession(root string, run func() error) error {
	before, err := captureRollingState(root)
	if err != nil {
		return fmt.Errorf("capture rolling files: %w", err)
	}
	for rel, current := range before {
		if current.mode&os.ModeSymlink == 0 {
			continue
		}
		if strings.HasSuffix(rel, ".md") {
			return fmt.Errorf("tools mining does not support a Markdown symlink: %s", rel)
		}
		target, err := os.Stat(filepath.Join(root, rel))
		if err != nil {
			return fmt.Errorf("inspect symlink target %s: %w", rel, err)
		}
		if target.IsDir() {
			return fmt.Errorf("tools mining does not support a nested directory alias: %s", rel)
		}
	}
	runErr := run()
	after, scanErr := captureRollingState(root)
	for rel, current := range after {
		prior, existed := before[rel]
		unchanged := existed && prior.mode == current.mode && bytes.Equal(prior.data, current.data)
		if current.mode&os.ModeSymlink == 0 || unchanged {
			continue
		}
		if !strings.HasSuffix(rel, ".md") || !isSessionRollingFile(root, rel, prior.data) {
			scanErr = errors.Join(scanErr, fmt.Errorf("cannot inventory changed symlink: %s", rel))
		}
	}
	err = errors.Join(runErr, scanErr)
	if err == nil {
		for _, rel := range changedRollingFiles(root, before, after) {
			current, exists := after[rel]
			if !exists {
				err = fmt.Errorf("rolling file deleted: %s", rel)
				break
			}
			if current.mode&os.ModeSymlink != 0 {
				err = fmt.Errorf("rolling file is a symlink: %s", rel)
				break
			}
			proposal, repairErr := repairSessionRolling(root, rel, current.data)
			err = repairErr
			if err == nil && !bytes.Equal(proposal, current.data) {
				var path string
				path, err = validateActionPath(root, rel)
				if err == nil {
					err = writeFileAtomic(path, proposal, current.mode)
				}
			}
			if err != nil {
				err = fmt.Errorf("%s: %w", rel, err)
				break
			}
		}
	}
	if err != nil {
		if restoreErr := restoreRollingState(root, before, after); restoreErr != nil || scanErr != nil {
			return fmt.Errorf("%w: %w", errRollingRestore, errors.Join(err, restoreErr))
		}
	}
	return err
}

var rollingEntryDateRE = regexp.MustCompile(`^##[ \t]+(\d{4}-\d{2}-\d{2})([ \t]|$)`)
var rollingSourceRE = regexp.MustCompile(`session:[A-Za-z0-9][A-Za-z0-9._-]*`)

func rollingEntryProvenance(body string) (string, string, []string) {
	var first, last, fence string
	sources := make([]string, 0)
	seen := make(map[string]bool)
	for line := range strings.SplitSeq(body, "\n") {
		if strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if fence != "" {
			closing := strings.TrimRight(trimmed, " \t")
			if len(closing) >= len(fence) && strings.Trim(closing, fence[:1]) == "" {
				fence = ""
			}
			continue
		}
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			length := 0
			for length < len(trimmed) && trimmed[length] == trimmed[0] {
				length++
			}
			fence = trimmed[:length]
			continue
		}
		if match := rollingEntryDateRE.FindStringSubmatch(line); match != nil {
			date := match[1]
			if _, err := time.Parse(time.DateOnly, date); err == nil && !isFutureDate(date, time.Now()) {
				if first == "" || date < first {
					first = date
				}
				if date > last {
					last = date
				}
			}
		}
		for _, source := range rollingSourceRE.FindAllString(line, -1) {
			if !seen[source] {
				sources = append(sources, source)
				seen[source] = true
			}
		}
	}
	return first, last, sources
}

func repairSessionRolling(root, rel string, data []byte) ([]byte, error) {
	span, err := frontmatterSpan(string(data))
	var raw map[string]any
	hasHeader := err == nil
	if errors.Is(err, errNoFrontmatter) && !strings.HasPrefix(string(data), "---") {
		err = nil
		raw = make(map[string]any)
	}
	if err != nil {
		return nil, err
	}
	if hasHeader {
		if err := yaml.Unmarshal(data[span.YAMLStart:span.YAMLEnd], &raw); err != nil {
			return nil, fmt.Errorf("invalid YAML: %w", err)
		}
	}
	if value, exists := raw["rolling"]; exists && value != true {
		return nil, errors.New("rolling marker conflicts with file name")
	}
	for _, field := range []string{"title", "type", "domain", "confidence", "authority"} {
		if value, exists := raw[field]; exists {
			text, ok := value.(string)
			if !ok || strings.TrimSpace(text) == "" {
				return nil, fmt.Errorf("%s must be a nonempty string", field)
			}
		}
	}
	for _, field := range []string{"created", "updated"} {
		if value, exists := raw[field]; exists {
			if _, err := time.Parse(time.DateOnly, stringFromAny(value)); err != nil {
				return nil, fmt.Errorf("%s is not a valid calendar date", field)
			}
		}
	}
	body := string(data[span.BodyStart:])
	first, last, sources := rollingEntryProvenance(body)
	domain := filepath.Base(filepath.Dir(rel))
	if !validDomainsForRoot(root)[domain] {
		domain = "general"
	}
	defaults := []struct {
		key   string
		value any
	}{
		{"title", rollingRepairTitle(rel)},
		{"type", "project"}, {"domain", domain}, {"confidence", "medium"},
		{"authority", "contextual"}, {"rolling", true},
		{"tags", []string{}}, {"related", []string{}}, {"sources", sources},
		{"created", first}, {"updated", last},
	}
	var additions bytes.Buffer
	for _, field := range defaults {
		if _, exists := raw[field.key]; exists {
			continue
		}
		if (field.key == "created" || field.key == "updated") && field.value == "" {
			return nil, fmt.Errorf("cannot recover %s from dated entries", field.key)
		}
		encoded, err := yaml.Marshal(map[string]any{field.key: field.value})
		if err != nil {
			return nil, err
		}
		additions.Write(encoded)
	}
	proposal := []byte("---\n")
	if hasHeader {
		proposal = append([]byte(nil), data[:span.YAMLEnd]...)
	}
	if len(proposal) > 0 && proposal[len(proposal)-1] != '\n' {
		proposal = append(proposal, '\n')
	}
	proposal = append(proposal, additions.Bytes()...)
	if hasHeader {
		proposal = append(proposal, data[span.YAMLEnd:]...)
	} else {
		proposal = append(proposal, []byte("---\n")...)
		proposal = append(proposal, data...)
	}
	if failures := validateArticleContent(root, proposal); len(failures) != 0 {
		return nil, errors.New(strings.Join(failures, "; "))
	}
	return proposal, nil
}

func rollingRepairTitle(rel string) string {
	name := filepath.Base(filepath.Dir(rel)) + "-" + strings.TrimSuffix(filepath.Base(rel), ".md")
	words := strings.Split(name, "-")
	for i, word := range words {
		runes := []rune(word)
		if len(runes) != 0 {
			words[i] = strings.ToUpper(string(runes[:1])) + string(runes[1:])
		}
	}
	return strings.Join(words, " ")
}
