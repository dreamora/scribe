# Issue #2: slice commit verification

Date: 2026-11-08. The user requested one local commit for each completed slice. S1 and S2 were already committed as `335cb44` and `eb48423`.

## Method and boundary

Each candidate starts from its committed parent and applies only the next slice delta from preserved pre-slice snapshots. The checkout contains unrelated provider, deep, triage and recovery work; those changes remain outside these commits. Working source bytes are preserved while exact candidate blobs are staged.

S3 requires the existing collision executor and its two safety tests from the preserved baseline. These supporting files are included with S3. Other collision wiring in extraction/deep paths and unrelated generic sanitization changes remain excluded. The isolated writer retains the committed ordinary-article sanitizer while applying S3's rolling metadata policy.

The earlier independent gates reviewed the recorded working targets. The checks below verify the isolated commit candidates separately. No push, install, live KB repair or authenticated model call is part of this operation.

## S3: protected envelope application

Parent: `eb48423`. Candidate scratch: `/private/tmp/scribe-slice-commits-20261108/s3/tree`. The S3 delta uses the preserved pre-S3 and pre-S4 snapshots.

- Focused: 114 passing named outcomes, zero failures/skips; package 15.608 seconds.
- Targeted race: 117 passing named outcomes, zero failures/skips; package 16.106 seconds.
- `make check`: exit 0; full package tests 36.049 seconds, then successful vet.

All checks set `GOCACHE=/private/tmp/scribe-issue2-go-cache`, `GOPROXY=off`, `GOTOOLCHAIN=local`, and `GOWORK=off`. Go commands use `sqlite_fts5`; race adds `-race`. Focused selection includes S1/S2 tools tests, S3 envelope tests and the required collision safety tests. Race adds rate-limit, processed-marker and rolling-initialization regressions. Raw JSONL, full-check log and staging manifest are stored beside the candidate tree. Early scratch merge setup errors are not counted as product failures or successful checks.

## S4: rolling-only repair preview

S3 commit: `803fce1`. S4 parent: `803fce1`. Candidate scratch: `/private/tmp/scribe-slice-commits-20261108/s4/tree`. The S4 delta uses the preserved pre-S4 and pre-S5 snapshots; the intermediate apply refusal remains in this commit.

- Focused: 141 passing named outcomes, zero failures/skips; package 11.199 seconds.
- Targeted race: 182 passing named outcomes, zero failures/skips; package 24.786 seconds.
- `make check`: exit 0; full package tests 38.681 seconds, then successful vet.

The focused selection covers all issue #2 fixtures and collision safety. Race also selects ordinary lint, lock identity/acquisition, rate-limit, marker and initialization regressions. The environment and raw artifact layout follow S3 above.
