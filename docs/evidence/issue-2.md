# Issue #2: tools-mode rolling files

Date: 2026-11-06
Verdict: S1–S3 VERIFIED within their slice boundaries; S3 fresh independent gate PASSED after one pass. The initial S1 gate remains historically BLOCKED.
Confidence: high

Evidence applies to the named local checks and reviewer probes. The initial gate's directory-alias counterexample is retained below. The authorized correction passes matching regressions and the fresh independent gate. No deployed behavior or whole-PRD completion is claimed.

## Tried

Implement only S1 from the [slice plan](../issue-2-rolling-frontmatter-plan.md). A fixture executable replaces the model service while the production process adapter, filesystem writes, validation, processed log, and local Git operations remain real.

The first regression ran before production edits. It reproduced a successful mining result with all nine required fields missing. The implementation now serializes tools-mode sessions and wraps normal-session rolling writes in capture, repair, validation, and restoration.

## Findings

- [`session_rolling.go:191`](../../cmd/scribe/session_rolling.go#L191) owns the tools transaction. S1 proved the normal path and missing project `learnings.md` metadata from supported evidence; S2 extends it below.
- [`session_rolling.go:207`](../../cmd/scribe/session_rolling.go#L207) rejects pre-existing nested directory aliases before child launch. Markdown symlinks and uninspectable alias targets also fail preflight. The selected KB root is outside that nested inventory, so stable root aliases remain supported.
- [`session_rolling.go:163`](../../cmd/scribe/session_rolling.go#L163) rejects symlink ancestors below the selected KB root before restoration writes or removals. Stable configured KB-root aliases remain supported.
- [`session_rolling.go:261`](../../cmd/scribe/session_rolling.go#L261) excludes fenced and indented examples from entry provenance. Both delimiter-length and indented-closing cases are covered.
- [`validate.go:167`](../../cmd/scribe/validate.go#L167) validates proposed content without first writing it to the article path.
- [`sync_sessions.go:502`](../../cmd/scribe/sync_sessions.go#L502) wraps the tools subprocess; success marking follows validation. Normal and large cost labels remain distinct, and rate-limit and daily-budget stops remain supported.
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

### Historical S1 reviewed code identity

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

The fresh gate is separate from the initial three-pass BLOCKED gate. No nested gate was started. At this handoff, S1 was VERIFIED only for the bounded normal tools-mode scenario in the slice contract; S2–S5 were unstarted. Their subsequent authorization and S2 work are recorded below.

## Learned

The original in-place mechanism established only the named passing cases; the directory-alias counterexample disproved its complete S1 guarantee. The new bounded correction prevents the unsupported input before mutation instead of extending snapshot traversal. Snapshot or helper tests alone do not establish checkpoint safety; the existing assembled checks retain that proof boundary.

The fatal-stop guarantee applies to the current sync invocation. It is not a persisted quarantine across future processes, and this work does not claim recovery after a process crash.

The selected KB root is an existing trusted routing input and can be a stable alias. Restoration must reject aliases below that boundary, including in-KB destinations, without rejecting the configured root itself. The injected parent fault establishes this recovery boundary; it does not establish safety against a separate concurrent mutator between path checks and I/O.

## Next

Retain the verified S1 path and its evidence. The initial stopped gate remains BLOCKED; the separate fresh gate passed after the authorized correction.

At the S1 handoff, S2–S5 were unstarted. The user subsequently authorized their sequential reassessment, implementation, checks, and bounded independent gates. No whole-PRD implementation claim is supported.

## S2: remaining tools-mode variants

Date: 2026-11-06. Status at the S2 gate handoff: VERIFIED; focused, race, and full local checks passed; fresh independent gate PASSED after one pass. S3–S5 were unstarted at that handoff and required sequential reassessment. S1 was committed separately as `335cb44`. The user later requested the S2 commit; see the [isolated commit verification](issue-2-s2-commit.md).

### Tried and findings

S2 retains the S1 transaction and extends its selection and assessment through normal tools-mode mining, then through large tools-mode mining. Its [current slice contract](../issue-2-rolling-frontmatter-plan.md#s2-complete-protected-tools-mode-rolling-file-variants) supplies B1–B6.

- [`session_rolling.go:83`](../../cmd/scribe/session_rolling.go#L83) recognizes standard/configured project targets and yearly archives, marked files, and malformed headers with a top-level rolling declaration. A parse error must not silently turn explicit rolling output into an ordinary article.
- [`session_rolling.go:235`](../../cmd/scribe/session_rolling.go#L235) assesses every changed selected file. Missing headers/fields are repairable when dated entries establish provenance. Full directory/filename titles retain Unicode; allowed domains or `general` supply missing domains. Existing invalid values remain unresolved instead of being replaced.
- [`session_rolling.go:325`](../../cmd/scribe/session_rolling.go#L325) rejects invalid scalar metadata and calendar dates before persistence. Proposed-content validation still rejects contradictory date ranges. Existing valid fields, custom data, and body bytes remain unchanged.
- [`sync_sessions.go:502`](../../cmd/scribe/sync_sessions.go#L502) protects both normal and large tools prompts. [`sync_sessions.go:962`](../../cmd/scribe/sync_sessions.go#L962) propagates large-session restoration failure immediately, before scan progress or later phases.

The first normal real-child check failed on a repairable marked nonstandard file. After the shared assessment change, nine selected variants passed. The inspected full result contained the expected title, defaults, date range and source; code examples did not contaminate provenance, and the original body bytes survived. That observation preceded large-session implementation and supported retaining the mechanism.

Further discriminating checks reproduced malformed nonstandard output escaping selection, impossible existing calendar dates and empty confidence being accepted, and the unprotected large path accepting invalid output or leaving caught-error writes. A full-sync large restoration fault launched another session and committed before the correction. A new unclassified Markdown alias also falsely recorded success. The latter now causes an incomplete-inventory fatal stop without alias traversal or referent repair; known rolling leaf-alias recovery remains supported. The fixture changes an alias only; it does not claim recovery of an unknown backing file.

### Check and observed output

All Go commands use `GOCACHE=/private/tmp/scribe-issue2-go-cache`:

```sh
go test -tags sqlite_fts5 ./cmd/scribe -run '^TestIssue2Tools(Rolling|Variants)' -count=1 -json
go test -race -tags sqlite_fts5 ./cmd/scribe -run '^TestIssue2Tools(Rolling|Variants)|^TestMineSessionBatches_StopsAfterRateLimit$|^TestRecordSessionProcessed$|^TestApplyMetaRollingAppend_InitFrontmatterLintValid$' -count=1 -json
make check
```

Focused: **passed**, 23 top-level tests plus 35 subtests = 58 pass events, zero failures/skips; package 12.666 seconds. Race: **passed**, 26 top-level tests plus 35 subtests = 61 pass events, zero failures/skips; package 12.837 seconds. Full `make check`: **passed**, exit 0. Its observed output was:

```text
go test -tags "sqlite_fts5"  -count=1 -timeout 20m ./...
ok  github.com/oliver-kriska/scribe/cmd/scribe  36.907s
go vet -tags "sqlite_fts5" ./...
```

The full check used approved local-loopback listener access for the existing hermetic `httptest` fixtures. No live service or model calls were enabled.

| Case | Local observation |
| --- | --- |
| B1 | Both real tools prompts produce valid standard/custom/archive/marked outputs, including a missing header and Unicode filename. Pending marker is observed before validation; successful marker follows. Ordinary child-created and untouched invalid regular files remain unchanged. |
| B2 | Defaults, allowed/fallback domains, full titles, valid custom/contributor fields, preserved dates/sources and body bytes pass. Fenced/indented provenance examples remain excluded; missing references recover to an empty list. Malformed YAML, explicit false, invalid/empty values and unresolved/contradictory dates are rejected. Metadata-rejection fixtures complete successfully as children. |
| B3 | Both prompts demonstrate mutations before caught failure. Existing rolling variants return to original bytes, new variants disappear, markers stay absent and unrelated bytes survive. A later invalid file also rolls back files repaired earlier in that session. |
| B4 | Large tools-mode checks use the production large prompt and real process adapter, filesystem and processed log. |
| B5 | Real large-session Git checkpoint contains validated output and matching marker, excluding failed output. The full-sync restoration-fault case stops later children, staging and final commit, leaves HEAD unchanged and does not advance scan time. The harness injects the filesystem fault; no model capability to create it is claimed. |
| B6 | S1 regular/alias/root controls pass. Both prompts retain rate-limit rollback/stop and daily-budget refusal before child launch. The focused race run retains the three named processed-log/rate/initialization regressions. |

Raw artifacts: `/private/tmp/scribe-issue2-s2-before-20261106/{normal-red.log,normal-green.log,large-rejection-red.log,variants-green.log,limits-green.log,alias-red.log,alias-green.log,focused.jsonl,race.jsonl,make-check.log}`. Initial compile-only fixture setup errors were corrected before the preserved product-failure reproductions; they are not counted as product failures.

### S2 target identity and preservation

Baseline: `335cb44a7c24656dddd8cd06ac007499f9b9c8ec` plus pre-existing unrelated working changes. The scratch root above retains 387 baseline files, their hashes, original status and tracked patch. `final-target-hashes.json` identifies the frozen seven-file S2 product target. Unrelated baseline file hashes remain unchanged; current source still includes their pre-existing edits.

| S2 final file | SHA-256 |
| --- | --- |
| `cmd/scribe/session_rolling.go` | `a74465ec35448fc31cddb7c38f9e9deeeb3d548821e55630db210648c473e4fa` |
| `cmd/scribe/sync_sessions.go` | `adfbbeb55af55acccc8e78ffd49582453b8ec6f6122729c91523ef3239e92fe9` |
| `cmd/scribe/issue2_tools_variants_test.go` | `740f6f25228a1c1652be2831e25992f3d98eb7eecff3d8f3d0b22d978cfee80c` |
| `README.md` | `96052d88bf3a63cec9246fe9ee14a9a298d644c1bdab7363439c196f29b04018` |
| `cmd/scribe/prompts/session-extract.md` | `df8837d00abad04a0a9ec1bdcc290482cc13dfe500ddcecc2b6af61f94bbf413` |
| `cmd/scribe/prompts/session-extract-large.md` | `8735c03bda4efe76edf527374abf4bb223ebbac0f45f5c25e1b03d77df5165a8` |
| `cmd/scribe/templates/kb-CLAUDE.md` | `95ae039c2e6799f9fa954ef9a98d4b6e7d1d73304ae93e3e1306ac52a26492b5` |

### S2 gate and next decision

Fresh independent gate: **PASSED**, one independent pass, technical verdict **ready**, no supported material findings. The reviewer verified all seven target hashes before/after and six pre-S2 file origins against the preserved baseline. Package/module origins resolved to the reviewed checkout. The reviewer inspected matching 58 focused and 61 race outcomes plus full test/vet results, then independently ran:

```sh
go test -tags sqlite_fts5 ./cmd/scribe -run '^TestIssue2ToolsVariants' -count=1 -json
```

Independent result: **passed**, 33 pass events, zero failures/skips. Raw artifact: `/private/tmp/scribe-issue2-s2-review/pass-1/independent-focused.jsonl`. The parent independently reconfirmed all seven unchanged hashes and the 33 independent passes. The historical S1 blocked gate and later passed S1 gate retain their separate identities.

Retain verified S2 and reassess S3. This proves the named local tools-mode behavior, not deployed behavior or whole-PRD completion. No S2 commit, push, installation, live KB change, authenticated model call or publication ran. Envelope integration and bulk repair remain unimplemented at this handoff.

## S3: protected envelope application

Date: 2026-11-06. Status: VERIFIED; focused/race/full local checks passed; fresh independent gate PASSED after one pass. S4/S5 remain unstarted at this handoff. S2 was subsequently committed separately as `eb48423`; its commit verification is a separate record linked from the evidence index. No S3 source was included in that commit.

### Tried and findings

The [S3 contract](../issue-2-rolling-frontmatter-plan.md#s3-isolate-envelope-apply-while-generation-stays-parallel) retains S1/S2 rules and adds envelope isolation across ccrider normal/large sessions and the separate Codex mining driver.

- [`session_mine_envelope.go:152`](../../cmd/scribe/session_mine_envelope.go#L152) generates envelopes without mutations. Workers run these model calls concurrently; the collector at [`session_mine_envelope.go:310`](../../cmd/scribe/session_mine_envelope.go#L310) alone applies, validates/restores, records checked markers and checkpoints. Fatal failures cancel/drain generation without applying queued results.
- [`session_mine_envelope.go:184`](../../cmd/scribe/session_mine_envelope.go#L184) shares the protected apply boundary with Codex. [`session_rolling.go:195`](../../cmd/scribe/session_rolling.go#L195) runs its checked completion callback after rolling validation. A failed marker write restores rolling output and never increments completion counts.
- [`wiki_actions.go:470`](../../cmd/scribe/wiki_actions.go#L470) keeps rolling metadata out of generic clamps/date stamps for session output. Ordinary articles retain normalization and collision protection. New rolling meta files start with a marker; S2 policy supplies missing metadata from entry evidence. The existing entity collision path accepts the session-scoped options through [`entity_collision.go:25`](../../cmd/scribe/entity_collision.go#L25).
- Codex uses its checked `markCodexProcessed` primitive after validation. Fatal restoration propagates through the driver and [`sync.go:361`](../../cmd/scribe/sync.go#L361), stopping subsequent phases and the final commit. Corrective-retry rate/budget limits also stop Codex mining while leaving sessions pending.

The first assembled SQLite → generated envelope → real rolling append → validation → persisted marker check reproduced today's dates and empty sources instead of entry provenance. Its repaired result recovered 2026-10-01/2026-11-01 and the explicit source, preserving the appended body and ignoring a forged model-supplied success ID. This inspected result preceded scheduling and Codex changes.

The deterministic isolation check first reproduced sibling completion/checkpointing while a failed session was held after a demonstrated rolling mutation. After the stage split, two primary generations rendezvous concurrently, but another generation completing cannot apply or checkpoint during that held mutation. Restoration completes before the successful sibling applies. A real Git checkpoint then contains only validated sibling output and its matching marker while a third generation remains held.

Codex regressions first reproduced invented dates, invalid rolling output being counted/marked, and a marker-write error still counting success while leaving changes. All now pass. The large envelope dispatch also incorrectly selected the normal prompt family; its exact `large-session` label now selects the large family without changing operation labels. A corrective-retry daily-budget error now retains the stop result instead of continuing Codex candidates.

### Check and observed output

All final Go commands used `GOCACHE=/private/tmp/scribe-issue2-go-cache`:

```sh
go test -tags sqlite_fts5 ./cmd/scribe -run '^TestIssue2(Tools|Envelope)' -count=1 -json
go test -race -tags sqlite_fts5 ./cmd/scribe -run '^TestIssue2(Tools|Envelope)|^TestMineSessionBatches_StopsAfterRateLimit$|^TestRecordSessionProcessed$|^TestApplyMetaRollingAppend_InitFrontmatterLintValid$|^TestRejectedSessionStaysPending$|^TestEntityCollision|^TestCodexProcessedLog' -count=1 -json
make check
```

Focused: **passed**, 31 top-level tests plus 81 subtests = 112 pass events, zero failures/skips; package 15.922 seconds. Race: **passed**, 38 top-level tests plus 81 subtests = 119 pass events, zero failures/skips; package 16.126 seconds. Full `make check`: **passed**, exit 0:

```text
go test -tags "sqlite_fts5"  -count=1 -timeout 20m ./...
ok  github.com/oliver-kriska/scribe/cmd/scribe  37.434s
go vet -tags "sqlite_fts5" ./...
```

The full check used approved local-loopback access for existing hermetic `httptest` fixtures. Fixture providers replace model reasoning/service access; transcripts, parsed envelopes, filesystem mutations, validation, both logs, restoration and local Git remain real.

| Case | Local evidence |
| --- | --- |
| C1 | Complete representative result inspected before expansion: recovered dates/source, preserved append body, complete valid metadata and only the actual session marker. |
| C2 | Normal/large/Codex caller variants pass for standard/configured targets, archives, marked Unicode content files, missing headers, provenance examples, curated/custom fields and body bytes. All three reject malformed, false, invalid type/domain/confidence, unresolved and contradictory output. Ordinary-article normalization and untouched invalid-file isolation remain intact. |
| C3 | Two primary calls rendezvous. Parent observes failed-session bytes before releasing failure. A later generation starts without applying its queued sibling, proving generation can advance while the single apply phase remains held. Successful sibling content survives restoration; failed content/marker do not. |
| C4 | Real local Git checkpoint inspected while third generation is held: validated successful sibling only, matching marker, no failed or incomplete output. No checkpoint occurs while the failed apply is held. |
| C5 | Both marker-log failures restore an observed rolling mutation and report zero completion. Full sync normal/large/Codex filesystem-fault cases observe actual mutation, then force restoration failure. Each returns the fatal error, leaves sessions pending and HEAD/staging unchanged. No serial Codex candidate/phase runs afterward; already in-flight ccrider generation is cancelled/drained without apply. |
| C6 | Actual fixture Codex rollout/scoring driver and separate log pass success/rejection cases. Empty transcripts, no-action envelopes, low-score skips, rate/budget limits and corrective-retry limits pass. Race selection retains collision/provenance, processed-log, rate and rolling-init regressions. |

For a discriminating fatal-stop baseline check, copied 382 command source/input files from the preserved pre-S3 snapshot into `baseline-probe/`, verified their hashes, then overlaid only the new regression file and unchanged module files from `eb48423`. Ran:

```sh
go test -C /private/tmp/scribe-issue2-s3-before-20261106/baseline-probe -tags sqlite_fts5 ./cmd/scribe -run '^TestIssue2EnvelopeFatalStopsSync$' -count=1 -v
```

Baseline result: **failed in all three caller subcases**. The prior source continued after the caught failure and committed; the expected fatal error was nil. Current matching cases pass. Initial full-sync fixture barrier failures came from a shallow excluded project path and Codex filename ordering; those were corrected before the baseline comparison and are setup failures, not product evidence. Parent-directory replacement is a harness-injected filesystem fault, not a claimed model tool capability.

Raw artifacts under `/private/tmp/scribe-issue2-s3-before-20261106`: `representative-{red,green}.log`, `isolation-{red,green}.log`, `codex-{red,green}.log`, `variants-first.log`, `limits-{first,green}.log`, `envelope-green.log`, `fatal-baseline-red.log`, `focused.jsonl`, `race.jsonl`, and `make-check.log`.

### S3 target identity and preservation

The pre-S3 snapshot contains 388 file origins and hashes, taken before product edits at `335cb44` plus verified S2 and unrelated working changes. HEAD subsequently advanced to `eb48423` for the isolated S2 commit; that operation preserved working bytes. S3 review uses these pre-S3 file origins, not the entire current Git diff. Unrelated original source hashes remain unchanged.

`final-target-hashes.json` and `target.patch` in the scratch root identify the frozen 13-file S3 product delta:

| S3 final file | SHA-256 |
| --- | --- |
| `cmd/scribe/session_rolling.go` | `0c4f0b174bc844bdf49c3d395cca487d1393d8507102f0a5b75850bc449235f0` |
| `cmd/scribe/session_mine_envelope.go` | `d5a61f66bd934ec416335f05221be0edade38d829664ff506fc27fdd1fc474fb` |
| `cmd/scribe/entity_collision.go` | `fde18e68165ce7b3ac7506e1d21f2b17ea355667928185eb23b5cb60b5c9a2ee` |
| `cmd/scribe/wiki_actions.go` | `f9863c9d0a2f5a701607a324ea09a0e074ff636ed0b01d62b43d027519d0c94f` |
| `cmd/scribe/codex_mine.go` | `cbf4963569a4e03d106cb8c2643cc087bcfbcfe118e8eb498306ae52bd1a7822` |
| `cmd/scribe/sync.go` | `fced13377d92653a06fce8d2ac47b051cbd2a8892bb9863a6f4b7ac579ca3c55` |
| `cmd/scribe/issue2_envelope_rolling_test.go` | `9bdffed6cf3820e4851513ba39dd6666117e88eacf0e00e5c96de6046252cd03` |
| `README.md` | `e18bde8cf2f14a807dcb61369b4e47799089799a868c01276b4b9225a754e4c2` |
| `cmd/scribe/prompts/session-mine-anthropic.md` | `19c946b303d41524fb0c0486777c607a4c83c90418ff18cc330a4620a5a19fb7` |
| `cmd/scribe/prompts/session-mine-ollama.md` | `877f4e83e5e2e22505b10c7620f8322b2b053bd3d3bc736b4d0fcc0da10c1588` |
| `cmd/scribe/prompts/session-extract-anthropic.md` | `e1f8a38ec6c39bba621498ab7ef2fa29a1896a3d1d63c3a13bef38becd816edd` |
| `cmd/scribe/prompts/session-extract-ollama.md` | `932c517cea9de7e6ca20ac7b86f2ba679981ee9d2bfb1eccddb75354261812ef` |
| `cmd/scribe/templates/kb-CLAUDE.md` | `76abe0246725d044d739ca491a9037f97b832b12d02f3007d70bdfb173bc584d` |

### S3 gate and next decision

Fresh independent gate: **PASSED** after one independent pass, technical verdict **ready**, no supported findings or testing gaps. The reviewer verified 13/13 final hashes before/after, 12/12 baseline origins, and absence of the new test from the baseline. It independently ran `go test -tags sqlite_fts5 ./cmd/scribe -run '^TestIssue2Envelope' -count=1 -json`: **54 passing named outcomes, zero failures/skips, exit 0**, package 1.976 seconds.

The reviewer reconstructed `335cb44` plus all 388 pre-S3 baseline files and verified module/package origins. The representative check there produced one expected behavioral failure: invented dates and empty sources. Artifacts: `/private/tmp/scribe-issue2-s3-review/pass-1/{independent-focused.jsonl,independent-baseline-representative.jsonl,baseline-probe-origins.json,baseline-probe-package.json,target-hashes.json}`. The parent independently reconfirmed all 13 unchanged hashes, the 54 independent passes, and final full test/vet results. S1/S2 historical gates and commits remain separate. Retain verified S3 and reassess S4.

The rolling-only recovery boundary excludes unrelated article output. Primary generation can already be in flight when a fatal error occurs; queued envelopes are discarded and workers drained before return. These local caught-failure checks do not prove crash recovery, external-mutator safety, deployed behavior, or whole-PRD completion. No S3 commit, installation, push, live KB or authenticated-model action ran.
