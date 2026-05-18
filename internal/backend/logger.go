package backend

import (
	"fmt"
	"os"
	"sync"
)

type Logger struct {
	mu   sync.Mutex
	file *os.File
}

func NewLogger(path string) (*Logger, error) {
	if err := ensureDir(filepathDir(path)); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}

	return &Logger{file: file}, nil
}

func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}

func (l *Logger) Write(entry LogEntry) error {
	if l == nil || l.file == nil {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	_, err := fmt.Fprintf(l.file, "%s | %s | %s | %s\n", entry.Timestamp, entry.Kind, entry.Level, entry.Message)
	return err
}

func filepathDir(path string) string {
	last := len(path) - 1
	for last >= 0 && (path[last] == '\\' || path[last] == '/') {
		last--
	}
	if last < 0 {
		return ""
	}
	for i := last; i >= 0; i-- {
		if path[i] == '\\' || path[i] == '/' {
			return path[:i]
		}
	}
	return ""
}
