package util

import (
	"bytes"
	"strings"
	"sync"
	"testing"
)

// Test that logger writes a line that contains formatted message
func TestLoggerWritesTimestampedLine(t *testing.T) {
	var buf bytes.Buffer
	l := NewLogger(&buf)

	l.Printf("hello %d", 123)

	out := buf.String()
	if !strings.Contains(out, "hello 123") {
		t.Fatalf("expected log to contain message; got: %q", out)
	}
	// quick sanity: expect at least one '|' separator (our logger prints ts|msg)
	if !strings.Contains(out, "|") {
		t.Fatalf("expected timestamp separator '|' in log, got: %q", out)
	}
}

// Test logger under concurrent use (thread-safety)
func TestLoggerConcurrency(t *testing.T) {
	var buf bytes.Buffer
	l := NewLogger(&buf)

	const N = 100
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func(i int) {
			defer wg.Done()
			l.Printf("x=%d", i)
		}(i)
	}
	wg.Wait()

	out := buf.String()
	// count newline occurrences (each Printf writes one line ending with '\n')
	lines := strings.Count(out, "\n")
	if lines != N {
		t.Fatalf("expected %d log lines, got %d, full output:\n%s", N, lines, out)
	}
}
