# Issue #2: rolling-file validity slice plan

Status: verified for S1 only

Intent: [confirmed PRD](issue-2-rolling-frontmatter-prd.md). Its requirements and acceptance results remain authoritative.

Execution permission: The user authorized local S1 implementation on 2026-11-05, then the JEP-3 correction and fresh gate on 2026-11-06. The user subsequently requested the S1 commit and sequential reassessment, implementation, checks, and bounded independent gates for S2–S5. Tracker writes, live KB repair, installation, pushes, and publication remain unauthorized.

Current slice: S1 is VERIFIED for the bounded normal tools-mode scenario. The fresh independent gate PASSED after one pass with technical verdict ready and no supported material findings. The initial three-pass gate remains historically BLOCKED. S2–S5 are authorized sequentially but remain unimplemented and provisional; reassess each after its predecessor's evidence.

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

## Provisional continuation

These are outcomes and dependency edges, not implementation tickets. Refine only the next selected slice after inspecting its predecessor's evidence. Execute serially; the slices share state and code.

| Slice | Complete behavior and proof to develop | Depends on and required behavior | PRD coverage |
| --- | --- | --- | --- |
| S2 | Extend protected tools-mode mining to large sessions and the remaining rolling-file variants. Prove defaults, date recovery, invalid-value rejection, custom targets, archives, and untouched-file isolation through that workflow. | S1 supplies subprocess change assessment, valid completion, restoration, and fatal-stop behavior. | Complete tools-mode R1, R3–R4, R7–R8; establish reusable R2 recognition for mining. |
| S3 | Apply the same validity contract to envelope mining. Keep model calls parallel while serializing apply, validation, restoration, and checkpoints. Prove that one failed session cannot undo a successful sibling. Check the Codex mining caller's separate processed log. | S2 supplies the complete selection and metadata policy; S1 supplies proven caught-failure restoration and commit suppression. Envelope locking still needs its own proof. | Complete mining R1, R7–R8; preserve R3–R4 across envelope writers. |
| S4 | Preview rolling-only repair across a selected KB, including untouched projects and archives. Exercise the real CLI, selectors, sync lock, proposed counts, nonzero unresolved results, and a no-content-write assertion. | S2 supplies selection and metadata assessment. S3 is scheduled first but is not a behavioral prerequisite for read-only preview. | R2–R4 across the full selection; preview portion of R5–R6; preview R9. |
| S5 | Apply rolling-only repair through the same CLI and selection. Validate and replace each file atomically; prove mixed outcomes, preservation, repeated-run stability, counts, nonzero unresolved results, and a busy lock. | S4 supplies a verified CLI assessment and locking path. This slice adds actual persistence and must prove it separately. | Complete R5–R6 and apply R9; retain full-selection R2–R4. |

Preview and apply are separate independently usable behaviors. While only preview is implemented, an apply request must fail explicitly instead of falling through to legacy repair or duplicate cleanup. This is an intermediate containment rule, not a change to the final PRD interface.

## Requirement accounting

| Requirement | Current or provisional delivery | Verified evidence |
| --- | --- | --- |
| R1 | S1 representative tools path → S2 remaining tools variants → S3 envelope callers | S1 bounded path verified; named cases, JEP-3 rejection, and fresh gate pass; remaining variants unstarted |
| R2 | S1 bounded recognition → S2 full mining recognition → S4–S5 all-KB selection | S1 now rejects unsupported directory aliases before execution; broader selection remains later work |
| R3 | S1 representative preservation and defaults → S2 remaining cases → S3–S5 reuse with boundary checks | S1 local defaults, valid values, and body-byte proof |
| R4 | S1 representative recovery → S2 remaining cases → S3–S5 reuse with boundary checks | S1 local date/source recovery and code-example rejection |
| R5 | S4 preview CLI → S5 apply CLI; update help, README, prompts, and template with affected slices | None |
| R6 | S4 proposed outcomes → S5 actual per-file persistence and mixed outcomes | None |
| R7 | S1 tools serialization and checkpoint safety → S2 large-session coverage → S3 envelope isolation | S1 local serialization and real checkpoint proof |
| R8 | S1 caught-failure restoration and fatal propagation → S2 remaining tools paths → S3 envelope paths | Named rollback cases pass; JEP-3 input is rejected before mutation, with backing bytes preserved |
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

The [S1 evidence record](evidence/issue-2.md) retains both the initial stopped gate and the later authorized JEP-3 correction. It identifies current code, red/green results, successful checks, the passed fresh gate, and its audited independence. No later slice ran.

Current decision: commit the verified S1 path, then reassess S2 using its evidence. Complete each authorized slice and its gate before starting the next. Verification is local and bounded; it does not establish deployed behavior or whole-PRD completion.

After each verified slice, update this record's status and the PRD's delivery progress without changing accepted requirements. Reassess the provisional continuation before starting the next behavior.
