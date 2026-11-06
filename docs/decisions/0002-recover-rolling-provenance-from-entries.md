# Recover rolling-file provenance from entries

Repair preserves valid existing metadata and recovers missing dates from dated entries and missing sources from explicit `session:<id>` references. It reports unresolved dates instead of assigning the repair date as an unsupported historical date. This preserves provenance; assigning current dates would satisfy validation but could misrepresent when the recorded knowledge originated.

For missing dates, repair uses the earliest valid entry date for `created` and the latest for `updated`, regardless of entry order. Dates inside code blocks are not evidence. Repair reports unresolved or contradictory dates without changing that file; no explicit session references produces an empty recovered source list.
