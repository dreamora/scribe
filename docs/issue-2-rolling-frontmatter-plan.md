# Issue #2: rolling-file validity slice plan

Status: S1–S3 verified; S4 reassessment

Intent: [confirmed PRD](issue-2-rolling-frontmatter-prd.md). Its requirements and acceptance results remain authoritative.

Execution permission: The user authorized local S1 implementation on 2026-11-05, then the JEP-3 correction and fresh gate on 2026-11-06. The user subsequently requested the S1 commit and sequential reassessment, implementation, checks, and bounded independent gates for S2–S5. Tracker writes, live KB repair, installation, pushes, and publication remain unauthorized.

Current slice: S4 reassessment. S1–S3 are VERIFIED within their slice boundaries. Each fresh independent gate PASSED after one pass with technical verdict ready and no supported material findings. The initial three-pass S1 gate remains historically BLOCKED. S4/S5 remain unimplemented and provisional. Finish each slice and its gate before advancing. S1/S2 were committed separately as `335cb44` and `eb48423`.

## Inspection and task boundary

Repository base at inspection: `54f5762c6458c26e504508398731824fdefeb265`. The working tree contains existing tracked and untracked changes in affected code. The base revision alone does not identify that code state. Before implementation, inspect and preserve the current diff and relevant untracked files. Do not revert other work or substitute a clean base without carrying required changes forward.

Planning reused the [glossary](domain-model/GLOSSARY.md), all four PRD-linked ADRs, and the existing flat `docs/*-plan.md` location. No plan for this issue existed.

Source observations are **inspected, not executed**:

| Boundary | Observation and consequence |
| --- | --- |
| Tools-mode dispatch | `mineSessionBatches` starts concurrent model processes when parallelism exceeds one. `mineSessionsSerial` records success after the model returns. S1 must protect both the subprocess result and the processed marker. |
| Failure propagation | Batch helpers return a count and rate-limit flag. `extractPhase` logs mining errors and continues; `SyncCmd.Run` can subsequently enter its final commit phase. Restoration failure therefore needs a path that prevents later commits, not only an error log. |
| Repair policy | Generic lint repair supplies current dates, changes some invalid values, and can strip body whitespace. It cannot be reused unchanged for PRD R3–R4. |
| Persistence and locking | `writeFileAtomic`, `lockPathFor`, `acquireLock`, and `withLock` provide existing filesystem mechanisms. Their existence does not prove the new transaction or repair contracts. |
| Test isolation | `TestMain` isolates home and configuration. `sync_sessions_test.go` warns that mining also invokes ccrider and triage subprocesses. A new assembled check must replace those external inputs with fixtures. |

Code graph lookup was unavailable. Direct source and diff inspection supplied the observations above.

## S1: one tools-mode session produces valid rolling output or restores it

Status: VERIFIED for the named normal tools-mode path and its required safeguards. A1–A6 checks, JEP-3 regressions, and the fresh independent gate pass. The proof and limits are retained in [S1 evidence](evidence/issue-2.md).

### Aim

Outcome: A normal tools-mode session writes a project's `learnings.md` through a real child process. Go completes valid metadata before recording success. A caught failure restores that session's rolling output and leaves the session pending.

Consequential uncertainty: Can the current in-place tools workflow identify a subprocess's rolling-file changes, preserve supported metadata and body bytes, and restore failures before any commit?

PRD scope: [R1](issue-2-rolling-frontmatter-prd.md#r1-go-enforces-validity-during-mining), [R3](issue-2-rolling-frontmatter-prd.md#r3-fill-missing-metadata-without-replacing-valid-values), [R4](issue-2-rolling-frontmatter-prd.md#r4-recover-missing-provenance-from-entries), and the tools-mode portions of [R7](issue-2-rolling-frontmatter-prd.md#r7-isolate-session-writes-and-checkpoints) and [R8](issue-2-rolling-frontmatter-prd.md#r8-restore-rolling-writes-after-a-caught-session-failure).

Artifact: a retained tracer bullet through the tools-mode process and persistence boundaries.

Implemented mechanism: Capture pre-session Markdown state, run tools-mode sessions serially, assess changed rolling files after the child exits, and record success only after validation. Restore captured rolling state on a caught failure. Incomplete post-child inventory or failed restoration stops the current sync invocation before later mining and commits.

Historical counterexample: a pre-existing `projects/acme -> ../raw/acme` directory alias hid changes to an unprotected backing file. JEP-3 was an S1 defect, not a later-slice exclusion. The authorized correction now rejects nested directory aliases before child launch while retaining selected KB-root alias support. It does not add traversal or referent restoration.

Alternative: Isolate each tools session's output and merge it after validation. [ADR 0003](decisions/0003-serialize-tools-mode-session-mining.md) chose serial tools sessions to avoid that larger change. If the proposed mechanism cannot satisfy the agreed contract, record the failed assumption and revisit the design before adopting isolation.

### Boundary and immediate prerequisites

Depends on: no earlier implementation slice. The confirmed PRD and ADRs provide the contract.

The immediate prerequisite is a hermetic test harness. It must supply a temporary KB and lock directory, synthetic session data, a local Git repository, and controlled subprocess inputs. Establish that these inputs cannot reach the user's ccrider state, authenticated model, live KB, or remote Git service. Harness preparation alone does not complete S1.

Path: existing session dispatcher → real child process that writes rolling content → Go metadata assessment and validation → persisted processed marker → eligible checkpoint. The failure path returns through restoration and the sync commit decision.

Start with the standard project `learnings.md` path and normal-session prompt. Use two entries dated 2026-10-01 and 2026-11-01, newest first, with an explicit synthetic session reference. Include a variant with valid existing metadata and a new body entry.

Narrowing the first proof does not permit success for unsupported or invalid output. Detect such output, restore the session's rolling changes, and keep it pending. Do not enable a broader claim until its checks pass.

Keep these protections in S1:

1. Preserve body bytes across Go's metadata repair and preserve valid metadata values under R3.
2. Recover missing provenance under R4; do not invent historical dates.
3. Serialize tools-mode child writes, even when configured parallelism exceeds one.
4. Restore changed rolling files and remove newly created rolling files after caught failure; do not record success.
5. Prevent later mining and commits when restoration fails. Do not roll back unrelated article output or claim crash recovery.

Preparatory refactoring may expose error propagation or file-operation seams needed by these protections. Keep it local and checked. A helper-only change is not the completed slice.

### Proof

The model substitute must be an executable child reached through the production process adapter, such as `realRunClaude` with a fixture executable on the test PATH. The substitute replaces model reasoning and service access. It must not replace filesystem writes, validation, processed-log persistence, restoration, or Git observations with fake success responses.

Use real temporary files and a temporary Git repository. Disable external reindexing and supply ccrider and triage fixtures when exercising the sync entry point. Existing `stubClaude` tests can supplement this proof; they cannot establish the subprocess boundary by themselves.

| Acceptance case | Observable expected result |
| --- | --- |
| A1: recoverable new file | The child creates incomplete frontmatter and dated entries. At a pre-validation barrier, the processed marker is absent. Release the barrier; real validation passes before the marker appears. |
| A2: existing valid values | The child adds an entry. Go's repair preserves that resulting body byte-for-byte and retains valid existing metadata, including contributor and source values. |
| A3: caught failure after writes | Observe changed existing bytes and a newly created rolling file before releasing the child to fail. After handling, original rolling bytes are restored, the new file is absent, and the session stays pending. Repeat with an unrecoverable metadata result. |
| A4: competing tools sessions | Request parallelism greater than one. Two fixture sessions target the same rolling file. Child writes do not overlap, and both successful entries survive. |
| A5: checkpoint visibility | A real local Git checkpoint contains only validated output and matching success markers. Failed-session rolling changes never appear in a subsequent checkpoint. |
| A6: restoration fails | Inject a restoration failure after a demonstrated mutation. Through the sync entry point, verify that later mining and checkpoint or final commit paths do not run. Report the restoration failure. |

A3 must observe a real mutation before failure; an unchanged final fixture alone is insufficient. A6 must exercise the orchestration that currently continues after errors. A fake commit callback alone cannot establish that the final sync commit is suppressed.

Before the implementation fix, use the new discriminating check to expose missing frontmatter enforcement. Label a setup or compilation failure separately from a reproduced product failure.

The executed focused command selected this prefix:

```sh
go test -tags sqlite_fts5 ./cmd/scribe -run '^TestIssue2ToolsRolling' -count=1 -v
```

The `TestIssue2ToolsRolling` tests now cover A1–A6. Exact commands, execution counts, observations, and target hashes are in the linked evidence record. No relevant test skipped.

Retain existing rate-limit, budget, processed-log, and rolling initialization regressions. Update the relevant tools-mode prompt or template guidance and user documentation when S1 changes their supported behavior. Run focused checks first, then `make check` on the completed slice. Inspect the resulting files, logs, and local Git state before selecting another slice.

The checks ran during the authorized S1 implementation. No live model call, real KB repair, install, push, or external publication ran.

### Decision rule

| Decision | Required observation |
| --- | --- |
| Continue | A1–A6 pass through their real boundaries, relevant regressions pass, and the assembled output satisfies the PRD. Retain the path, link its evidence, and reassess S2. |
| Revise | Metadata repair changes body bytes, omits a changed rolling file, loses provenance, or cannot propagate a restoration failure. Correct the mechanism and repeat the affected proof before dependent work. |
| Stop | Required isolation is unavailable, another agent's work cannot be preserved, or satisfying the contract requires a new product decision. Record the gap; do not mark S1 verified or start dependent implementation. |

Implementation used `kaizen-small-change` and `jidoka-focused-testing`. The initial gate stopped after three material-finding passes. A later explicit user request authorized this bounded correction and a fresh gate. Local checks and the fresh gate passed. The historical gate is not relabeled as passed.

## S2: complete protected tools-mode rolling-file variants

Status: VERIFIED; focused/race/full local checks passed; fresh independent gate PASSED after one pass with technical verdict ready and no supported material findings. Reassessed against verified S1 and the current working source. Baseline: `335cb44` plus preserved unrelated dirty work, captured under `/private/tmp/scribe-issue2-s2-before-20261106` before edits.

### Aim and boundary

Outcome: Normal and large tools-mode sessions complete only after Go repairs and validates every changed selected rolling file. Caught failures restore all session rolling changes and leave the session pending. Failed restoration stops later mining, checkpoints, and the final commit.

Question: Can S1's local transaction support the full confirmed rolling selection and metadata policy without touching unrelated files or weakening recovery? Retain S1's snapshot and restoration mechanism; extend its local selection/assessment rules instead of adding an isolated-output merge path. Revise that choice if the assembled variants expose incomplete rollback or incorrect selection.

Depends on S1's verified real-child boundary, serial tools execution, processed-marker ordering, caught-error restoration, alias rejection, and fatal-stop propagation. Reuse the fixture subprocess, temporary KB, synthetic sessions, and real local Git harness. Preserve unrelated dirty work, stable selected KB-root aliases, operation labels, rate-limit and daily-budget stops. No live KB or authenticated model calls.

Scope: complete tools-mode R1, R3–R4, R7–R8 and the R2 recognition used by mining. Select changed `rolling: true` Markdown files in KB content directories, standard and configured project targets with a missing marker, and yearly archives. Derive missing titles from the full directory and filename, preserving Unicode. Use allowed domains or `general`. Preserve body bytes and valid existing fields, including custom/contributor fields. Reject malformed metadata, explicit marker conflicts, invalid existing values, and unresolved or contradictory dates. Missing explicit session references recover to an empty source list. Untouched invalid regular files do not block or change.

Exclude envelope apply/locking, lint CLI preview/apply, and all-KB bulk repair. These remain S3–S5. S1's supported alias containment remains in force; this slice does not add alias traversal or referent restoration.

### Build and proof

First extend shared selection and metadata assessment through the normal tools path. Inspect a complete successful fixture output, its body and fields, and its processed marker before extending the independently observable large-session path. Then route both tools prompts through protection and verify large-session fatal propagation through actual sync orchestration.

Normal-path observation, 2026-11-06: `TestIssue2ToolsVariantsNormal` first failed because a marked nonstandard file received validation without repair. After extending the local assessment, the real child produced nine valid selected files in one normal session: both standard targets, a configured target, their yearly archives, a marked nonstandard content file, a Unicode configured target, and an archive with no frontmatter. Inspected output recovered the expected dates and source, excluded fenced examples, derived full titles, preserved body bytes, and recorded the processed marker. An ordinary child-created article and an untouched invalid regular file stayed unchanged. Decision: retain the shared assessment and continue to large-session integration. Raw red/green logs are in the S2 baseline scratch root.

Completed observation: B1–B6 pass through normal and large tools paths. Final focused checks passed 58 events, targeted race checks passed 61 events, with zero failures/skips. `make check` passed tests and vet. The fresh gate PASSED after one independent pass with technical verdict ready and 33 independently rerun pass events. The [S2 evidence](evidence/issue-2.md#s2-remaining-tools-mode-variants) records exact commands, origins, preserved baseline and seven frozen target hashes. Decision: retain verified S2 and reassess S3.

| Case | Required observation |
| --- | --- |
| B1: normal variants | A real child writes decisions, configured targets, yearly archives, and nonstandard marked files. Missing fields/markers recover correctly; full filename-derived titles, allowed-domain defaults, date range, sources, body bytes, and processed marker are observed. Standard filenames remain selected when custom targets are configured. |
| B2: metadata preservation and provenance | Valid metadata, contributor/custom fields, and body bytes survive. Code examples do not supply dates or references; no references yields `sources: []`. Unresolved/contradictory dates, malformed YAML, explicit false markers, and invalid existing values fail without manual-value replacement. |
| B3: complete rollback | Observe actual child mutation before caught failure. Restore pre-existing selected variants, remove newly created variants, leave the marker absent, and preserve unrelated regular files. A later invalid file must also undo an earlier repair. |
| B4: large integration | The production large prompt reaches a real child. Both successful completion and caught failures obey the same validation, preservation, and pending-marker contract. |
| B5: checkpoint and fatal stop | Real Git checkpoints exclude invalid large-session output. Inject a restoration failure after an observed large-session mutation through `SyncCmd.Run`; no later child, staging, checkpoint, or final commit runs. |
| B6: retained safeguards | S1 alias/root controls and untouched-invalid-file isolation still pass. Retain rate-limit, daily-budget, processed-log, and rolling initialization regressions; tools sessions remain serial. |

Run discriminating checks before the relevant product change. After the normal result, record the observation and continue/revise decision here. On completed S2 run focused S1/S2 checks, targeted race checks and regressions, then `make check` with `sqlite_fts5`. Update affected README, both tools prompts, and embedded guidance. Freeze the tested target for the parent's fresh independent gate; no nested gate.

### Decision rule

Continue: B1–B6 pass at the named real boundaries, focused/race and required checks pass, and a fresh independent gate passes. Retain S2, record exact evidence, then reassess S3.

Revise: a selected variant escapes assessment, repair changes body/valid metadata, recovery leaves session output, or fatal errors allow later writes/commits. Correct within S2 and repeat affected proof before dependent work.

Stop: the contract needs a new product decision, safe local isolation is unavailable, or unrelated work cannot be preserved. Report the gap. Do not claim S2 verified or begin S3.

## S3: isolate envelope apply while generation stays parallel

Status: VERIFIED; focused/race/full local checks passed; fresh independent gate PASSED after one pass with technical verdict ready and no supported findings/testing gaps. Pre-S3 dirty baseline: `335cb44` plus preserved S2/unrelated working edits, captured with file origins/hashes under `/private/tmp/scribe-issue2-s3-before-20261106`. HEAD later advanced to `eb48423` for the isolated verified S2 commit without changing those working bytes.

### Aim and boundary

Outcome: ccrider envelope model calls remain parallel, while one Go coordinator applies, validates or restores, records the correct session marker, and checkpoints between sessions. Codex mining uses the same protected apply boundary with its separate processed log. A failed session cannot erase a successful sibling's rolling content. Unrestorable output stops later mining and final commits.

Question: Can the existing entity executor and S2 assessment share one envelope-session boundary without the generic sanitizer/date stamps replacing rolling evidence or checkpoints observing partial output? Retain the current provider boundary and ordinary-article collision/normalization behavior. Add a session-scoped rolling policy so raw rolling metadata reaches S2 assessment. Split parallel generation from serial apply rather than placing a lock around each full model call. The latter would lose required generation parallelism.

Depends on S1's observed restoration/fatal-commit protections and S2's verified selection, metadata, alias and body-preservation rules. Preserve normal/large prompt and cost labels, existing provider selection, empty-transcript/low-score skip behavior, ordinary article normalization and create-collision protection. Use fixture providers only for model responses; SQLite transcripts, generated envelopes, filesystem mutations, validation, both processed logs, restoration and local Git remain real. No authenticated providers or live KBs.

Scope: complete mining R1/R7/R8 and retain R2–R4 across both ccrider envelope sizes and the Codex caller. Archives, configured targets, marked nonstandard outputs, all metadata/provenance rejection cases, and existing root-alias containment remain included. Restoration still covers rolling output, not unrelated articles. The fatal guarantee remains one sync invocation; no crash recovery or persistent quarantine is added. S4/S5 CLI preview/apply remain excluded.

### Build and proof

First prove one ccrider session from a fixture SQLite transcript through generation, real rolling meta append, shared repair/validation and persisted success marker. Inspect its complete body/dates/sources before changing batch scheduling or adding Codex variants. Then retain that boundary while splitting primary generation from serial application and adding other callers.

Representative observation, 2026-11-06: `TestIssue2EnvelopeRepresentative` first failed because new rolling metadata used the current date and empty sources. After the session-scoped apply policy, the real SQLite → structured response → filesystem → validation → processed-log path passed. Inspected output recovered 2026-10-01/2026-11-01 and its explicit session reference, preserved the appended body, and recorded only the actual session, ignoring a forged model marker. The shared transaction now supports a checked completion callback after validation, so marker-write failure can restore rolling output instead of claiming success. Decision: retain this path and continue to batch isolation and other callers. Raw representative red/green logs are in the pre-S3 scratch root.

Completed observation: C1–C6 pass for normal/large ccrider and Codex envelope callers. Final focused checks passed 112 events, targeted race checks passed 119 events, zero failures/skips; `make check` passed tests and vet. The same full-sync fatal cases fail on the verified pre-S3 source origins. The fresh gate PASSED in one pass with 54 independently rerun passing outcomes and an independently reproduced representative baseline failure. [S3 evidence](evidence/issue-2.md#s3-protected-envelope-application) records origins, exact commands and limits. Decision: retain verified S3 and reassess S4.

| Case | Required observation |
| --- | --- |
| C1: representative envelope | A real fixture transcript yields a structured rolling append. Result has supported earliest/latest dates and sources, preserved body bytes and complete metadata before its correct processed marker. No model-supplied extra session can be marked. |
| C2: variants and provenance | Both ccrider prompt families and Codex use S2's metadata/selection policy for rolling meta and content actions. Preserve valid/custom fields, reject malformed/false/invalid/unrecoverable values, avoid invented dates and retain ordinary-article normalization. |
| C3: parallel generation, isolated apply | At least two provider generations rendezvous concurrently. After an actual failed-session mutation to a shared rolling file, another session cannot apply or checkpoint until failure restoration completes. Its successful content and marker survive; failed content/marker do not. |
| C4: checkpoint boundary | Inspect a real local Git checkpoint while a sibling generation/apply is held at a deterministic barrier. No unvalidated bytes or failed marker appear. Generation may overlap that safe checkpoint; apply may not. |
| C5: caught/fatal failures | Observe real mutations before injected apply/validation failure; restore original rolling bytes/remove new files and leave the appropriate log pending. Through actual sync orchestration, failed restoration stops later ccrider/Codex mining, staging/checkpoints and final commits. |
| C6: caller compatibility | Codex fixtures cover its actual rollout/scoring driver and separate `_codex_sessions_log.json`; ccrider covers normal/large dispatch. Retain empty transcript, no-action, low-score, rate/budget and collision regressions without live calls. |

Run discriminating checks before each relevant fix. Record the first complete observed result before expanding. On the final S3 target run focused S1/S2/S3 checks, targeted race/compatibility checks, then required `make check`. Synchronize affected envelope prompts, README and template claims. Freeze source for the parent's fresh independent gate; do not start a nested gate.

### Decision rule

Continue: C1–C6 pass at their named real boundaries, focused/race/full checks pass and a fresh independent gate passes. Retain verified S3, then reassess S4.

Revise: generic metadata clamps replace rolling evidence, apply/restore/checkpoint phases overlap, a failed session undoes sibling output, or a caller records success/continues commits after an unresolved failure. Correct within S3 and repeat the affected proof.

Stop: a new product decision is necessary, supported caller behavior cannot be preserved, or required mutation/isolation evidence cannot be obtained safely. Keep the gap explicit and do not start S4.

## Provisional continuation

These are outcomes and dependency edges, not implementation tickets. Refine only the next selected slice after inspecting its predecessor's evidence. Execute serially; the slices share state and code.

| Slice | Complete behavior and proof to develop | Depends on and required behavior | PRD coverage |
| --- | --- | --- | --- |
| S2 | VERIFIED under the contract above; fresh gate PASSED in one pass. | S1 supplies subprocess change assessment, valid completion, restoration, and fatal-stop behavior. | Tools-mode R1, R3–R4, R7–R8 and R2 mining recognition. |
| S3 | VERIFIED under C1–C6; fresh gate PASSED in one pass. | S2 supplies selection/metadata policy; S1 supplies caught-failure recovery. The single apply coordinator supplies envelope isolation while generation stays parallel. | Mining R1, R7–R8 and retained R2–R4. |
| S4 | Preview rolling-only repair across a selected KB, including untouched projects and archives. Exercise the real CLI, selectors, sync lock, proposed counts, nonzero unresolved results, and a no-content-write assertion. | S2 supplies selection and metadata assessment. S3 is scheduled first but is not a behavioral prerequisite for read-only preview. | R2–R4 across the full selection; preview portion of R5–R6; preview R9. |
| S5 | Apply rolling-only repair through the same CLI and selection. Validate and replace each file atomically; prove mixed outcomes, preservation, repeated-run stability, counts, nonzero unresolved results, and a busy lock. | S4 supplies a verified CLI assessment and locking path. This slice adds actual persistence and must prove it separately. | Complete R5–R6 and apply R9; retain full-selection R2–R4. |

Preview and apply are separate independently usable behaviors. While only preview is implemented, an apply request must fail explicitly instead of falling through to legacy repair or duplicate cleanup. This is an intermediate containment rule, not a change to the final PRD interface.

## Requirement accounting

| Requirement | Current or provisional delivery | Verified evidence |
| --- | --- | --- |
| R1 | S1 representative tools path → S2 remaining tools variants → S3 envelope callers | S1–S3 mining callers verified |
| R2 | S1 bounded recognition → S2 full mining recognition → S4–S5 all-KB selection | S2 mining selection checks pass; all-KB selection remains later work |
| R3 | S1 representative preservation and defaults → S2 remaining cases → S3–S5 reuse with boundary checks | S1–S3 mining preservation/defaults verified; bulk assessment remains later work |
| R4 | S1 representative recovery → S2 remaining cases → S3–S5 reuse with boundary checks | S1–S3 mining provenance verified; bulk assessment remains later work |
| R5 | S4 preview CLI → S5 apply CLI; update help, README, prompts, and template with affected slices | None |
| R6 | S4 proposed outcomes → S5 actual per-file persistence and mixed outcomes | None |
| R7 | S1 tools serialization and checkpoint safety → S2 large-session coverage → S3 envelope isolation | S1–S3 serialization/isolation and real checkpoints verified |
| R8 | S1 caught-failure restoration and fatal propagation → S2 remaining tools paths → S3 envelope paths | S1–S3 actual-mutation recovery and full-sync fatal stops verified |
| R9 | S4 preview lock → S5 apply lock | None |

No dependency cycle exists in the proposed order: S1 → S2 → S3, S2 → S4, and S4 → S5. S4's placement after S3 is a sequence choice, not an invented blocking dependency. No requirement is considered complete merely because a shared helper exists.

## Evidence and handoff state

| Procedure | State | What it establishes |
| --- | --- | --- |
| Read PRD, glossary, ADRs, current call paths, test isolation, and relevant diff | Inspected, not executed | The plan addresses the confirmed scope and known integration boundaries. |
| Check requirement accounting, dependency order, links, and record structure | Passed during planning; rechecked after the S1 documentation updates | Document consistency only; no software result. |
| Execute S1 focused checks, focused race checks, and required `make check` | Passed after the authorized JEP-3 correction; see S1 evidence | Named local behavior and alias rejection on the unchanged reviewed target. |
| Initial independent gate, passes 1–3 | Historically BLOCKED; technical pass-3 verdict ready with fixes | That gate stopped at its required limit. Its result is not relabeled as passed. |
| Newly authorized fresh independent gate | PASSED after one pass; technical verdict ready | No supported material findings remain in the seven-file S1 scope; two extra real-child reviewer probes also passed. |
| S2 focused, targeted race and `make check` | PASSED; 58 focused and 61 race events, zero failures/skips | Local normal/large tools variants, restoration and checkpoint boundaries. |
| S2 fresh independent gate | PASSED in one pass; technical verdict ready | Seven hashes/origins verified; 33 independent pass events, no supported material findings. |
| S3 focused, targeted race and `make check` | PASSED; 112 focused and 119 race events, zero failures/skips | Local normal/large/Codex envelope validity, checked markers, isolation and fatal propagation. |
| S3 fresh independent gate | PASSED in one pass; technical verdict ready | 13 hashes/12 origins verified; 54 independent passes and one expected representative baseline failure. |

The [evidence record](evidence/issue-2.md) retains S1's initial stopped gate and later passed fresh gate, plus S2/S3's passed checks/gates and exact targets. S4/S5 remain unimplemented at this handoff.

Current decision: retain committed S1/S2 and verified S3, then reassess S4. Complete each authorized slice and its gate before starting the next. Verification is local and bounded; it does not establish deployed behavior or whole-PRD completion.

After each verified slice, update this record's status and the PRD's delivery progress without changing accepted requirements. Reassess the provisional continuation before starting the next behavior.
