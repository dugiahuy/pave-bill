package billing

import (
	"context"
	"time"

	"encore.dev/rlog"
)

// runAsync is an indirection over safeAsync so tests can override
// asynchronous behavior and execute operations synchronously.
// Production code uses safeAsync (goroutine) by default.
var runAsync = safeAsync

// safeAsync runs a function in a goroutine with a timeout and structured error logging.
// It prevents silent failures of background operations (signals, terminations, etc.).
func safeAsync(op string, fn func(ctx context.Context) error) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := fn(ctx); err != nil {
			rlog.Error("async operation failed", "op", op, "error", err)
		} else {
			rlog.Debug("async operation succeeded", "op", op)
		}
	}()
}
