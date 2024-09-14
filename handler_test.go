package slogbuffer_test

import (
	"fmt"
	"github.com/delicb/slogbuffer"
	"log/slog"
	"testing"
)

func TestBufferLogHandler_Handle_Level(t *testing.T) {
	// given
	h := slogbuffer.NewBufferLogHandler(slog.LevelInfo)
	l := slog.New(h)

	l.Debug("discarded message")
	l.Info("info msg")
	l.Warn("warn")

	// when
	rh, reader := getSimplifiedTextHandler()
	setRealHandler(t, h, rh)
	lines := getLines(t, reader)

	// then
	expectLinesNo(t, lines, 2)

	expectLevel(t, lines[0], slog.LevelInfo)
	expectMsg(t, lines[0], "info msg")

	expectLevel(t, lines[1], slog.LevelWarn)
	expectMsg(t, lines[1], "warn")
}

func TestBufferLogHandler_recordWithAttributes(t *testing.T) {
	// given
	h := slogbuffer.NewBufferLogHandler(slog.LevelDebug)
	l := slog.New(h)

	l.Debug("debug msg", "my-level", "debug")
	l.Info("info msg", slog.String("my-level", "info"))

	// when
	rh, reader := getSimplifiedTextHandler()
	setRealHandler(t, h, rh)
	lines := getLines(t, reader)

	// then
	expectLinesNo(t, lines, 2)

	expectMsg(t, lines[0], "debug msg")
	expectLevel(t, lines[0], slog.LevelDebug)
	expectAttr(t, lines[0], "my-level", "debug")

	expectMsg(t, lines[1], "info msg")
	expectLevel(t, lines[1], slog.LevelInfo)
	expectAttr(t, lines[1], "my-level", "info")
}

func TestBufferLogHandler_WithAttrs(t *testing.T) {
	// given
	h := slogbuffer.NewBufferLogHandler(slog.LevelDebug)
	l := slog.New(h)

	commonAttrs := l.With("common", "attr")
	commonAttrs.With("sub-common", "sub-attr").Info("info msg", "rec-attr", "some value")
	commonAttrs.Warn("warn msg", slog.Int("some-int", 42))
	l.Error("error msg")

	// when
	rh, reader := getSimplifiedTextHandler()
	setRealHandler(t, h, rh)
	lines := getLines(t, reader)

	// then
	expectLinesNo(t, lines, 3)

	expectLevel(t, lines[0], slog.LevelInfo)
	expectMsg(t, lines[0], "info msg")
	expectAttr(t, lines[0], "common", "attr")
	expectAttr(t, lines[0], "sub-common", "sub-attr")
	expectAttr(t, lines[0], "rec-attr", "some value")

	expectLevel(t, lines[1], slog.LevelWarn)
	expectMsg(t, lines[1], "warn msg")
	expectAttr(t, lines[1], "some-int", "42")
	expectAttr(t, lines[1], "common", "attr")
	expectNoAttr(t, lines[1], "sub-common", "sub-attr")

	expectLevel(t, lines[2], slog.LevelError)
	expectMsg(t, lines[2], "error msg")
	expectNoAttr(t, lines[2], "common", "attr")
}

func TestBufferLogHandler_WithGroup(t *testing.T) {
	// given
	h := slogbuffer.NewBufferLogHandler(slog.LevelDebug)
	l := slog.New(h)
	l.WithGroup("g1").Info("info msg", "foo", "bar")
	l.WithGroup("g1").WithGroup("g2").Warn("warn msg", "foo", "bar")

	// when
	rh, reader := getSimplifiedTextHandler()
	setRealHandler(t, h, rh)
	lines := getLines(t, reader)

	// then
	expectLinesNo(t, lines, 2)

	expectLevel(t, lines[0], slog.LevelInfo)
	expectMsg(t, lines[0], "info msg")
	expectAttr(t, lines[0], "g1.foo", "bar")

	expectLevel(t, lines[1], slog.LevelWarn)
	expectMsg(t, lines[1], "warn msg")
	expectAttr(t, lines[1], "g1.g2.foo", "bar")

}

func TestBufferLogHandler_WithRealHandler(t *testing.T) {
	// given
	h := slogbuffer.NewBufferLogHandler(slog.LevelDebug)
	l := slog.New(h)
	rh, reader := getSimplifiedTextHandler()
	setRealHandler(t, h, rh)

	// when
	l.Info("info msg")
	l.With("common", "attr").Warn("warn msg")
	l.WithGroup("g1").Error("error msg", "in-group", "group")

	// then
	lines := getLines(t, reader)

	expectLinesNo(t, lines, 3)

	expectLevel(t, lines[0], slog.LevelInfo)
	expectMsg(t, lines[0], "info msg")

	expectLevel(t, lines[1], slog.LevelWarn)
	expectMsg(t, lines[1], "warn msg")
	expectAttr(t, lines[1], "common", "attr")

	expectLevel(t, lines[2], slog.LevelError)
	expectMsg(t, lines[2], "error msg")
	expectAttr(t, lines[2], "g1.in-group", "group")
}

func TestBufferLogHandler_LimitedBuffer(t *testing.T) {
	// given
	h := slogbuffer.NewBoundBufferLogHandler(slog.LevelDebug, 3)
	l := slog.New(h)

	// adding 5 messages (from 0 to 4), while having capcity only for 3
	for i := range 5 {
		l.Info("msg", slog.Int("no", i))
	}

	// when
	rh, reader := getSimplifiedTextHandler()
	setRealHandler(t, h, rh)
	lines := getLines(t, reader)

	// then
	expectLinesNo(t, lines, 3) // only buffered lines, oldest are dropped

	for i := range 3 {
		line := lines[i]
		// same level and msg for all records
		expectLevel(t, line, slog.LevelInfo)
		expectMsg(t, line, "msg")

		// different attr, expecting 0 and 1 to be dropped since they are logged first
		// so adding +2 to loop variable (i) to get expected value
		expectAttr(t, line, "no", fmt.Sprintf("%d", i+2))
	}
}

func TestBufferLogHandler_Discard(t *testing.T) {
	// given
	h := slogbuffer.NewBufferLogHandler(slog.LevelDebug)
	l := slog.New(h)

	l.Info("info msg")

	h.Discard()

	// when
	rh, reader := getSimplifiedTextHandler()
	setRealHandler(t, h, rh)
	lines := getLines(t, reader)

	// then
	expectLinesNo(t, lines, 0)
}

func TestBufferLogHandler_AfterSetRealHandler(t *testing.T) {
	// given
	h := slogbuffer.NewBufferLogHandler(slog.LevelDebug)
	l := slog.New(h)
	withAttr := l.With("common", "attr")

	rh, reader := getSimplifiedTextHandler()
	setRealHandler(t, h, rh)

	// when
	withAttr.Info("info msg")

	// then
	lines := getLines(t, reader)

	expectLinesNo(t, lines, 1)
	expectLevel(t, lines[0], slog.LevelInfo)
	expectMsg(t, lines[0], "info msg")
	expectAttr(t, lines[0], "common", "attr")
}
