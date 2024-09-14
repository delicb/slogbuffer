package slogbuffer_test

import (
	"context"
	"github.com/delicb/slogbuffer"
	"log/slog"
	"os"
)

func ExampleBufferLogHandler() {
	// buffer log message up to 256 messages, drop oldest one is more messages are logged
	h := slogbuffer.NewBoundBufferLogHandler(slog.LevelInfo, 256)
	log := slog.New(h)

	// use logger normally
	log.Info("some message", slog.Int("key", 11))
	log.Debug("discarded message")
	log.Warn("warn message")

	// read config, config file, whatever to figure out how real handler looks like
	realHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey { // remove time to have constant output
				return slog.Attr{}
			}
			return attr
		},
	})

	// after setup is complete, we know the config of real handler
	err := h.SetRealHandler(context.Background(), realHandler)
	if err != nil {
		panic("setting real handler")
	}

	// from this point on, log records are not buffered, they are just passed to real handler
	log.Info("direct logging")

	// output:
	// level=INFO msg="some message" key=11
	// level=WARN msg="warn message"
	// level=INFO msg="direct logging"
}
