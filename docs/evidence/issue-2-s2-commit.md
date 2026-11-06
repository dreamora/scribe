# Issue #2 S2 commit verification

Date: 2026-11-06. Status: isolated S2 commit candidate verified.

The user requested a commit after S2 passed its bounded independent gate. S3 was already in progress, so the commit excludes its unfinished source and contract. It also excludes unrelated provider, cap, triage, deep, and collision changes. The working files were preserved.

## Candidate origin

Parent: `335cb44a7c24656dddd8cd06ac007499f9b9c8ec` (verified S1).

The candidate starts from that commit. It applies only the S2 delta between the preserved pre-S2 and pre-S3 snapshots. The snapshots and file hashes are under `/private/tmp/scribe-issue2-s2-before-20261106` and `/private/tmp/scribe-issue2-s3-before-20261106`. The candidate and staging manifest are under `/private/tmp/scribe-s2-commit-20261106`.

Five of the seven product files exactly match the reviewed S2 target. `README.md` and `sync_sessions.go` exclude their pre-existing unrelated edits. The documentation index excludes the unrelated origin/upstream entry. The [S2 gate](issue-2.md#s2-gate-and-next-decision) reviewed the working target; the checks below establish the isolated commit candidate. These are separate evidence scopes.

## Executed checks

Working directory: `/private/tmp/scribe-s2-commit-20261106/tree`. All commands set `GOCACHE=/private/tmp/scribe-issue2-go-cache`, `GOPROXY=off`, `GOTOOLCHAIN=local`, and `GOWORK=off`. `go list -m -json` confirmed that this directory owns the main module.

```sh
go test -tags sqlite_fts5 -count=1 -timeout 2m -json ./cmd/scribe -run '^TestIssue2Tools(Rolling|Variants)'
go test -race -tags sqlite_fts5 -count=1 -timeout 2m -json ./cmd/scribe -run '^TestIssue2Tools(Rolling|Variants)|^TestMineSessionBatches_StopsAfterRateLimit$|^TestRecordSessionProcessed$|^TestApplyMetaRollingAppend_InitFrontmatterLintValid$'
make check
```

- Focused: 58 passing test outcomes, zero failures/skips; package 11.167 seconds.
- Race: 61 passing test outcomes, zero failures/skips; package 9.353 seconds.
- `make check`: exit 0; package tests passed in 30.405 seconds, followed by successful vet.

Raw results: `focused.jsonl`, `race.jsonl`, and `make-check.log` in the candidate scratch root. The full check used local-loopback permission for existing hermetic test servers. No installation, live KB change, authenticated model call, push, or publication ran.
