// Package logger provides structured logging for kallm.
package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Level represents a log level.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger is a structured logger.
type Logger struct {
	mu       sync.Mutex
	out      io.Writer
	level    Level
	jsonMode bool
}

// New creates a new logger.
func New(jsonMode bool) *Logger {
	return &Logger{
		out:      os.Stdout,
		level:    LevelDebug,
		jsonMode: jsonMode,
	}
}

// log writes a log entry.
func (l *Logger) log(level Level, msg string, keyvals ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.jsonMode {
		l.logJSON(level, msg, keyvals...)
	} else {
		l.logText(level, msg, keyvals...)
	}
}

func (l *Logger) logJSON(level Level, msg string, keyvals ...interface{}) {
	entry := map[string]interface{}{
		"time":  time.Now().Format(time.RFC3339),
		"level": level.String(),
		"msg":   msg,
	}

	for i := 0; i < len(keyvals)-1; i += 2 {
		key, ok := keyvals[i].(string)
		if !ok {
			continue
		}
		entry[key] = keyvals[i+1]
	}

	data, _ := json.Marshal(entry)
	fmt.Fprintln(l.out, string(data))
}

func (l *Logger) logText(level Level, msg string, keyvals ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(l.out, "%s %s %s", timestamp, level.String(), msg)

	for i := 0; i < len(keyvals)-1; i += 2 {
		fmt.Fprintf(l.out, " %v=%v", keyvals[i], keyvals[i+1])
	}
	fmt.Fprintln(l.out)
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string, keyvals ...interface{}) {
	l.log(LevelDebug, msg, keyvals...)
}

// Info logs an info message.
func (l *Logger) Info(msg string, keyvals ...interface{}) {
	l.log(LevelInfo, msg, keyvals...)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, keyvals ...interface{}) {
	l.log(LevelWarn, msg, keyvals...)
}

// Error logs an error message.
func (l *Logger) Error(msg string, keyvals ...interface{}) {
	l.log(LevelError, msg, keyvals...)
}
