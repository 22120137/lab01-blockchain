package util

import (
	"fmt"
	"io"
	"sync"
	"time"
)

type Logger struct {
	out           io.Writer
	mu            sync.Mutex
	deterministic bool
	counter       uint64
}

func NewLogger(w io.Writer) *Logger { return &Logger{out: w} }

func NewDeterministicLogger(w io.Writer) *Logger {
	return &Logger{out: w, deterministic: true}
}

func (l *Logger) timestamp() string {
	if l.deterministic {
		ts := fmt.Sprintf("TICK%012d", l.counter)
		l.counter++
		return ts
	}
	return time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
}

func (l *Logger) Printf(format string, a ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	ts := l.timestamp()
	fmt.Fprintf(l.out, "%s|%s\n", ts, fmt.Sprintf(format, a...))
}
