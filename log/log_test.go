package log

import (
	"bytes"
	stdlog "log"
	"strings"
	"testing"
)

func TestLevelInfoEmitsInfoWarnError(t *testing.T) {
	previousLevel := Level(globalLevel.Load())
	t.Cleanup(func() { SetLevel(previousLevel) })

	logger := NewLogger("test")
	SetLevel(LevelInfo)

	output := captureLogOutput(
		t, func() {
			logger.Debug("debug-message")
			logger.Info("info-message")
			logger.Warn("warn-message")
			logger.Error("error-message")
		},
	)

	if strings.Contains(output, "debug-message") {
		t.Fatalf("expected debug to be suppressed at LevelInfo, got %q", output)
	}
	if !strings.Contains(output, "info-message") {
		t.Fatalf("expected info output, got %q", output)
	}
	if !strings.Contains(output, "warn-message") {
		t.Fatalf("expected warning output, got %q", output)
	}
	if !strings.Contains(output, "error-message") {
		t.Fatalf("expected error output, got %q", output)
	}
}

func TestLevelNoneSuppressesAllLogs(t *testing.T) {
	previousLevel := Level(globalLevel.Load())
	t.Cleanup(func() { SetLevel(previousLevel) })

	logger := NewLogger("test")
	SetLevel(LevelNone)

	output := captureLogOutput(
		t, func() {
			logger.Info("info-message")
			logger.Warn("warn-message")
			logger.Error("error-message")
		},
	)

	if output != "" {
		t.Fatalf("expected no output at LevelNone, got %q", output)
	}
}

func TestAliasesRemainCompatible(t *testing.T) {
	if Debug != LevelDebug {
		t.Fatalf("Debug alias mismatch: got %v want %v", Debug, LevelDebug)
	}
	if Info != LevelInfo {
		t.Fatalf("Info alias mismatch: got %v want %v", Info, LevelInfo)
	}
	if Warn != LevelWarning {
		t.Fatalf("Warn alias mismatch: got %v want %v", Warn, LevelWarning)
	}
	if Error != LevelError {
		t.Fatalf("Error alias mismatch: got %v want %v", Error, LevelError)
	}
	if Off != LevelNone {
		t.Fatalf("Off alias mismatch: got %v want %v", Off, LevelNone)
	}
}

func captureLogOutput(t *testing.T, fn func()) string {
	t.Helper()

	prevOut := stdlog.Writer()
	prevFlags := stdlog.Flags()
	prevPrefix := stdlog.Prefix()

	var out bytes.Buffer
	stdlog.SetOutput(&out)
	stdlog.SetFlags(0)
	stdlog.SetPrefix("")
	t.Cleanup(
		func() {
			stdlog.SetOutput(prevOut)
			stdlog.SetFlags(prevFlags)
			stdlog.SetPrefix(prevPrefix)
		},
	)

	fn()

	return out.String()
}
