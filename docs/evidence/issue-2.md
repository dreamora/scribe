# Issue #2: tools-mode rolling files

Date: 2026-11-06
Verdict: S1–S5 VERIFIED locally; current R1–R9 acceptance audit complete with no open material gaps. The expanded S5 gate PASSED after two passes. The initial S1 gate remains historically BLOCKED.
Confidence: medium

Evidence applies to the named local checks and reviewer probes. Historical counterexamples and their corrections are retained below. The confirmed PRD is implemented and locally verified; deployed behavior is not established.

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

The rolling-only recovery boundary excludes unrelated article output. Primary generation can already be in flight when a fatal error occurs; queued envelopes are discarded and workers drained before return. These local caught-failure checks do not prove crash recovery, external-mutator safety, deployed behavior, or whole-PRD completion. No S3 commit, installation, push, live KB or authenticated-model action ran during that gate handoff. The later local slice commits are recorded in [commit verification](issue-2-slice-commits.md).

## S4: rolling-only CLI preview

Date: 2026-11-06. Status: VERIFIED; focused/race/full local checks passed; fresh independent gate PASSED after one pass. S5 apply remains unstarted at this handoff. S1/S2 commits and the prior gates remain separate; no S4 commit was created during that handoff. Later local commits are recorded in [commit verification](issue-2-slice-commits.md).

### Tried and findings

The [S4 contract](../issue-2-rolling-frontmatter-plan.md#s4-explicit-rolling-only-repair-preview) adds an explicit read-only rolling assessment, preserving ordinary lint and the selected KB's existing sync coordination.

- [`lint.go:35`](../../cmd/scribe/lint.go#L35) dispatches `--rolling-only` before other lint operations and legacy repair. It requires `--fix` and currently `--dry-run`; apply is explicitly unavailable until S5. Conflicting lint operations fail. [`lint.go:129`](../../cmd/scribe/lint.go#L129) keeps these preview/error paths from appending run records.
- `lint_rolling.go` uses the shared rolling recognizer and byte-preserving proposal policy. Explicit paths retain normal filesystem argument semantics and must stay within selected content directories. `--changed` retains its existing precedence and scope. Full selection also includes marked files that the ordinary article walker omits, such as `_memory.md`.
- Preview reports candidate counts as scanned, would repair, unchanged and unresolved. Mixed inputs report every result; unresolved files make the CLI exit nonzero without content replacement. Unsafe aliases, inaccessible inputs and invalid metadata remain unresolved. Stable selected-root aliases work, including explicit relative arguments.
- [`lockfile.go:52`](../../cmd/scribe/lockfile.go#L52) writes bounded operation/PID details only after acquiring the existing inode's nonblocking lock. [`sync.go:107`](../../cmd/scribe/sync.go#L107) labels sync; preview labels itself. An unlabeled caller clears old details after acquiring the lock, so busy reporting does not reuse a released holder's identity. Unknown holder details are reported truthfully.

The first production CLI subprocess rejected the unknown flag with exit 80. After the isolated dispatch it exits 0 with:

```text
would repair projects/acme/learnings.md
scanned 1, would repair 1, unchanged 0, unresolved 0
```

The complete temporary-KB file/directory/hash comparison remained unchanged, including an unrelated duplicate artifact and absence of run records. This inspected representative preceded expanded selection and lock variants.

Expanded checks first exposed omitted marked underscore files, silent alias omissions, rejected stable root-alias selectors, missing busy-owner details, and selection of an explicitly named non-Markdown file. Those regressions now pass. No model, duplicate-cleanup or legacy repair path supplies the preview result.

### Check and observed output

Tests execute the real production `main()` and Kong command path in a child test executable, with actual CLI arguments, exit codes, temporary KBs/Git and OS locks. The child startup harness replaces no selection, repair assessment, lock, output or persistence behavior. Guard executables prevent accidental Claude/Codex/qmd/ccrider calls. The helper entry test is excluded from focused selection; its CLI invocations are the proof boundary.

All final Go commands use `GOCACHE=/private/tmp/scribe-issue2-go-cache`:

```sh
go test -tags sqlite_fts5 ./cmd/scribe -run '^TestIssue2(Tools|Envelope|RollingPreview)' -count=1 -json
go test -race -tags sqlite_fts5 ./cmd/scribe -run '^TestIssue2(Tools|Envelope|RollingPreview)|^TestLint|^TestChangedWikiFiles|^TestKBLockScope|^TestLockPathFor|^TestAcquireLock|^TestMineSessionBatches_StopsAfterRateLimit$|^TestRecordSessionProcessed$|^TestApplyMetaRollingAppend_InitFrontmatterLintValid$' -count=1 -json
make check
```

Focused: **passed**, 39 top-level tests plus 99 subtests = 138 pass events, zero failures/skips; package 13.933 seconds. Race: **passed**, 64 top-level tests plus 117 subtests = 181 pass events, zero failures/skips; package 24.353 seconds. Full `make check`: **passed**, exit 0:

```text
go test -tags "sqlite_fts5"  -count=1 -timeout 20m ./...
ok  github.com/oliver-kriska/scribe/cmd/scribe  37.323s
go vet -tags "sqlite_fts5" ./...
```

The full check used approved local-loopback listener access for existing hermetic `httptest` fixtures. No live network/model services were enabled.

| Case | Local observation |
| --- | --- |
| D1 | Actual CLI exits 0, reports one proposed repair and leaves the complete fixture tree unchanged. No duplicate summary or run record appears. |
| D2 | Full selected-KB preview includes standard/configured targets, yearly archives, untouched projects, Unicode marked files and a marked underscore file. Readable unrelated articles, raw content and non-Markdown files are excluded. Shared mining/default/provenance regressions remain green. |
| D3 | Explicit arguments, duplicate arguments, tracked/untracked `--changed` files and its precedence pass. Outside-root paths fail before content assessment. Root aliases retain supported selection. |
| D4 | Mixed preview reports `scanned 7, would repair 4, unchanged 1, unresolved 2` and exits nonzero. Malformed/false/confidence/calendar/contradictory/undated cases report causes and remain byte-identical. Code-only example dates do not supply provenance. |
| D5 | Missing `--fix`, intermediate apply requests, duplicate-scan and model-operation combinations fail visibly without content writes. CLI help names the boundary. Ordinary `lint --fix --dry-run` retains its result and does not write content. |
| D6 | Real cross-process flock contention reports operation/PID and preserves the held inode/details. Root aliases contend on the same lock; a different KB with the same lock directory proceeds. After a plain existing caller acquires, stale owner details are cleared and preview reports unavailable details. After release preview succeeds; no content changes occur. |

Raw artifacts: `/private/tmp/scribe-issue2-s4-before-20261106/{representative-red.log,representative-green.log,expanded-red.log,expanded-green.log,markdown-filter-red.log,preview-green.log,preview-final-precheck.log,focused.jsonl,race.jsonl,make-check.log}`. A compile-only fixture import error was corrected before the successful representative; it is not product-failure evidence.

### S4 target identity and preservation

Baseline: `eb48423` plus verified S3 and preserved unrelated working edits. The scratch root retains all 425 source/module/document origins and hashes under `baseline/` and `baseline-hashes.json`. `target.patch` is the exact S4 delta against those origins; whole Git diff is not the review target. Six existing file origins and two new files form the frozen product target. Unrelated baseline source hashes remain unchanged.

| S4 final file | SHA-256 |
| --- | --- |
| `cmd/scribe/lint.go` | `eb07f092e6aae75d42509ba380f1c45fdba6b5f0c4538c688527fbd3d3d21ca2` |
| `cmd/scribe/lint_rolling.go` | `0fb92a21c04ee88175bd4926044167d96dc0a8a4c7a50ade1a063c091563b8c0` |
| `cmd/scribe/lockfile.go` | `a0808733b9b1631d241306731e2d0bc68de318078c5639e99bbe29ff216b1b62` |
| `cmd/scribe/sync.go` | `de60c5bef96fc5debb6e231d87c46e690fddbc27f4cfc9b9f74a80210bf93f14` |
| `cmd/scribe/issue2_rolling_preview_test.go` | `454b11e432dd140bf5872399b29556e267d6a2000f7c9293ef10dfa03c67de0a` |
| `README.md` | `757cda827413bb568a8f8fae3e67896ead36ea24b2994ad15b8b4e8079d9086b` |
| `cmd/scribe/templates/kb-CLAUDE.md` | `0601ca7e872c3624d9d543bc4dda9423aabac30340b57833747c8c4ab9631394` |
| `cmd/scribe/skills/scribe-cli/SKILL.md` | `c07981110a7a6b5d640f4d09d98e5ce8c9e3cfa4a7ccc1724fa40be012c2bbbc` |

### S4 gate and next decision

Fresh independent gate: **PASSED** after one independent pass, technical verdict **ready**, no supported findings or testing gaps. The reviewer verified eight final hashes before/after, all 425 baseline file origins, six old/two new target origins, and the main Go module/package checkout origins. Of the baseline files, 415 non-target files were unchanged and four expected canonical documentation records were excluded from the product target.

Independent command: `go test -tags sqlite_fts5 ./cmd/scribe -run '^TestIssue2RollingPreview' -count=1 -json`. Result: **26 passes, zero failures/skips, exit 0**, package 0.714 seconds. Raw artifact: `/private/tmp/scribe-issue2-s4-review/pass-1/independent-focused.jsonl`. The reviewer reused matching 138 focused/181 race passes and terminal full test/vet results; it did not claim independent baseline execution. The parent independently reconfirmed eight unchanged hashes and 26 independent passes. Retain verified S4 and reassess S5.

This proves named local preview and coordination behavior. Apply/atomic replacement and repeated repair remain unimplemented for S5. Lock-file metadata is coordination state; fixture locks are outside KB content. Unknown owner details and unsupported aliases remain explicit. No live KB repair, installed binary, commit, push, or authenticated model action ran.

## S5: per-file rolling repair apply

Date: 2026-11-06. Status: VERIFIED; expanded focused/race/full local checks passed, independent gate PASSED after two passes, and current R1–R9 audit complete. Prior slice gates and the S1/S2 commits remain separate. Local slice commits followed on 2026-11-08; see [commit verification](issue-2-slice-commits.md).

### Tried and findings

The [S5 contract](../issue-2-rolling-frontmatter-plan.md#s5-apply-rolling-only-repairs-independently-per-file) completes the actual apply interface through the same scope, metadata assessment and sync lock as preview.

- [`lint_rolling.go:14`](../../cmd/scribe/lint_rolling.go#L14) requires `--fix`, rejects conflicting lint operations and acquires the same sync lock for preview/apply. [`lint.go:131`](../../cmd/scribe/lint.go#L131) marks only a real apply that entered the acquired-lock phase as mutating, so busy/invalid requests do not append run records.
- [`lint_rolling.go:51`](../../cmd/scribe/lint_rolling.go#L51) shares selection and proposal assessment. The proposal policy validates before replacement; paths/regular-file state are rechecked before writes. Only successful replacements count as repaired. Errors remain per-file and do not stop valid siblings.
- [`wiki_actions.go:1198`](../../cmd/scribe/wiki_actions.go#L1198) uses the existing same-directory atomic writer with an explicit permission-mode variant. The temporary file receives the original permission bits before replacement; default atomic-writer callers retain their prior umask behavior. Write/close/rename failures clean the temporary file and leave the target unreplaced.

The first real CLI apply reproduced S4's explicit refusal. After implementation it exits 0, reports `scanned 1, repaired 1, unchanged 0, unresolved 0`, and persists supported dates/sources/defaults while preserving body, contributor/custom/high-confidence values and mode 0600. Inode replacement and the complete resulting article were inspected before expanding variants. Unrelated duplicates, placeholders and processed-log bytes stayed unchanged; real apply retained its normal run record.

Expanded checks exposed two additional problems. Passing mode 0660 to the generic writer under umask 077 produced 0600; the exact-mode adapter now preserves 0660 without changing generic caller semantics. Busy apply initially wrote an error run record; the acquired-apply state now prevents any KB file creation on busy requests. Both have discriminating red/green results.

### Check and observed output

All final commands use `GOCACHE=/private/tmp/scribe-issue2-go-cache`:

```sh
go test -tags sqlite_fts5 ./cmd/scribe -run '^TestIssue2(Tools|Envelope|RollingPreview|RollingApply)' -count=1 -json
go test -race -tags sqlite_fts5 ./cmd/scribe -run '^TestIssue2(Tools|Envelope|RollingPreview|RollingApply)|^TestLint|^TestChangedWikiFiles|^TestKBLockScope|^TestLockPathFor|^TestAcquireLock|^TestMineSessionBatches_StopsAfterRateLimit$|^TestRecordSessionProcessed$|^TestApplyMetaRollingAppend_InitFrontmatterLintValid$' -count=1 -json
make check
```

Focused: **passed**, 46 top-level tests plus 107 subtests = 153 pass events, zero failures/skips; package 14.151 seconds. Race: **passed**, 71 top-level tests plus 125 subtests = 196 pass events, zero failures/skips; package 32.330 seconds. Full `make check`: **passed**, exit 0:

```text
go test -tags "sqlite_fts5"  -count=1 -timeout 20m ./...
ok  github.com/oliver-kriska/scribe/cmd/scribe  38.016s
go vet -tags "sqlite_fts5" ./...
```

The full check used approved local-loopback access for existing hermetic `httptest` fixtures. Real CLI main/Kong, filesystem writes, validation, atomic replacement and OS locks remain real. No installed binary, live KB, network/model service or production fault seam is used.

| Case | Local observation |
| --- | --- |
| E1 | Complete persisted representative inspected after real CLI success: metadata/date/source defaults, byte-identical body/valid fields, original mode, inode replacement and normal apply telemetry. |
| E2 | Full apply repairs standard/configured targets, archives, untouched projects, Unicode marked content and underscore-marked files. Explicit/duplicate/changed/changed-precedence/root-alias selectors pass; ordinary/non-Markdown/raw files remain excluded. |
| E3 | Mixed apply reports `scanned 6, repaired 4, unchanged 1, unresolved 1`, returns nonzero, repairs valid siblings and preserves unresolved/valid/unrelated bytes. Repeat reports zero repairs and identical article content; subsequent preview proposes zero. Operational apply run records are excluded from article-stability comparisons. |
| E4 | Child-only `RLIMIT_FSIZE=4096`, with SIGXFSZ ignored, makes the real temporary write fail with `file too large`. Output reports `scanned 2, repaired 1, unchanged 0, unresolved 1`: failed large original remains byte-identical, small sibling validates, and no temporary file leaks. This is a caught write failure, separate from validation rejection; it does not prove crash recovery. |
| E5 | Both busy apply and preview exit nonzero, identify sync/PID and preserve the complete KB tree and holder details. Successful apply still writes normal telemetry. Help/preview/ordinary lint remain checked; only the obsolete intermediate apply-block row was removed and replaced by final-interface coverage. |

The caught write-fault output was:

```text
unresolved projects/acme/decisions-log.md: write tmp: write <fixture>/.scribe-<random>.tmp: file too large
repaired projects/acme/learnings.md
scanned 2, repaired 1, unchanged 0, unresolved 1
scribe: error: rolling repair: 1 unresolved file(s)
```

Here `<fixture>` and `<random>` replace only the temporary location/name from the preserved raw log. The resource limit and umask affect only the child test process. No privileged permission assumption, mocked writer or live process limit is involved.

Raw artifacts under `/private/tmp/scribe-issue2-s5-before-20261106`: `representative-{red,green}.log`, `expanded-{first,green}.log`, `cli-green.log`, `busy-{red,green}.log`, `focused.jsonl`, `race.jsonl`, and `make-check.log`.

### Historical S5 pass-1 target identity

Baseline: `eb48423` plus verified S3/S4 and preserved unrelated working changes. All 427 source/module/document origins and hashes are retained under `baseline/` and `baseline-hashes.json`. The initial eight-file delta below is retained as `pass1-target.patch` and `pass1-target-hashes.json`; current expanded identity follows under AUDIT-1. Whole Git diff is not the review target. Unrelated baseline source hashes remain unchanged.

| S5 initial pass-1 file | SHA-256 |
| --- | --- |
| `cmd/scribe/lint.go` | `85becb0f664eaa3507315ccb7b94a9e50dea6583d595c6b9c2b5d110e8c4e06e` |
| `cmd/scribe/lint_rolling.go` | `f1b831805c012c4320a3eec5f3614ace04f74ef92e5901ef40a7916180c8606c` |
| `cmd/scribe/wiki_actions.go` | `d2e961b97befbf0e5e8302701889d52108a59e69e61aa9a20588b90996a625c3` |
| `cmd/scribe/issue2_rolling_preview_test.go` | `098c9db6d85b254487748c519bf17934a05943a4d042cbc4fcc9107b1f20d49c` |
| `cmd/scribe/issue2_rolling_apply_test.go` | `66b54fcd45c1c5794353a9f82c933daa7c0bc99817fc66188fed82720871fe20` |
| `README.md` | `1b5b1c313d65bf3824f0f9387b6ff279d99fe8d96118fe4b8c98ca3dcf9236da` |
| `cmd/scribe/templates/kb-CLAUDE.md` | `3e7b2b6c1cd174d54f0dcbd406aeb850f761500cec421d56761c815b8389106f` |
| `cmd/scribe/skills/scribe-cli/SKILL.md` | `a055319e798540199c83d6865115f4e5e61f8c96006573dcc98efd65555f2eda` |

### S5 gate and next decision

Initial S5 independent gate: **PASSED** after one pass; technical verdict **ready**, no supported findings/testing gaps. Eight target hashes, seven old origins/new test absence, all 427 baseline files and exact parent `eb48423` were verified with module/package origins. Independent `^TestIssue2Rolling(Apply|Preview)` executed 41 passing outcomes (15 top-level tests), zero failures/skips, exit 0, package 1.180 seconds. The baseline representative produced one expected behavioral assertion failure (apply unavailable), not a setup/compile error. Artifacts: `/private/tmp/scribe-issue2-s5-review/pass-1/{independent-focused.jsonl,baseline-representative.jsonl,baseline-probe-origins.json,baseline-package-origin.txt,post-review-hash-check.json}`. Parent checks reconfirmed the eight hashes and outcomes.

The subsequent whole-current R1–R9 audit found a required lifecycle gap, **AUDIT-1 (P2)**, in the tools completion path. At that point `mineSessionsSerial` incremented count/batch and called void `recordSessionProcessed`; that helper logged persistence errors without returning them. A session could leave rolling output available for a checkpoint without its persisted marker. This conflicted with A5's matching-marker and caught-session completion/recovery intent across R1/R7/R8. It was a missing checked completion boundary, not a new product decision. The correction and refreshed gate below close it.

The initial eight-file pass remains historical. S5 completion was reopened for the bounded correction and refreshed whole-target review. Source/hash origins were captured before correction under `/private/tmp/scribe-issue2-audit1-before-20261106/{baseline/,baseline-hashes.json}`. The original S5 manifest/patch are retained as `pass1-target-hashes.json`/`pass1-target.patch` in the pre-S5 scratch root. The correction reused the existing validated transaction callback and checked marker primitive, preserving rolling-only rollback and unrelated callers.

The named local checks do not establish deployed behavior, crash recovery, unsupported alias traversal or rollback of unrelated articles. No source staging/commit, installation, push, live repair or authenticated model call occurred.

### AUDIT-1 correction target and checks

AUDIT-1 disposition: **corrected and independently verified**. Both normal and large real-child regressions first demonstrated `mined=2`, retained existing/new rolling output and an actual Git checkpoint whose committed marker log still contained its original damaged bytes. The parent observed the child mutations before releasing its successful return. This was a product failure, not setup or compilation failure.

The tools caller now passes a checked `applyMetaSessionsLogAppend` completion callback to [`runRollingSession`](../../cmd/scribe/session_rolling.go#L191). Validation precedes that callback; completion count, batch tracking and checkpoints follow only its success. On callback error, the same rolling restoration/fatal-error path used for model/metadata failures runs. The obsolete tools wrapper and void marker wrapper were removed; their existing tests now use the real shared transaction/checked primitive, retaining original assertions and ownership/idempotence checks. Unrelated production marker callers were not changed.

The matching green regressions demonstrate zero completion, exact old rolling bytes, removal of the new rolling file, preserved damaged-log/control bytes, pending sessions, unchanged HEAD and no staging. Marker-specific failed restoration uses the existing shared error/restore branch; actual full-sync fatal propagation remains covered by the normal/large tools and normal/large/Codex envelope fault tests rather than a new fault mechanism.

Latest commands, with `GOCACHE=/private/tmp/scribe-issue2-go-cache`:

```sh
go test -tags sqlite_fts5 ./cmd/scribe -run '^TestIssue2(Tools|Envelope|RollingPreview|RollingApply)' -count=1 -json
go test -race -tags sqlite_fts5 ./cmd/scribe -run '^TestIssue2(Tools|Envelope|RollingPreview|RollingApply)|^TestLint|^TestChangedWikiFiles|^TestKBLockScope|^TestLockPathFor|^TestAcquireLock|^TestMineSessionBatches_StopsAfterRateLimit$|^TestRecordSessionProcessed$|^TestApplyMetaSessionsLogAppend$|^TestApplyMetaRollingAppend_InitFrontmatterLintValid$' -count=1 -json
make check
```

Expanded focused: **passed**, 47 top-level tests plus 109 subtests = 156 pass events, zero failures/skips; package 14.479 seconds. Expanded race: **passed**, 73 top-level tests plus 127 subtests = 200 pass events, zero failures/skips; package 32.689 seconds. Full `make check`: **passed**, terminal exit 0, tests 38.247 seconds followed by vet. Approved local-loopback access was retained for existing hermetic test servers; no live services/model calls were enabled.

Correction artifacts: `/private/tmp/scribe-issue2-audit1-before-20261106/{baseline/,baseline-hashes.json,marker-red.log,marker-green.log,focused.jsonl,race.jsonl,make-check.log}`. This pre-correction snapshot preserves 428 file origins. The complete expanded S5 target still compares against the **original pre-S5 427-file baseline**, retaining all original eight target files. It now has 14 files: 12 old origins and two new test files. Current `final-target-hashes.json`/`target.patch` are in `/private/tmp/scribe-issue2-s5-before-20261106`.

| Expanded S5 independently reviewed file | SHA-256 |
| --- | --- |
| `cmd/scribe/lint.go` | `85becb0f664eaa3507315ccb7b94a9e50dea6583d595c6b9c2b5d110e8c4e06e` |
| `cmd/scribe/lint_rolling.go` | `f1b831805c012c4320a3eec5f3614ace04f74ef92e5901ef40a7916180c8606c` |
| `cmd/scribe/wiki_actions.go` | `d2e961b97befbf0e5e8302701889d52108a59e69e61aa9a20588b90996a625c3` |
| `cmd/scribe/issue2_rolling_preview_test.go` | `098c9db6d85b254487748c519bf17934a05943a4d042cbc4fcc9107b1f20d49c` |
| `cmd/scribe/issue2_rolling_apply_test.go` | `66b54fcd45c1c5794353a9f82c933daa7c0bc99817fc66188fed82720871fe20` |
| `README.md` | `3b40ef003248d02ec697da75acfb4297072fd13a5da639fe3f01f050c065ea05` |
| `cmd/scribe/templates/kb-CLAUDE.md` | `3e7b2b6c1cd174d54f0dcbd406aeb850f761500cec421d56761c815b8389106f` |
| `cmd/scribe/skills/scribe-cli/SKILL.md` | `a055319e798540199c83d6865115f4e5e61f8c96006573dcc98efd65555f2eda` |
| `cmd/scribe/sync_sessions.go` | `790f74b103097baee093ab9489b03c38c0f2891b0f34a9c9d1921d5a78cae3bb` |
| `cmd/scribe/session_rolling.go` | `a0dca4f97cfe84b80efcf9822d383e3e5a195f8504e3d7b161f729e9dcc1dc16` |
| `cmd/scribe/issue2_tools_rolling_test.go` | `3bbe1f74b321e0e6968dafd95022cc384812408e0f3bd3414bc8d9883d9e8d7b` |
| `cmd/scribe/issue2_tools_marker_test.go` | `91460f1ae980eaf9779dff8d633526f79a695b5dd0042de21931c2ef7e08feee` |
| `cmd/scribe/sessions_test.go` | `f2a9597b85e672428f099e7473328711f49d4fd9f0874a3b8a7e233cc0562bb6` |
| `cmd/scribe/wiki_actions_meta_test.go` | `b98b1af4a3fce2df437390cc384ee06030557e52219839cb46ac1ec1fe619638` |

Refreshed S5 review (pass 2): **PASSED**, latest technical verdict **ready**, no supported findings/testing gaps in the full 14-file scope. The historical eight-file pass-1 result is not relabeled. No source or scope was discarded from the expanded review target.

### Expanded S5 review and final prose identity

The expanded gate **PASSED after two fresh passes**, latest technical verdict **ready**, no supported findings/testing gaps. Pass 2 independently verified all 14 target hashes before/after, original 427-file overlays and module/package origins. Its command `go test -tags sqlite_fts5 ./cmd/scribe -run '^TestIssue2Rolling(Apply|Preview)|^TestIssue2ToolsMarker|^TestRecordSessionProcessed|^TestApplyMetaSessionsLog' -count=1 -json` passed **46 named outcomes**, zero failures/skips, exit 0, package 1.668 seconds. Exact baseline probes produced three expected marker-failure events (normal, large and parent test) and one expected apply-unavailable failure. These were behavioral assertion failures, not setup/compile errors; the current target passed. Parent independently checked those results and unchanged hashes.

Review artifacts: `/private/tmp/scribe-issue2-s5-review/pass-2/{independent-focused.jsonl,independent-baseline-marker.jsonl,independent-baseline-apply.jsonl,independent-baseline-proof.json,independent-baseline-origins.json}` and `/private/tmp/scribe-issue2-s5-review/gate-result.json`. The independently reviewed 14-file identity/patch are retained as `pass2-target-hashes.json`/`pass2-target.patch` under `/private/tmp/scribe-issue2-s5-before-20261106`.

After review, only stale ordinary comments in `claude.go`/`wiki_actions.go` and one test failure-message string in `wiki_actions_meta_test.go` were synchronized to the checked completion writer. Exact inverse replacement/readback proved these were the only changes: no executable, prompt or contract changes. This is separate prose readback, not a claim that the reviewer examined changed hashes. Artifacts `/private/tmp/scribe-issue2-s5-postreview-prose/{before/,before-hashes.json,prose-only.patch,readback.json,final-target-hashes.json,final-target.patch}` preserve the difference and final identity. The final delta retains all 14 reviewed files and adds `claude.go` with one comment-only hunk against the original pre-S5 origin. No material pass 3 or software rerun was needed.

### Current R1–R9 completion audit

Date: 2026-11-06. Result: **all nine requirements and their acceptance boundaries locally VERIFIED; no open material gaps**. This audit inspected current owning source and actual named assertions/outcomes rather than inferring delivery from a helper or slice status. Every `TestIssue2...` named below appears as passing in the latest 156-outcome focused log and 200-outcome race log. Existing ordinary-lint, changed-file, lock, budget/rate and checked-marker regressions also pass in the race/full checks. Historical tests/counts above remain tied to their original targets.

| Requirement | Inspected current boundary and named tests | Observed acceptance result |
| --- | --- | --- |
| R1 | [`runRollingSession`](../../cmd/scribe/session_rolling.go#L191), tools checked callback before counts in [`sync_sessions.go`](../../cmd/scribe/sync_sessions.go#L502), envelope apply/collector and separate Codex checked callback. `TestIssue2ToolsRollingSuccess`, `TestIssue2ToolsVariantsNormal`, `TestIssue2ToolsVariantsLarge`, `TestIssue2EnvelopeRepresentative`, `TestIssue2EnvelopeVariants`, `TestIssue2EnvelopeCodexCaller`, `TestIssue2ToolsMarkerFailureRestoresBeforeCheckpoint`. | Real tools child output is observed invalid with no marker, then repaired/validated before persistence of success. Both normal/large prompts and normal/large/Codex envelope callers create/update valid rolling output. Untouched invalid regular files remain unchanged and do not block unrelated sessions. Failed marker persistence cannot count or checkpoint a session. |
| R2 | `isSessionRollingFile`, [`rollingRepairFiles`](../../cmd/scribe/lint_rolling.go#L144), shared selected-file loop and path guard. `TestIssue2ToolsVariantsNormal`, `TestIssue2ToolsVariantsLarge`, `TestIssue2ToolsVariantsRejectAndRestore`, `TestIssue2RollingPreviewFullScope`, `TestIssue2RollingPreviewSelectors`, `TestIssue2RollingApplySelectors`, `TestIssue2RollingApplyMixedAndStable`. | Standard learnings/decisions, configured targets, each yearly archive variant, headerless/missing-marker files under projects, untouched projects, marked nonstandard/Unicode/underscore content are selected. Malformed explicitly marked YAML and named `rolling: false` conflicts are unresolved. Ordinary articles, raw content and non-Markdown files are excluded. Explicit duplicate selectors deduplicate; --changed retains precedence. Nested aliases reject safely; selected KB-root aliases remain supported. |
| R3 | [`repairSessionRolling`](../../cmd/scribe/session_rolling.go#L309), full-filename Unicode title policy and proposal validation before write. `TestIssue2ToolsVariantsNormal`, `TestIssue2ToolsVariantsLarge`, `TestIssue2ToolsRollingPreservesValues`, `TestIssue2ToolsVariantsPreservesMetadata`, `TestIssue2EnvelopeVariants`, `TestIssue2RollingApplyRepresentative`, `TestIssue2RollingApplyUnresolvedMetadata`. | Assertions check directory plus full filename titles including custom/archive/Unicode names, type project, allowed directory domain or general, medium/contextual, rolling true, empty tags/related and required provenance. Valid contributor/custom fields and metadata values survive; body bytes are identical. Invalid existing confidence/types/calendar values are rejected with original bytes preserved. |
| R4 | `rollingEntryProvenance` and shared missing-only additions followed by full schema/date validation. `TestIssue2ToolsRollingProvenanceRejectsExamples`, `TestIssue2ToolsVariantsNoReferences`, `TestIssue2ToolsVariantsRejectAndRestore`, `TestIssue2EnvelopeRepresentative`, `TestIssue2EnvelopeRejectsMetadata`, `TestIssue2RollingPreviewUnresolvedMetadata`, `TestIssue2RollingApplyRepresentative`, `TestIssue2RollingApplyUnresolvedMetadata`. | Newest-first entries yield created 2026-10-01 and updated 2026-11-01. Fenced examples, longer fence delimiters and indented closing-looking lines do not contaminate dates or sources. Explicit session references recover sources; none gives an empty list. Valid existing dates/sources remain intact. Undated or contradictory evidence leaves files unchanged/unresolved; no repair-day substitution occurs. |
| R5 | Early rolling-only dispatch in [`lint.go`](../../cmd/scribe/lint.go#L38), common preview/apply CLI path, existing selectors and legacy repair branch. `TestIssue2RollingPreviewRepresentative`, `TestIssue2RollingPreviewSelectors`, `TestIssue2RollingPreviewHelpAndOrdinaryLint`, `TestIssue2RollingPreviewFlagsAndBusy`, `TestIssue2RollingApplyRepresentative`, `TestIssue2RollingApplySelectors`. | Actual CLI main/Kong subprocesses accept both final commands; --rolling-only without --fix fails visibly. Preview reports proposals without content writes. Apply persists only selected rolling repairs. Neither enters duplicate cleanup: duplicate/placeholder/unrelated and processed-log bytes survive. Ordinary lint/repair still passes its existing regressions. Help, README, two tools prompts, four envelope prompts, KB template and embedded CLI guidance match delivered scope/default/provenance/recovery behavior. |
| R6 | Shared assessment/count loop in [`lint_rolling.go`](../../cmd/scribe/lint_rolling.go#L51), validated proposal before rechecked regular-path atomic replacement and exact-mode writer. `TestIssue2RollingApplyRepresentative`, `TestIssue2RollingApplyMixedAndStable`, `TestIssue2RollingApplyWriteFailure`, `TestIssue2RollingApplyPreservesMode`, `TestIssue2RollingApplyUnresolvedMetadata`, `TestIssue2RollingPreviewFullScope`. | Mixed batch scans 6, repairs 4, leaves 1 unchanged/1 unresolved, reports paths/causes and exits nonzero. Repeated apply repairs 0 with unchanged article bytes; following preview proposes 0. Child-only RLIMIT_FSIZE makes a real temporary write fail: the large original survives, a valid small sibling repairs, no temp leaks, 2/1/0/1 counts and nonzero. Body/valid/custom metadata and Unix modes 0600/0660 survive; replacement changes inode. Preview uses the same assessment and writes no KB content, including operational records. |
| R7 | Serial tools routing, parallel generation workers with collector-only Go apply/validate/restore/marker/checkpoint, and sequential Codex apply. `TestIssue2ToolsRollingSerializesWorkers`, `TestIssue2ToolsRollingCheckpoint`, `TestIssue2ToolsVariantsLargeCheckpoint`, `TestIssue2EnvelopeParallelIsolationCheckpoint`, `TestIssue2ToolsMarkerFailureRestoresBeforeCheckpoint`. | Configured parallel tools calls never overlap. Two primary envelope generations demonstrably overlap; while failed-session output exists, a completed sibling cannot apply or checkpoint. After rollback, a real Git checkpoint contains only valid successful sibling content and matching marker; failed output/marker is absent. Tools marker-error regressions prove no unmatched-marker checkpoint in either prompt family. |
| R8 | Shared snapshot/restore and fatal sentinel through batch/extraction/SyncCmd.Run, distinct Codex callback. `TestIssue2ToolsRollingRestoresAfterFailure`, `TestIssue2ToolsVariantsRejectAndRestore`, `TestIssue2ToolsMarkerFailureRestoresBeforeCheckpoint`, `TestIssue2EnvelopeMarkerFailureRestores`, `TestIssue2ToolsRollingRestoreFailureStopsSync`, `TestIssue2ToolsVariantsLargeRestoreFailureStopsSync`, `TestIssue2EnvelopeFatalStopsSync`. | Parent observes actual existing/new rolling mutation before releasing caught child/model/metadata/marker failures. Old bytes restore, new output is removed, sessions remain pending and damaged control logs stay intact. Normal/large tools and normal/large/Codex full sync faults prove failed restoration stops later mining, staging, checkpoints and final commits. Inventory/alias tests prevent unsafe recovery from touching alias destinations. Rollback remains rolling-only; no crash recovery is claimed. |
| R9 | [`runRollingOnly`](../../cmd/scribe/lint_rolling.go#L14), existing canonical sync lock identity and nonblocking flock owner reporting in [`lockfile.go`](../../cmd/scribe/lockfile.go#L24). `TestIssue2RollingPreviewLockOwnership`, `TestIssue2RollingPreviewFlagsAndBusy`, `TestIssue2RollingPreviewContainment`, `TestIssue2RollingApplyBusy`, representative successful CLI tests and existing lock regressions. | Both apply and preview refuse a held selected-KB sync lock with nonzero status and active sync/PID, preserving the complete KB tree and holder details. Unlabeled holders get truthful unavailable-owner fallback rather than stale details. Locks remain on the same inode, different KBs remain independent and root aliases contend. Available-lock paths proceed normally. Ordinary lint coordination remains unchanged. |

Completion limits: fixture-local subprocesses and provider responses, temporary KB/filesystem/Git and OS locks establish the named implementation behavior. No live provider reasoning, installed runtime, scheduler, deployment or production KB repair was tested or changed. Process-crash recovery, rollback of unrelated article output and supported nested-alias traversal remain excluded. File-mode tests do not claim ACL/xattr preservation. S3–S5 were later committed separately on 2026-11-08 after isolated candidate verification. No installation, push or publication was performed. S1/S2 local commits and all earlier gate outcomes remain separate historical records.

### Final post-review source identity

| Final file | SHA-256 |
| --- | --- |
| `cmd/scribe/lint.go` | `85becb0f664eaa3507315ccb7b94a9e50dea6583d595c6b9c2b5d110e8c4e06e` |
| `cmd/scribe/lint_rolling.go` | `f1b831805c012c4320a3eec5f3614ace04f74ef92e5901ef40a7916180c8606c` |
| `cmd/scribe/wiki_actions.go` | `4eefabf56f956c73064498792fbfba5966b96502bdd53a7f1023f3cff484403d` |
| `cmd/scribe/issue2_rolling_preview_test.go` | `098c9db6d85b254487748c519bf17934a05943a4d042cbc4fcc9107b1f20d49c` |
| `cmd/scribe/issue2_rolling_apply_test.go` | `66b54fcd45c1c5794353a9f82c933daa7c0bc99817fc66188fed82720871fe20` |
| `README.md` | `3b40ef003248d02ec697da75acfb4297072fd13a5da639fe3f01f050c065ea05` |
| `cmd/scribe/templates/kb-CLAUDE.md` | `3e7b2b6c1cd174d54f0dcbd406aeb850f761500cec421d56761c815b8389106f` |
| `cmd/scribe/skills/scribe-cli/SKILL.md` | `a055319e798540199c83d6865115f4e5e61f8c96006573dcc98efd65555f2eda` |
| `cmd/scribe/sync_sessions.go` | `790f74b103097baee093ab9489b03c38c0f2891b0f34a9c9d1921d5a78cae3bb` |
| `cmd/scribe/session_rolling.go` | `a0dca4f97cfe84b80efcf9822d383e3e5a195f8504e3d7b161f729e9dcc1dc16` |
| `cmd/scribe/issue2_tools_rolling_test.go` | `3bbe1f74b321e0e6968dafd95022cc384812408e0f3bd3414bc8d9883d9e8d7b` |
| `cmd/scribe/issue2_tools_marker_test.go` | `91460f1ae980eaf9779dff8d633526f79a695b5dd0042de21931c2ef7e08feee` |
| `cmd/scribe/sessions_test.go` | `f2a9597b85e672428f099e7473328711f49d4fd9f0874a3b8a7e233cc0562bb6` |
| `cmd/scribe/wiki_actions_meta_test.go` | `190ff2387048ed043c743f94391840ff65c231b62257ec719b175a68dee8333f` |
| `cmd/scribe/claude.go` | `32c4fe42f046ef385f904f05170f95d1b0d966d9b1973c001d29740091b2d859` |
