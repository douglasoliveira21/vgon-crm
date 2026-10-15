package safe

import (
	"testing"
	"time"
)

func TestGoRecoversPanicAndKeepsProcessAlive(t *testing.T) {
	panicked := make(chan struct{})
	Go("test-panic", func() {
		defer close(panicked)
		panic("boom")
	})

	select {
	case <-panicked:
	case <-time.After(2 * time.Second):
		t.Fatal("the task never ran")
	}

	// Reaching here at all proves the panic did not take the process down;
	// a second task confirms new work still runs afterwards.
	ran := make(chan struct{})
	Go("test-after", func() { close(ran) })
	select {
	case <-ran:
	case <-time.After(2 * time.Second):
		t.Fatal("work started after a recovered panic did not run")
	}
}
