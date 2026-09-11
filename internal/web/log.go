package web

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Level is a logback log level. The demos configure only
// `logging.level.springfox.documentation`, so a per-logger-name threshold with
// an application-wide default is all that is needed.
type Level int

// The levels the original can be configured with.
const (
	LevelTrace Level = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
)

var levelNames = map[Level]string{
	LevelTrace: "TRACE",
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERROR",
}

// ParseLevel converts a configured level name, defaulting to INFO.
func ParseLevel(s string) Level {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "TRACE":
		return LevelTrace
	case "DEBUG":
		return LevelDebug
	case "WARN":
		return LevelWarn
	case "ERROR":
		return LevelError
	default:
		return LevelInfo
	}
}

// Logger routes messages through per-name thresholds, reproducing logback's
// hierarchical `logging.level.<name>` configuration.
type Logger struct {
	mu       sync.Mutex
	out      io.Writer
	root     Level
	byLogger map[string]Level
}

// NewLogger returns a logger writing to stdout with the Spring Boot default
// root level of INFO.
func NewLogger() *Logger {
	return &Logger{out: os.Stdout, root: LevelInfo, byLogger: map[string]Level{}}
}

// SetOutput redirects the logger.
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.out = w
}

// SetLevel applies `logging.level.<name>=<level>`.
func (l *Logger) SetLevel(name string, level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.byLogger[name] = level
}

// LevelFor resolves the effective level of a logger name, walking up the
// dot-separated hierarchy exactly as logback does.
func (l *Logger) LevelFor(name string) Level {
	l.mu.Lock()
	defer l.mu.Unlock()
	for n := name; ; {
		if lvl, ok := l.byLogger[n]; ok {
			return lvl
		}
		i := strings.LastIndex(n, ".")
		if i < 0 {
			break
		}
		n = n[:i]
	}
	return l.root
}

// Enabled reports whether the named logger would emit at the given level.
func (l *Logger) Enabled(name string, level Level) bool { return level >= l.LevelFor(name) }

// Log writes one line for the named logger if its level allows it.
func (l *Logger) Log(name string, level Level, format string, args ...any) {
	if !l.Enabled(name, level) {
		return
	}
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("%s %-5s --- [%s] : %s\n",
		time.Now().Format("2006-01-02 15:04:05.000"), levelNames[level], name, msg)
	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = io.WriteString(l.out, line)
}

// Infof logs at INFO under the application's own logger name.
func (l *Logger) Infof(format string, args ...any) {
	l.Log("springfoxdemo", LevelInfo, format, args...)
}

// Debugf logs at DEBUG under the given logger name.
func (l *Logger) Debugf(name, format string, args ...any) {
	l.Log(name, LevelDebug, format, args...)
}
