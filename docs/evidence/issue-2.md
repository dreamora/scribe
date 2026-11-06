# Issue #2: S1 tools-mode rolling files

Date: 2026-11-06
Verdict: S1 VERIFIED for its bounded normal tools-mode scenario. Fresh independent gate PASSED after one pass. The initial gate remains historically BLOCKED.
Confidence: high

Evidence applies to the named local checks and reviewer probes. The initial gate's directory-alias counterexample is retained below. The authorized correction passes matching regressions and the fresh independent gate. No deployed behavior or whole-PRD completion is claimed.

## Tried

Implement only S1 from the [slice plan](../issue-2-rolling-frontmatter-plan.md). A fixture executable replaces the model service while the production process adapter, filesystem writes, validation, processed log, and local Git operations remain real.

The first regression ran before production edits. It reproduced a successful mining result with all nine required fields missing. The implementation now serializes tools-mode sessions and wraps normal-session rolling writes in capture, repair, validation, and restoration.

## Findings

- [`session_rolling.go:174`](../../cmd/scribe/session_rolling.go#L174) owns the normal tools transaction. Inventoried project `learnings.md` files receive missing metadata from supported evidence.
- [`session_rolling.go:190`](../../cmd/scribe/session_rolling.go#L190) rejects pre-existing nested directory aliases before child launch. Markdown symlinks and uninspectable alias targets also fail preflight. The selected KB root is outside that nested inventory, so stable root aliases remain supported.
- [`session_rolling.go:146`](../../cmd/scribe/session_rolling.go#L146) rejects symlink ancestors below the selected KB root before restoration writes or removals. Stable configured KB-root aliases remain supported.
- [`session_rolling.go:252`](../../cmd/scribe/session_rolling.go#L252) excludes fenced and indented examples from entry provenance. Both delimiter-length and indented-closing cases are covered.
- [`validate.go:167`](../../cmd/scribe/validate.go#L167) validates proposed content without first writing it to the article path.
- [`sync_sessions.go:503`](../../cmd/scribe/sync_sessions.go#L503) wraps the normal tools subprocess; success marking follows validation. Normal and large cost labels remain distinct, and rate-limit and daily-budget stops remain supported.
- [`sync.go:159`](../../cmd/scribe/sync.go#L159) prevents the final sync commit after fatal restoration failure. The failure also prevents later mining.

Post-child inventory errors cannot establish complete restoration. They produce a fatal stop even when known files were restored. New or replaced rolling symlinks are rejected; restoration does not write through their targets.

**Historical JEP-3**, initial reviewer ID `pass-3/JEP-1`: the reviewed preflight rejected Markdown-file aliases but permitted a pre-existing nested directory alias. With `projects/acme -> ../raw/acme`, the child wrote `projects/acme/learnings.md` while the captured link text remained unchanged. The backing file under `raw/` was absent from snapshots. Validation and caught-failure restoration missed the write. This violated S1's contract, not a deferred S2 variation. The user-authorized correction rejects this alias before execution and passed the fresh independent gate.

## Check

All commands used `GOCACHE=/private/tmp/scribe-issue2-go-cache` because the sandbox could not write the default Go cache.

```sh
go test -tags sqlite_fts5 ./cmd/scribe -run '^TestIssue2ToolsRolling' -count=1 -json
go test -race -tags sqlite_fts5 ./cmd/scribe -run '^TestIssue2ToolsRolling|^TestMineSessionBatches_StopsAfterRateLimit$|^TestRecordSessionProcessed$|^TestApplyMetaRollingAppend_InitFrontmatterLintValid$' -count=1 -json
make check
```

The latest focused run executed fourteen top-level tests and eleven subtests: 25 pass events, zero failures, zero skips. The race run added three existing regressions: 28 pass events, zero failures, zero skips.

The fresh independent reviewer inspected matching raw validation and ran two additional real-child scratch probes. Both passed and left their sessions pending: rollback after a later file fails following an earlier repair, and rejection of contradictory recovered dates.

Reviewer command, executed in `/private/tmp/scribe-s1-pass1-probes-u0276452`:

```sh
go test -tags sqlite_fts5 ./cmd/scribe -run '^TestS1Pass1' -count=1 -v
```

No software rerun was needed for the final documentation update because product code and verification inputs remained unchanged.

The prior post-JEP-2 target passed 22 focused and 25 race events on 2026-11-05, but the later pass-3 probe failed. The initial gate then stopped without further corrections. On 2026-11-06 the user explicitly authorized the JEP-3 fix, matching regressions, fresh S1 checks, and a fresh bounded independent gate. The latest results and hashes below belong to that authorized correction.

| Plan case | Executed proof |
| --- | --- |
| A1 | A real child creates incomplete metadata and waits. The parent observes invalid content and no processed marker before releasing it. Validation then passes and the marker appears. |
| A2 | A child appends to a curated file. Existing fields and resulting body bytes remain unchanged by metadata repair. |
| A3 | The parent observes changed existing bytes and a new rolling file before releasing a child error. Restoration recovers original bytes, removes the new file, and leaves the session pending. Separate cases cover a successful child with invalid confidence and a rate limit. |
| A4 | Two child processes target one file with configured parallelism three. Their write intervals do not overlap and both entries survive. |
| A5 | A real temporary Git checkpoint contains validated output and its success marker. Failed output and failed-session markers are absent. |
| A6 | A real filesystem mutation makes restoration impossible. Through `SyncCmd.Run`, later sessions do not launch, HEAD stays unchanged, and no partial output is staged. |

Additional checks cover incomplete post-child inventory and new or replaced rolling symlinks. The provenance regression first failed with an indented closing-looking fence; it passes after the correction.

JEP-1 adds successful-child and child-error scenarios for an unchanged pre-existing rolling symlink. Both tests failed before the correction because the child ran. Both now prove that preflight prevents child execution, preserves backing bytes and link text, and leaves the session pending. The regular-file success check also retains an untouched invalid regular file without blocking or modifying it.

JEP-2 adds a full-sync filesystem-fault regression. After observing the child's actual mutation, the test harness replaces the parent directory with an in-KB symlink. Before the fix, rollback overwrote the untouched alias destination. After the fix, unrelated bytes remain unchanged, the session remains pending, and no later child, staging, or commit occurs. The harness injects this fault; the test does not establish that restricted model tools can create the link.

A separate real-filesystem check proves that rollback removal cannot delete an untouched file through such an ancestor. It failed before the fix and passes afterward. A positive control demonstrates caught-failure restoration through a stable configured KB-root alias; the first guard rejected that supported root and was corrected to check only ancestors below it.

The JEP-3 regression covers both a successful child and a child that would exit 7. Both failed before the new correction: the child ran, backing bytes changed, and the successful path recorded a processed marker. Both now pass because preflight rejects the directory alias before child launch. The backing bytes and original link stay unchanged, and the session stays pending. Existing stable-root-alias and untouched-invalid-regular-file positive controls also pass.

## Observed output

Before the implementation, `TestIssue2ToolsRollingSuccess` failed with:

```text
rolling output fails validation: [missing required fields: confidence, created, domain, related, sources, tags, title, type, updated]
```

Latest focused run after authorized JEP-3 correction: **passed**, package elapsed 5.180 seconds. Latest race run: **passed**, package elapsed 5.142 seconds.

Final `make check`: **passed**. Its observed output was:

```text
go test -tags "sqlite_fts5"  -count=1 -timeout 20m ./...
ok  github.com/oliver-kriska/scribe/cmd/scribe  34.876s
go vet -tags "sqlite_fts5" ./...
```

The first full-suite attempt was **not evaluable** because the sandbox rejected a local `httptest` loopback listener. The final run used approved local-listener access. This was an environment failure, not a product failure or a reason to skip tests.

### Historical pass-3 directory-alias probe

Reviewer procedure: `TestPass3ExistingDirectoryAlias` ran a real fixture child through the current tools-mode path. Both assertion subcases **failed**. The reviewer verified copied-current Go hashes in the probe tree before execution.

| Subcase | Observed wrong result |
| --- | --- |
| `success` | `mined=1`, `fatal=nil`, `processed=true`; backing bytes were invalid after the write. |
| `caught-failure` | The child exited 7; `mined=0`, `fatal=nil`; overwritten backing bytes survived without restoration. |

The probe is preserved at `/private/tmp/scribe-issue2-pass3-probe/cmd/scribe/pass3_probe_test.go`. No correction was applied inside that stopped gate. The later authorized regression reproduced both wrong outcomes before the new source change; its red log is `/private/tmp/scribe-issue2-jep3-before-20261106/jep3-red.log`.

No live model call, authenticated provider, real ccrider state, live KB repair, installation, push, or publication ran.

### Reviewed code identity

Base: `54f5762c6458c26e504508398731824fdefeb265`, with pre-existing dirty files preserved. A temporary snapshot retained the original affected files, tracked patch, and source hashes before implementation. Comparing the original source manifest found no changes to pre-existing source files outside the selected S1 files.

| Final file | SHA-256 |
| --- | --- |
| `cmd/scribe/session_rolling.go` | `caec9fdca264fe4d6ac30bd1f4dc9a521fc6b433b37253a0585c8aadb3b00f2d` |
| `cmd/scribe/issue2_tools_rolling_test.go` | `1353a24c63e5208beeabd6a2cefca76a5452c3a4c544027a6296d71ed366ccef` |
| `cmd/scribe/validate.go` | `defeadda0644583d4763fb522e6d366412a89bfc9afec04e44aa3528805cad37` |
| `cmd/scribe/sync.go` | `572d757f1cb523aa6fe9df2d58c9c0ce7a9d59798f7dfd5875c76265631ccc29` |
| `cmd/scribe/sync_sessions.go` | `390e3ea22a49091cf877ebc65b20c3c3a6b36f545be1736d91bfd40875572c6c` |
| `README.md` | `ef744b3cb91bc6e97a0043c592e8f282451780317fd7fc19d6a12799fe0150f6` |
| `cmd/scribe/prompts/session-extract.md` | `15e31a5061f1e2a198c029bf62e9c7979f44efa38f43d51b264a51d6886db49b` |

### Initial completion gate: stopped

Gate result: **BLOCKED**. Independent passes completed: **3**. Pass-3 technical verdict: **ready with fixes**; this is separate from the blocked gate result.

The initial gate result was **not-evaluable** because no fresh reviewer context was permitted. The user subsequently authorized sequential fresh read-only reviewers, with at most five passes. That authorization supersedes the earlier review-context blocker.

Pass 1 technical verdict: **ready with fixes**. It found one supported high-confidence P2 issue, **JEP-1**: an unchanged pre-existing rolling symlink hid writes to its backing file. The old snapshot compared only link text, so it could report success or a nonfatal child error while corrupted backing bytes remained stageable. The reviewed `session_rolling.go` hash was `851e250c90f96067eece067be781566697f0df4ada04fbd9d4eb31049ab76a53`.

JEP-1 disposition: **corrected locally; not repeated in pass 2**. Normal tools-mode preflight rejects Markdown symlinks in KB content directories before launching the child. This bounds unsupported S1 aliases without following or restoring referents. It does not reject untouched invalid regular files. README documents the containment, and the two new red/green scenarios prove the correction.

Pass 2 technical verdict: **not ready**. It found one supported high-confidence P1 issue, gate **JEP-2** (reviewer ID **pass-2/JEP-1**). After a parent directory became an in-KB alias, restoration wrote captured bytes into another project's untouched file. The path guard rejected outside-KB aliases but permitted aliases within the KB. The reviewed `session_rolling.go` hash was `1de55c5b0f1366a9a326178ef9ec362e8c6af9517caf84a119c79e5684a5e2d8`.

JEP-2 disposition: **corrected**. A restoration-specific check rejects symlink ancestors below the trusted selected root before any write or removal. The parent-replacement and removal checks failed before this correction and now pass. The stable-root-alias positive control also passes. Reviewer-reported baseline evidence established the unrelated-file preservation assertion on the preserved pre-S1 overlay, with 379 source hashes matched.

Pass 3 found one supported high-confidence P2 issue: gate **JEP-3**, reviewer ID **pass-3/JEP-1**. Status at the initial gate stop: **OPEN**. Its trigger and failing observations are retained above. That review's `session_rolling.go` hash was `252ddd416a184122ca5d36ee8765c04e1a44ae916dc12e1686aae59a7f36a064`; the table above now identifies the later authorized correction.

The `japanese-engineering-practices:adversarial-code-review` gate stops when new material findings occur in three consecutive passes. That condition was met. The alias-inventory failure also survives the earlier narrower alias correction. The controller therefore stopped corrections and review; no fourth pass or nested gate ran.

At the initial gate stop, JEP-1 and JEP-2 were corrected and JEP-3 remained unfixed. That gate's BLOCKED result remains historical and is not replaced by the later successful checks.

### Authorized JEP-3 correction and fresh gate

The user explicitly reopened this bounded work on 2026-11-06: reject nested directory aliases, preserve KB-root aliases, rerun S1 checks, and start a fresh bounded independent gate. S2 remains excluded.

JEP-3 disposition: **corrected; fresh independent gate passed**. The original wrong outcomes were reproduced before the fix. Preflight now inspects existing nested symlink targets and rejects directory aliases before the child can run. It does not descend into those directories or restore their referents. This leaves backing bytes and processed markers unchanged for the unsupported input.

Fresh gate result: **PASSED**. Independent passes completed: **1**. Technical verdict: **ready**. No supported material findings remain within the reviewed seven-file S1 scope.

The reviewer verified all seven hashes before and after review and reconstructed the target patch from verified baseline origins. The parent independently reconfirmed the unchanged hashes and the 25 focused and 28 race pass events, with zero failures or skips.

Independence audit: the reviewer made a generic lex lookup for `scribe overview`. The controller audited the returned paths, titles, and frontmatter snippets. They contained no authoring discussion, prior review conclusions, or gate outcomes; no documents were fetched. The controller established independence for the pass.

The fresh gate is separate from the initial three-pass BLOCKED gate. No nested gate was started. S1 is VERIFIED only for the bounded normal tools-mode scenario in the slice contract. S2–S5 remain unstarted, provisional, and unauthorized.

## Learned

The original in-place mechanism established only the named passing cases; the directory-alias counterexample disproved its complete S1 guarantee. The new bounded correction prevents the unsupported input before mutation instead of extending snapshot traversal. Snapshot or helper tests alone do not establish checkpoint safety; the existing assembled checks retain that proof boundary.

The fatal-stop guarantee applies to the current sync invocation. It is not a persisted quarantine across future processes, and this work does not claim recovery after a process crash.

The selected KB root is an existing trusted routing input and can be a stable alias. Restoration must reject aliases below that boundary, including in-KB destinations, without rejecting the configured root itself. The injected parent fault establishes this recovery boundary; it does not establish safety against a separate concurrent mutator between path checks and I/O.

## Next

Retain the verified S1 path and its evidence. The initial stopped gate remains BLOCKED; the separate fresh gate passed after the authorized correction.

At the S1 handoff, S2–S5 were unstarted. The user subsequently authorized their sequential reassessment, implementation, checks, and bounded independent gates. No whole-PRD implementation claim is supported.
