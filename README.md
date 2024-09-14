# slogbuffer

`slogbuffer` implements `slog.Handler` that buffers log records until real handler is available. 

## Use cases
This handler is useful in multiple situations. Some of them are:

* Application wants to dynamically create handler based on config file (e.g. config file 
  defines where logs should go, which level should be logged, etc). However, in order to 
  load config file some code has to be executed and it might produce logs. These logs can be
  buffered using `slogbuffer` until real handler is known. 
* Application might want to remove some attributes or change keys for attributes using 
  `ReplaceAttrs` option for stdlib handlers, but it might want to do it dynamically, based on
  external documentation. Until that config is known, logs can be buffered. 
* CLI applications might allow logging config via command line flags. Parsing flags might
  produce log lines, so buffering those might be useful.
* Application might want to produce logs to file or stderr only in case of error or panic. 
  However, once panic happens not all context might be available. Using `slogbuffer` 
  allows logging normally and flushing all accumulated logs (or only subset of latest
  log records) once condition is met. 
* Of course, this is useful in any subset of program, not just entire application. E.g. 
  HTTP handler can accept logger and log stuff, but level above (e.g. middleware) might 
  choose to flush those records only if non-200 status code is returned.
* Streaming logs over network in case when log sink still not ready is use case when buffering
  can se be useful with flushing all records once sink becomes available. 

## Usage
In order to use this handler, just create new `slog.Logger` using `slog.New` and provide
instance of `slogbuffer.BufferLogHandler`. `BufferLogHandler` can be created in two ways, using:
* `NewBufferLogHandler(slog.Level)` - creates unbound buffer, all log records will be stored. This
  is useful, but in case where a lot of lot messages can be produces can consume too much memory.
* `NewBoundBufferLogHandler(slog.Level, maxRecords int)` creates bound buffer. It can store at
  most `maxRecords` of log records. When new ones are created, oldest ones added are removed.

After real handler is known and created, `SetRealHandler(context.Context, slog.Handler)` method
should be called. At this point, all buffered log records are flushed to provided real logger
and from that point on `BufferLogHandler` behaves as simple proxy to real handler, which means
that any logger that already has instance of `BufferLogHandler` will continue working as if real
handler was used from the start.

## Contribution
While this was created to scratch personal itch (CLI application that allows user to configure
logging), contributions are welcome via PRs. 

## Author(s)
* Bojan Delić <bojan@delic.in.rs>
