package logging

import (
	"fmt"
	"io"
	"log/syslog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type (
	Color int
	Level int
)

// Colors for different log levels.
const (
	BLACK Color = (iota + 30)
	RED
	GREEN
	YELLOW
	BLUE
	MAGENTA
	CYAN
	WHITE
)

// Logging levels.
const (
	CRITICAL Level = iota
	ERROR
	WARNING
	NOTICE
	INFO
	DEBUG
)

var LevelNames = map[Level]string{
	CRITICAL: "CRITICAL",
	ERROR:    "ERROR",
	WARNING:  "WARNING",
	NOTICE:   "NOTICE",
	INFO:     "INFO",
	DEBUG:    "DEBUG",
}

var LevelColors = map[Level]Color{
	CRITICAL: MAGENTA,
	ERROR:    RED,
	WARNING:  YELLOW,
	NOTICE:   GREEN,
	INFO:     WHITE,
	DEBUG:    CYAN,
}

var (
	DefaultLogger    = NewLogger(procName())
	DefaultLevel     = INFO
	DefaultHandler   = StderrHandler
	DefaultFormatter = &defaultFormatter{}
	StdoutHandler    = NewWriterHandler(os.Stdout)
	StderrHandler    = NewWriterHandler(os.Stderr)
)

func init() {
	StdoutHandler.Colorize = true
	StderrHandler.Colorize = true
}

// Logger is the interface for outputing log messages in different levels.
// A new Logger can be created with NewLogger() function.
// You can changed the output handler with SetHandler() function.
type Logger interface {
	// SetLevel changes the level of the logger. Default is logging.Info.
	SetLevel(Level)

	// SetHandler replaces the current handler for output. Default is logging.StderrHandler.
	SetHandler(Handler)

	// Fatal is equivalent to l.Critical followed by a call to os.Exit(1).
	Fatal(format string, args ...interface{})

	// Panic is equivalent to l.Critical followed by a call to panic().
	Panic(format string, args ...interface{})

	// Critical logs a message using CRITICAL as log level.
	Critical(format string, args ...interface{})

	// Error logs a message using ERROR as log level.
	Error(format string, args ...interface{})

	// Warning logs a message using WARNING as log level.
	Warning(format string, args ...interface{})

	// Notice logs a message using NOTICE as log level.
	Notice(format string, args ...interface{})

	// Info logs a message using INFO as log level.
	Info(format string, args ...interface{})

	// Debug logs a message using DEBUG as log level.
	Debug(format string, args ...interface{})
}

// Handler handles the output.
type Handler interface {
	SetFormatter(Formatter)
	SetLevel(Level)

	// Handle single log record.
	Handle(*Record)

	// Close the handler.
	Close()
}

// Record contains all of the information about a single log message.
type Record struct {
	Format      string
	Args        []interface{}
	LoggerName  string
	Level       Level
	Time        time.Time
	Filename    string
	Line        int
	ProcessID   int
	ProcessName string
}

// Formatter formats a record.
type Formatter interface {
	// Format the record and return a message.
	Format(*Record) (message string)
}

///////////////////////
//                   //
// Default Formatter //
//                   //
///////////////////////

type defaultFormatter struct{}

// Format outputs a message like "2014-02-28 18:15:57 [example] INFO     something happened"
func (f *defaultFormatter) Format(rec *Record) string {
	return fmt.Sprintf("%s [%s] %-8s %s", fmt.Sprint(rec.Time)[:19], rec.LoggerName, LevelNames[rec.Level], fmt.Sprintf(rec.Format, rec.Args...))
}

///////////////////////////
//                       //
// Logger implementation //
//                       //
///////////////////////////

// logger is the default Logger implementation.
type logger struct {
	Name    string
	Level   Level
	Handler Handler
}

// NewLogger returns a new Logger implementation. Do not forget to close it at exit.
func NewLogger(name string) Logger {
	return &logger{
		Name:    name,
		Level:   DefaultLevel,
		Handler: DefaultHandler,
	}
}

func (l *logger) Close() {
	l.Handler.Close()
}

func (l *logger) SetLevel(level Level) {
	l.Level = level
}

func (l *logger) SetHandler(b Handler) {
	l.Handler = b
}

func (l *logger) Fatal(format string, args ...interface{}) {
	l.Critical(format, args...)
	l.Close()
	os.Exit(1)
}

func (l *logger) Panic(format string, args ...interface{}) {
	l.Critical(format, args...)
	l.Close()
	panic(fmt.Sprintf(format, args...))
}

func (l *logger) Critical(format string, args ...interface{}) {
	if l.Level >= CRITICAL {
		l.log(CRITICAL, format, args...)
	}
}

func (l *logger) Error(format string, args ...interface{}) {
	if l.Level >= ERROR {
		l.log(ERROR, format, args...)
	}
}

func (l *logger) Warning(format string, args ...interface{}) {
	if l.Level >= WARNING {
		l.log(WARNING, format, args...)
	}
}

func (l *logger) Notice(format string, args ...interface{}) {
	if l.Level >= NOTICE {
		l.log(NOTICE, format, args...)
	}
}

func (l *logger) Info(format string, args ...interface{}) {
	if l.Level >= INFO {
		l.log(INFO, format, args...)
	}
}

func (l *logger) Debug(format string, args ...interface{}) {
	if l.Level >= DEBUG {
		l.log(DEBUG, format, args...)
	}
}

func (l *logger) log(level Level, format string, args ...interface{}) {
	// Add missing newline at the end.
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}

	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "???"
		line = 0
	}

	rec := &Record{
		Format:      format,
		Args:        args,
		LoggerName:  l.Name,
		Level:       level,
		Time:        time.Now(),
		Filename:    file,
		Line:        line,
		ProcessName: procName(),
		ProcessID:   os.Getpid(),
	}

	l.Handler.Handle(rec)
}

// procName returns the name of the current process.
func procName() string { return filepath.Base(os.Args[0]) }

///////////////////
//               //
// DefaultLogger //
//               //
///////////////////

func Fatal(format string, args ...interface{}) {
	DefaultLogger.Fatal(format, args...)
}

func Panic(format string, args ...interface{}) {
	DefaultLogger.Panic(format, args...)
}

func Critical(format string, args ...interface{}) {
	DefaultLogger.Critical(format, args...)
}

func Error(format string, args ...interface{}) {
	DefaultLogger.Error(format, args...)
}

func Warning(format string, args ...interface{}) {
	DefaultLogger.Warning(format, args...)
}

func Notice(format string, args ...interface{}) {
	DefaultLogger.Notice(format, args...)
}

func Info(format string, args ...interface{}) {
	DefaultLogger.Info(format, args...)
}

func Debug(format string, args ...interface{}) {
	DefaultLogger.Debug(format, args...)
}

/////////////////
//             //
// BaseHandler //
//             //
/////////////////

type BaseHandler struct {
	Level     Level
	Formatter Formatter
}

func NewBaseHandler() *BaseHandler {
	return &BaseHandler{
		Level:     DefaultLevel,
		Formatter: DefaultFormatter,
	}
}

func (h *BaseHandler) SetLevel(l Level) {
	h.Level = l
}

func (h *BaseHandler) SetFormatter(f Formatter) {
	h.Formatter = f
}

func (h *BaseHandler) FilterAndFormat(rec *Record) string {
	if h.Level >= rec.Level {
		return h.Formatter.Format(rec)
	}
	return ""
}

///////////////////
//               //
// WriterHandler //
//               //
///////////////////

// WriterHandler is a handler implementation that writes the logging output to a io.Writer.
type WriterHandler struct {
	*BaseHandler
	w        io.Writer
	Colorize bool
}

func NewWriterHandler(w io.Writer) *WriterHandler {
	return &WriterHandler{
		BaseHandler: NewBaseHandler(),
		w:           w,
	}
}

func (b *WriterHandler) Handle(rec *Record) {
	message := b.BaseHandler.FilterAndFormat(rec)
	if message == "" {
		return
	}
	if b.Colorize {
		b.w.Write([]byte(fmt.Sprintf("\033[%dm", LevelColors[rec.Level])))
	}
	fmt.Fprint(b.w, message)
	if b.Colorize {
		b.w.Write([]byte("\033[0m")) // reset color
	}
}

func (b *WriterHandler) Close() {}

///////////////////
//               //
// SyslogHandler //
//               //
///////////////////

// SyslogHandler sends the logging output to syslog.
type SyslogHandler struct {
	*BaseHandler
	w *syslog.Writer
}

func NewSyslogHandler(tag string) (*SyslogHandler, error) {
	// Priority in New constructor is not important here because we
	// do not use w.Write() directly.
	w, err := syslog.New(syslog.LOG_INFO|syslog.LOG_USER, tag)
	if err != nil {
		return nil, err
	}
	return &SyslogHandler{
		BaseHandler: NewBaseHandler(),
		w:           w,
	}, nil
}

func (b *SyslogHandler) Handle(rec *Record) {
	message := b.BaseHandler.FilterAndFormat(rec)
	if message == "" {
		return
	}

	var fn func(string) error
	switch rec.Level {
	case CRITICAL:
		fn = b.w.Crit
	case ERROR:
		fn = b.w.Err
	case WARNING:
		fn = b.w.Warning
	case NOTICE:
		fn = b.w.Notice
	case INFO:
		fn = b.w.Info
	case DEBUG:
		fn = b.w.Debug
	}
	fn(message)
}

func (b *SyslogHandler) Close() {
	b.w.Close()
}

//////////////////
//              //
// MultiHandler //
//              //
//////////////////

// MultiHandler sends the log output to multiple handlers concurrently.
type MultiHandler struct {
	handlers []Handler
}

func NewMultiHandler(handlers ...Handler) *MultiHandler {
	return &MultiHandler{handlers: handlers}
}

func (b *MultiHandler) SetFormatter(f Formatter) {
	for _, h := range b.handlers {
		h.SetFormatter(f)
	}
}

func (b *MultiHandler) SetLevel(l Level) {
	for _, h := range b.handlers {
		h.SetLevel(l)
	}
}

func (b *MultiHandler) Handle(rec *Record) {
	wg := sync.WaitGroup{}
	wg.Add(len(b.handlers))
	for _, handler := range b.handlers {
		go func(handler Handler) {
			handler.Handle(rec)
			wg.Done()
		}(handler)
	}
	wg.Wait()
}

func (b *MultiHandler) Close() {
	wg := sync.WaitGroup{}
	wg.Add(len(b.handlers))
	for _, handler := range b.handlers {
		go func(handler Handler) {
			handler.Close()
			wg.Done()
		}(handler)
	}
	wg.Wait()
}
