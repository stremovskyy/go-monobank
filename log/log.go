package log

import (
	"fmt"
	stdlog "log"
	"strings"
	"sync/atomic"
)

// Level controls which logs are emitted.
//
// The ordering is: None < Error < Warning < Info < Debug < All.
// Any message above the configured level is ignored.
type Level int32

const (
	LevelNone Level = iota // disables all logging
	LevelError
	LevelWarning
	LevelInfo
	LevelDebug
	LevelAll // enables all logging
)

const (
	// Backward-compatible aliases.
	Error Level = LevelError
	Warn  Level = LevelWarning
	Info  Level = LevelInfo
	Debug Level = LevelDebug
	Off   Level = LevelNone
)

var globalLevel atomic.Int32

func init() {
	globalLevel.Store(int32(LevelNone))
}

// SetLevel changes global log level.
func SetLevel(level Level) {
	if level < LevelNone || level > LevelAll {
		level = LevelNone
	}
	globalLevel.Store(int32(level))
}

func levelEnabled(level Level) bool {
	current := Level(globalLevel.Load())
	switch current {
	case LevelNone:
		return false
	case LevelAll:
		return true
	default:
		if level < LevelError || level > LevelDebug {
			return false
		}
		return level <= current
	}
}

// Logger is a tiny wrapper around the standard logger.
// It exists mostly to keep the style consistent with other SDKs.
type Logger struct {
	prefix string
}

func NewLogger(prefix string) *Logger {
	return &Logger{prefix: prefix}
}

func (l *Logger) log(level Level, label string, format string, args ...any) {
	if !levelEnabled(level) {
		return
	}

	message := fmt.Sprintf(format, args...)
	prefix := ""
	if l != nil {
		prefix = strings.TrimSpace(l.prefix)
	}
	if prefix == "" {
		stdlog.Printf("[%s] %s", label, message)
		return
	}
	stdlog.Printf("[%s] %s %s", label, prefix, message)
}

func (l *Logger) Debug(format string, args ...any) {
	l.log(LevelDebug, "DEBUG", format, args...)
}

func (l *Logger) Info(format string, args ...any) {
	l.log(LevelInfo, "INFO", format, args...)
}

func (l *Logger) Warn(format string, args ...any) {
	l.log(LevelWarning, "WARN", format, args...)
}

func (l *Logger) Warning(format string, args ...any) {
	l.Warn(format, args...)
}

func (l *Logger) Error(format string, args ...any) {
	l.log(LevelError, "ERROR", format, args...)
}

// All is an alias for Debug verbosity.
func (l *Logger) All(format string, args ...any) {
	l.Debug(format, args...)
}
