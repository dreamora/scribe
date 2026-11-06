# Rolling-file frontmatter validity: product requirements

Status: confirmed

Implementation: S1 is VERIFIED for its bounded normal tools-mode scenario. The fresh independent gate PASSED after one pass; the initial gate remains historically BLOCKED. S2–S5 are authorized sequentially but remain unimplemented and provisional.

Decision sources: [issue #2](https://github.com/dreamora/scribe/issues/2), the [glossary](domain-model/GLOSSARY.md), the ADRs below, and the design interview in this chat.

Confirmation: The user accepted Q1–Q17 and confirmed the complete design on 2026-11-05. This PRD carries that confirmation forward.

Execution permission: The user authorized local S1 work, its commit, and sequential local implementation and verification of S2–S5 with bounded independent gates. Live KB repair, installation, pushes, and publication remain unauthorized.

## Problem and outcome

KB owners need rolling memory files that pass frontmatter validation immediately after session mining. Issue #2 reports incomplete metadata in new `learnings.md` and `decisions-log.md` files under scribe 0.5.10.

Go must enforce validity before recording a session as processed. Owners must also have an explicit command to repair all existing rolling files in the selected KB. Repair must preserve content and supported provenance, report unresolved files, and coordinate with mining.

The issue's runtime report is user evidence. Local inspection confirmed that Anthropic defaults to tools mode when `session_mine.mode` is unset. Before the S1 fix, tools mode recorded successful model calls without validating their rolling-file output. A fixture child reproduced the missing-metadata defect locally. Live provider reproduction: **not run**.

## Requirements and acceptance

### R1. Go enforces validity during mining

Go must initialize or repair required rolling-file metadata and validate the result before recording session success. Model instructions alone are insufficient. Apply this contract to new and updated rolling files in both mining modes, including normal and large sessions.

Acceptance: A model writes incomplete metadata with recoverable values. Go produces valid frontmatter before adding the processed-session marker. Unrecoverable output follows R8. Untouched invalid files belong to explicit bulk repair and must not block unrelated sessions.

Source and state: **confirmed**, Q2–Q3 and Q6; [ADR 0001](decisions/0001-go-owns-rolling-frontmatter.md).

### R2. Repair selects all rolling memory files

Include live files and archives across the selected KB, including untouched projects. Select files marked `rolling: true` within KB content directories. Also select standard or configured rolling filenames under `projects/` when the marker is missing. Include their yearly archives.

Acceptance: Selection includes `learnings.md`, `decisions-log.md`, configured rolling targets, and their `-archive-YYYY.md` variants. A nonstandard file marked `rolling: true` in a KB content directory also qualifies. Report malformed YAML and explicit `rolling: false` conflicts without changing those files. Do not select unrelated Markdown articles solely because they are in the KB.

Source and state: **confirmed**, Q3 and Q5–Q8. Existing configuration: `meta.rolling_targets`.

### R3. Fill missing metadata without replacing valid values

Repair must preserve body bytes and valid existing metadata values, including contributor data and fields outside the required set. Supply missing fields as follows:

| Field | Required value or source |
| --- | --- |
| `title` | Derive from the containing directory and filename. |
| `type` | `project` |
| `domain` | Use the directory name when it is an allowed domain; otherwise use `general`. |
| `confidence` | `medium` |
| `authority` | `contextual` |
| `rolling` | `true` for a selected file with a missing marker. |
| `tags`, `related` | Empty lists. |
| `created`, `updated`, `sources` | Recover under R4. |

Acceptance: A repairable file receives its missing fields and passes frontmatter validation. Its body bytes and valid pre-existing metadata values remain unchanged. Report invalid existing values, such as `confidence: certain`, for manual correction; leave that file unchanged.

Source and state: **confirmed**, Q3, Q8–Q9, and Q15. The required schema includes `title`, `type`, `created`, `updated`, `domain`, `confidence`, `tags`, `related`, and `sources`.

### R4. Recover missing provenance from entries

Apply [ADR 0002](decisions/0002-recover-rolling-provenance-from-entries.md). Preserve valid existing dates and sources. Recover missing `created` from the earliest valid entry date and missing `updated` from the latest. Entry order does not determine these dates. Ignore date examples inside code blocks.

Recover a missing source list from explicit `session:<id>` references in entries. No explicit session references produces an empty source list. Do not assign the repair date as an unsupported historical date.

Acceptance: Newest-first entries dated 2026-10-01 and 2026-11-01 yield those dates for missing `created` and `updated`, respectively. A date example inside a code block does not affect the range. Unresolved or contradictory dates leave the entire file unchanged and produce an unresolved result.

Source and state: **confirmed**, Q7 and Q10; ADR 0002.

### R5. Provide an explicit rolling-only repair command

Add `--rolling-only` to the existing lint repair command. Require `--fix`. Preserve existing file selectors and `--changed`; apply the rolling-file filter within the selected scope.

These are target interfaces, not currently verified commands:

```sh
scribe lint --fix --rolling-only --dry-run
scribe lint --fix --rolling-only
```

Acceptance: The first command previews repairs; the second applies them. Using `--rolling-only` without `--fix` fails visibly. Neither command performs duplicate cleanup. Ordinary lint and repair retain their existing behavior. CLI help and `README.md` explain scope, preview, unresolved files, and the busy result in R9. Applicable prompts and the KB template must agree with the metadata contract.

Source and state: **confirmed**, Q6, Q12–Q13, and Q17.

### R6. Bulk repair is independent for each file

Validate each proposed file before replacing it atomically. Repair valid candidates even when other files are unresolved. Leave unresolved files unchanged. Report scanned, repaired, unchanged, and unresolved counts, with affected paths and causes.

Acceptance: A batch contains one repairable file and one unresolved file. Repair changes only the repairable file and returns a nonzero exit status. A preview performs the same assessment, reports proposed repairs and unresolved files, and writes no content. Unresolved files also make preview return nonzero. A successful repair preserves body bytes; a repeated repair makes no further content changes.

Source and state: **confirmed**, Q3, Q7, and Q13. Repeated repair checks the requirement to preserve valid files.

### R7. Isolate session writes and checkpoints

Apply [ADR 0003](decisions/0003-serialize-tools-mode-session-mining.md). Tools-mode sessions run serially. Envelope-mode model calls remain parallel. Go serializes each envelope session's apply, validation, and restoration phase. Checkpoints run only between these phases.

Acceptance: Two envelope sessions target the same rolling file. One fails after a write; the other's successful content survives. No checkpoint captures unvalidated output. Normal tools-mode sessions no longer write concurrently, even when configured parallelism exceeds one.

Source and state: **confirmed**, Q11 and Q16; ADR 0003.

### R8. Restore rolling writes after a caught session failure

Apply [ADR 0004](decisions/0004-restore-failed-session-rolling-writes.md). Restore changed rolling files to their pre-session bytes. Remove rolling files created by that failed session. Keep the session pending. Apply restoration to model errors after partial writes and to failures during repair or validation.

Acceptance: After a real file mutation, inject a caught failure. Existing rolling files match their pre-session bytes, new rolling files from that session are absent, and the session has no processed marker. If restoration fails, stop mining and checkpoint commits and report the failure. Process-crash recovery is excluded.

Source and state: **confirmed**, Q2 and Q14; ADR 0004.

### R9. Coordinate repair with mining

Rolling-only repair and its preview must acquire the selected KB's existing sync lock. A busy lock must produce a nonzero status and identify the active operation. Neither command may wait indefinitely or proceed with concurrent edits.

Acceptance: Hold the selected KB's sync lock in a fixture. Both repair and preview fail visibly without changing KB content. With the lock available, each command proceeds according to R5–R6. This does not change ordinary lint repair's coordination behavior.

Source and state: **confirmed**, Q17.

## Decisions and constraints

Use the linked ADRs for ownership, provenance, concurrency, and restoration trade-offs. Use the glossary's terms throughout the implementation and documentation.

Preserve project-based tools-mode locations and domain-based envelope-mode locations. Do not migrate files or change provider defaults to solve this defect. Bulk repair must be an explicit action; mining enforces validity for files it creates or updates.

These requirements define the destination, not an implementation sequence. Use `tracer-bullet` before implementation because mining enforcement and bulk repair are separate behaviors with consequential failure paths. Use `kaizen-small-change` for edits, `jidoka-focused-testing` for verification, and `standard-work-documentation` for contract updates.

## Verification

**Inspected contracts:** the ticket, glossary, four ADRs, mode defaults, tools-mode success handling, and lint repair. The [S1 evidence](evidence/issue-2.md) records executed normal tools-mode checks and three existing regressions. The existing `TestApplyMetaRollingAppend_InitFrontmatterLintValid` checks one new envelope rolling file. It does not establish the new envelope transaction or repair command.

**Required verification matrix:** S1 supplies partial local evidence. Cases outside the named S1 evidence remain unexecuted.

| Boundary | Required evidence |
| --- | --- |
| Mining and processed log | Normal and large tools-mode fixtures cannot record invalid rolling output as processed; cover envelope mode too. |
| Selection and metadata | Cover archives, custom targets, missing markers, explicit conflicts, valid-value preservation, body bytes, and unrecoverable values. |
| Provenance | Cover newest-first dates, code-block examples, missing references, unresolved dates, and contradictory dates. |
| CLI and persistence | Cover preview, selectors, `--changed`, mixed outcomes, counts, nonzero failures, atomic replacement, and exclusion of duplicate cleanup. |
| Concurrency and recovery | Inject failure after mutation; prove restoration and isolation from successful sibling writes and checkpoints. Cover restoration failure and a busy sync lock. |

Use synthetic sessions and temporary KBs. Do not use the live KB or a paid model call to establish these contracts. Run focused regressions, then the repository's required test and vet checks with `sqlite_fts5`; the implementation phase selects the exact commands.

Investigation hints, not normative file ownership: `sync_sessions.go`, `session_mine_envelope.go`, `codex_mine.go`, `wiki_actions.go`, `frontmatter.go`, `lint.go`, `lint_fix.go`, and adjacent tests under `cmd/scribe/`. Documentation consumers include CLI help, `README.md`, session-extraction prompts, and the embedded KB template.

## Exclusions and open decisions

No current requirement is blocked by an unresolved product decision. Selection, defaults, repair, concurrency, and caught-failure behavior are confirmed. Implementation mechanisms and their evidence remain to be established.

Excluded: process-crash recovery, restoration of unrelated article output, file-location migration, generic metadata correction, automatic bulk repair during mining, and changes to ordinary lint cleanup. Editing a live KB, changing schedules, installing a binary, or publishing a release requires separate execution authorization.

## Delivery links and status

Track the defect in [issue #2](https://github.com/dreamora/scribe/issues/2). The [slice plan](issue-2-rolling-frontmatter-plan.md) records verified S1, the historical blocked gate, and the separate passed fresh gate. [S1 evidence](evidence/issue-2.md) retains the JEP-3 correction, successful rejection regressions, exact code identity, and independent review. S1 supplies bounded normal-session coverage of R1–R4 and R7–R8; it does not complete those requirements' later variants. R5–R6, R9, and S2–S5 remain unimplemented. No whole-PRD completion claim is supported.

The initial gate stopped on 2026-11-06 after three consecutive passes produced material findings. A later explicit user request authorized bounded directory-alias rejection, S1 checks, and a fresh gate. The fresh gate passed after one independent pass with technical verdict ready and no supported material findings. S1 is verified within its slice boundary. The user subsequently authorized S2–S5 sequentially; each requires reassessment before implementation and verification before the next slice.
