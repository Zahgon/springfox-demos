package web

import (
	"bytes"
	"strings"
	"testing"
)

// TestLevelResolutionWalksTheHierarchy covers logback's `logging.level.<name>`
// semantics, which the demos configure with
// `logging.level.springfox.documentation=DEBUG`: a threshold set on a parent
// logger applies to every logger beneath it, and the root default is INFO.
func TestLevelResolutionWalksTheHierarchy(t *testing.T) {
	l := NewLogger()
	l.SetLevel("springfox.documentation", LevelDebug)

	cases := []struct {
		logger string
		want   Level
	}{
		{"springfox.documentation", LevelDebug},
		{"springfox.documentation.spring.web", LevelDebug},
		{"springfox", LevelInfo},
		{"springfox.other", LevelInfo},
		{"unrelated", LevelInfo},
	}
	for _, c := range cases {
		if got := l.LevelFor(c.logger); got != c.want {
			t.Errorf("LevelFor(%q) = %v, want %v", c.logger, got, c.want)
		}
	}
	if !l.Enabled("springfox.documentation", LevelDebug) {
		t.Error("DEBUG is not enabled on the configured logger")
	}
	if l.Enabled("springfox", LevelDebug) {
		t.Error("DEBUG is enabled on a logger that was never configured")
	}
}

// TestLogWritesOnlyWhatTheLevelAdmits covers the filtering and the line format.
func TestLogWritesOnlyWhatTheLevelAdmits(t *testing.T) {
	var buf bytes.Buffer
	l := NewLogger()
	l.SetOutput(&buf)
	l.SetLevel("springfox.documentation", LevelDebug)

	l.Debugf("springfox.documentation", "scanning %d handlers", 7)
	l.Debugf("springfox", "this one is below the root level")
	l.Infof("started on port %d", 8080)

	out := buf.String()
	if !strings.Contains(out, "DEBUG") || !strings.Contains(out, "scanning 7 handlers") {
		t.Errorf("the DEBUG line is missing:\n%s", out)
	}
	if strings.Contains(out, "this one is below the root level") {
		t.Errorf("a line below the threshold was written:\n%s", out)
	}
	if !strings.Contains(out, "INFO") || !strings.Contains(out, "started on port 8080") {
		t.Errorf("the INFO line is missing:\n%s", out)
	}
	if lines := strings.Count(strings.TrimSpace(out), "\n") + 1; lines != 2 {
		t.Errorf("wrote %d lines, want 2:\n%s", lines, out)
	}
}

// TestParseLevel covers the configured level names, including the fallback to
// Spring Boot's INFO default for anything unrecognised.
func TestParseLevel(t *testing.T) {
	cases := map[string]Level{
		"TRACE": LevelTrace, "debug": LevelDebug, "INFO": LevelInfo,
		"warn": LevelWarn, "ERROR": LevelError, " Debug ": LevelDebug,
		"": LevelInfo, "nonsense": LevelInfo,
	}
	for in, want := range cases {
		if got := ParseLevel(in); got != want {
			t.Errorf("ParseLevel(%q) = %v, want %v", in, got, want)
		}
	}
}
