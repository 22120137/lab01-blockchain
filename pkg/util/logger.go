package util

import (
	"fmt"
	"io"
	"sync"
	"time"
)

type Logger struct {
	out io.Writer
	mu  sync.Mutex
}

func NewLogger(w io.Writer) *Logger { return &Logger{out: w} }

func (l *Logger) Printf(format string, a ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	ts := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	fmt.Fprintf(l.out, "%s|%s\n", ts, fmt.Sprintf(format, a...))
}
