package slogbuffer_test

import (
	"bytes"
	"context"
	"fmt"
	"github.com/delicb/slogbuffer"
	"io"
	"log/slog"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func getSimplifiedTextHandler() (slog.Handler, io.Reader) {
	writer := new(bytes.Buffer)
	return slog.NewTextHandler(writer, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			// remove time
			if attr.Key == "time" {
				return slog.Attr{}
			}
			return attr
		},
	}), writer
}

func filterOut[T any](s []T, f func(T) bool) []T {
	res := make([]T, 0, len(s))
	for _, el := range s {
		if f(el) {
			res = append(res, el)
		}
	}
	return slices.Clip(res)
}

func getLines(t *testing.T, r io.Reader) []string {
	t.Helper()
	all, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading all lines: %v", err)
	}

	return filterOut(
		strings.Split(strings.TrimSpace(string(all)), "\n"),
		func(s string) bool { return s != "" },
	)
}

func setRealHandler(t *testing.T, h *slogbuffer.BufferLogHandler, real slog.Handler) {
	t.Helper()
	if err := h.SetRealHandler(context.Background(), real); err != nil {
		t.Fatalf("setting real handler: %v", err)
	}
}

func expectLinesNo(t *testing.T, lines []string, no int) {
	t.Helper()
	if len(lines) != no {
		t.Fatalf("expect %d lines, got %d", no, len(lines))
	}
}

func expectLevel(t *testing.T, line string, level slog.Level) {
	t.Helper()
	if !strings.Contains(line, fmt.Sprintf("level=%s", level.String())) {
		t.Fatalf("expected level %s, line is %s", level.String(), line)
	}
}

var hasSpaceRe = regexp.MustCompile(`\s`)

func maybeQuoteVal(key string, val string) string {
	if hasSpaceRe.MatchString(val) {
		return fmt.Sprintf("%s=%q", key, val)
	}
	return fmt.Sprintf("%s=%s", key, val)
}

func expectMsg(t *testing.T, line string, msg string) {
	t.Helper()
	if !strings.Contains(line, maybeQuoteVal("msg", msg)) {
		t.Fatalf("expected msg %s, line is %s", msg, line)
	}
}

func expectAttr(t *testing.T, line string, key string, value string) {
	t.Helper()
	expectedAttr := maybeQuoteVal(key, value)
	if !strings.Contains(line, expectedAttr) {
		t.Fatalf("expected attribute %s, line is %s", expectedAttr, line)
	}
}

func expectNoAttr(t *testing.T, line string, key string, value string) {
	t.Helper()
	unexpectedAttr := maybeQuoteVal(key, value)
	if strings.Contains(line, unexpectedAttr) {
		t.Fatalf("unexpected attribute %s, line is %s", unexpectedAttr, line)
	}
}
