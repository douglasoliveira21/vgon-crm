// Package safe runs background work without letting it take the server down.
package safe

import (
	"log"
	"runtime/debug"
)

// Go runs fn in a new goroutine with a recover.
//
// Go's runtime kills the whole process on a panic in any goroutine, and
// Fiber's recover middleware only covers request handlers — so a single bad
// WhatsApp payload or a malformed bot flow running in the background took the
// CRM offline for every tenant at once. Here the panicking task dies alone
// and leaves a stack trace behind.
//
// name identifies the task in the log, e.g. "webhook".
func Go(name string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[PANIC] %s: %v\n%s", name, r, debug.Stack())
			}
		}()
		fn()
	}()
}
