// Package bolt test helpers

package bolt

import (
	"bytes"
	"sync"
)

// ThreadSafeBuffer is a thread-safe wrapper around bytes.Buffer
// Used in tests to avoid false positive race conditions in test infrastructure
type ThreadSafeBuffer struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

func (tsb *ThreadSafeBuffer) Write(p []byte) (n int, err error) {
	tsb.mu.Lock()
	defer tsb.mu.Unlock()
	return tsb.buf.Write(p)
}

func (tsb *ThreadSafeBuffer) Bytes() []byte {
	tsb.mu.Lock()
	defer tsb.mu.Unlock()
	return tsb.buf.Bytes()
}

func (tsb *ThreadSafeBuffer) String() string {
	tsb.mu.Lock()
	defer tsb.mu.Unlock()
	return tsb.buf.String()
}
