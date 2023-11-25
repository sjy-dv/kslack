package log

import (
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kataras/pio"
)

type Log struct {
	// Logger is the original printer of this Log.
	Logger *Logger `json:"-"`
	// Time is the time of fired.
	Time time.Time `json:"-"`
	// Timestamp is the unix time in second of fired.
	Timestamp int64 `json:"timestamp,omitempty"`
	// Level is the log level.
	Level Level `json:"level"`
	// Message is the string reprensetation of the log's main body.
	Message string `json:"message"`
	// Fields any data information useful to represent this log.
	Fields Fields `json:"fields,omitempty"`
	// Stacktrace contains the stack callers when on `Debug` level.
	// The first one should be the Logger's direct caller function.
	Stacktrace []Frame `json:"stacktrace,omitempty"`
	// NewLine returns false if this Log
	// derives from a `Print` function,
	// otherwise true if derives from a `Println`, `Error`, `Errorf`, `Warn`, etc...
	//
	// This NewLine does not mean that `Message` ends with "\n" (or `pio#NewLine`).
	// NewLine has to do with the methods called,
	// not the original content of the `Message`.
	NewLine bool `json:"-"`
}

// Frame represents the log's caller.
type Frame struct {
	// Function is the package path-qualified function name of
	// this call frame. If non-empty, this string uniquely
	// identifies a single function in the program.
	// This may be the empty string if not known.
	Function string `json:"function"`
	// Source contains the file name and line number of the
	// location in this frame. For non-leaf frames, this will be
	// the location of a call.
	Source string `json:"source"`
}

// String method returns the concat value of "file:line".
// Implements the `fmt.Stringer` interface.
func (f Frame) String() string {
	return f.Source
}

// FormatTime returns the formatted `Time`.
func (l *Log) FormatTime() string {
	if l.Logger.TimeFormat == "" {
		return ""
	}
	return l.Time.Format(l.Logger.TimeFormat)
}

var funcNameReplacer = strings.NewReplacer(")", "", "(", "", "*", "")

// GetStacktrace tries to return the callers of this function.
func GetStacktrace(limit int) (callerFrames []Frame) {
	if limit < 0 {
		return nil
	}

	var pcs [32]uintptr
	n := runtime.Callers(1, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])

	for {
		f, more := frames.Next()
		file := filepath.ToSlash(f.File)

		if strings.Contains(file, "go/src/") {
			continue
		}

		if strings.Contains(file, "github.com/kataras/log") &&
			!(strings.Contains(file, "_examples") ||
				strings.Contains(file, "_test.go") ||
				strings.Contains(file, "integration.go")) {
			continue
		}

		if file != "" { // keep it here, break should be respected.
			funcName := f.Function
			if idx := strings.Index(funcName, ".("); idx > 1 {
				funcName = funcNameReplacer.Replace(funcName[idx+1:])
				// e.g. method: github.com/kataras/iris/v12.(*Application).Listen to:
				//      Application.Listen
			} else if idx = strings.LastIndexByte(funcName, '/'); idx >= 0 && len(funcName) > idx+1 {
				funcName = strings.Replace(funcName[idx+1:], ".", "/", 1)
				// e.g. package-level function: github.com/kataras/iris/v12/context.Do to
				// context/Do
			}

			callerFrames = append(callerFrames, Frame{
				Function: funcName,
				Source:   fmt.Sprintf("%s:%d", f.File, f.Line),
			})

			if limit > 0 && len(callerFrames) >= limit {
				break
			}
		}

		if !more {
			break
		}
	}

	return
}

// Now is called to set the log's timestamp value.
// It can be altered through initialization of the program
// to customize the behavior of getting the current time.
var Now func() time.Time = time.Now

// NewLine can override the default package-level line breaker, "\n".
// It should be called (in-sync) before  the print or leveled functions.
//
// See `github.com/kataras/pio#NewLine` and `Logger#NewLine` too.
func NewLine(newLineChar string) {
	pio.NewLine = []byte(newLineChar)
}

// Default is the package-level ready-to-use logger,
// level had set to "info", is changeable.
var Default = New()

// Reset re-sets the default logger to an empty one.
func Reset() {
	Default = New()
}

// SetOutput overrides the Default Logger's Printer's output with another `io.Writer`.
func SetOutput(w io.Writer) {
	Default.SetOutput(w)
}

// AddOutput adds one or more `io.Writer` to the Default Logger's Printer.
//
// If one of the "writers" is not a terminal-based (i.e File)
// then colors will be disabled for all outputs.
func AddOutput(writers ...io.Writer) {
	Default.AddOutput(writers...)
}

// SetPrefix sets a prefix for the default package-level Logger.
//
// The prefix is the first space-separated
// word that is being presented to the output.
// It's written even before the log level text.
//
// Returns itself.
func SetPrefix(s string) *Logger {
	return Default.SetPrefix(s)
}

// SetTimeFormat sets time format for logs,
// if "s" is empty then time representation will be off.
func SetTimeFormat(s string) *Logger {
	return Default.SetTimeFormat(s)
}

// SetStacktraceLimit sets a stacktrace entries limit
// on `Debug` level.
// Zero means all number of stack entries will be logged.
// Negative value disables the stacktrace field.
func SetStacktraceLimit(limit int) *Logger {
	return Default.SetStacktraceLimit(limit)
}

// RegisterFormatter registers a Formatter for this logger.
func RegisterFormatter(f Formatter) *Logger {
	return Default.RegisterFormatter(f)
}

// SetFormat sets a default formatter for all log levels.
func SetFormat(formatter string, opts ...interface{}) *Logger {
	return Default.SetFormat(formatter, opts...)
}

// SetLevelFormat changes the output format for the given "levelName".
func SetLevelFormat(levelName string, formatter string, opts ...interface{}) *Logger {
	return Default.SetLevelFormat(levelName, formatter, opts...)
}

// SetLevelOutput sets a destination log output for the specific "levelName".
// For multiple writers use the `io.Multiwriter` wrapper.
func SetLevelOutput(levelName string, w io.Writer) *Logger {
	return Default.SetLevelOutput(levelName, w)
}

// GetLevelOutput returns the responsible writer for the given "levelName".
// If not a registered writer is set for that level then it returns
// the logger's default printer. It does NOT return nil.
func GetLevelOutput(levelName string) io.Writer {
	return Default.GetLevelOutput(levelName)
}

// SetLevel accepts a string representation of
// a `Level` and returns a `Level` value based on that "levelName".
//
// Available level names are:
// "disable"
// "fatal"
// "error"
// "warn"
// "info"
// "debug"
//
// Alternatively you can use the exported `Default.Level` field, i.e `Default.Level = log.ErrorLevel`
func SetLevel(levelName string) {
	Default.SetLevel(levelName)
}

// Print prints a log message without levels and colors.
func Print(v ...interface{}) {
	Default.Print(v...)
}

// Println prints a log message without levels and colors.
// It adds a new line at the end.
func Println(v ...interface{}) {
	Default.Println(v...)
}

// Logf prints a leveled log message to the output.
// This method can be used to use custom log levels if needed.
// It adds a new line in the end.
func Logf(level Level, format string, args ...interface{}) {
	Default.Logf(level, format, args...)
}

// Fatal `os.Exit(1)` exit no matter the level of the logger.
// If the logger's level is fatal, error, warn, info or debug
// then it will print the log message too.
func Fatal(v ...interface{}) {
	Default.Fatal(v...)
}

// Fatalf will `os.Exit(1)` no matter the level of the logger.
// If the logger's level is fatal, error, warn, info or debug
// then it will print the log message too.
func Fatalf(format string, args ...interface{}) {
	Default.Fatalf(format, args...)
}

// Error will print only when logger's Level is error, warn, info or debug.
func Error(v ...interface{}) {
	Default.Error(v...)
}

// Errorf will print only when logger's Level is error, warn, info or debug.
func Errorf(format string, args ...interface{}) {
	Default.Errorf(format, args...)
}

// Warn will print when logger's Level is warn, info or debug.
func Warn(v ...interface{}) {
	Default.Warn(v...)
}

// Warnf will print when logger's Level is warn, info or debug.
func Warnf(format string, args ...interface{}) {
	Default.Warnf(format, args...)
}

// Info will print when logger's Level is info or debug.
func Info(v ...interface{}) {
	Default.Info(v...)
}

// Infof will print when logger's Level is info or debug.
func Infof(format string, args ...interface{}) {
	Default.Infof(format, args...)
}

// Debug will print when logger's Level is debug.
func Debug(v ...interface{}) {
	Default.Debug(v...)
}

// Debugf will print when logger's Level is debug.
func Debugf(format string, args ...interface{}) {
	Default.Debugf(format, args...)
}

// Install receives  an external logger
// and automatically adapts its print functions.
//
// Install adds a log handler to support third-party integrations,
// it can be used only once per `log#Logger` instance.
//
// For example, if you want to print using a logrus
// logger you can do the following:
//
//	Install(logrus.StandardLogger())
//
// Or the standard log's Logger:
//
//	import "log"
//	myLogger := log.New(os.Stdout, "", 0)
//	Install(myLogger)
//
// Or even the slog/log's Logger:
//
//	import "log/slog"
//	myLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
//	Install(myLogger) OR Install(slog.Default())
//
// Look `log#Logger.Handle` for more.
func Install(logger any) {
	Default.Install(logger)
}

// Handle adds a log handler to the default logger.
//
// Handlers can be used to intercept the message between a log value
// and the actual print operation, it's called
// when one of the print functions called.
// If it's return value is true then it means that the specific
// handler handled the log by itself therefore no need to
// proceed with the default behavior of printing the log
// to the specified logger's output.
//
// It stops on the handler which returns true firstly.
// The `Log` value holds the level of the print operation as well.
func Handle(handler Handler) {
	Default.Handle(handler)
}

// Hijack adds a hijacker to the low-level logger's Printer.
// If you need to implement such as a low-level hijacker manually,
// then you have to make use of the pio library.
func Hijack(hijacker func(ctx *pio.Ctx)) {
	Default.Hijack(hijacker)
}

// Scan scans everything from "r" and prints
// its new contents to the logger's Printer's Output,
// forever or until the returning "cancel" is fired, once.
func Scan(r io.Reader) (cancel func()) {
	return Default.Scan(r)
}

// Child (creates if not exists and) returns a new child
// Logger based on the current logger's fields.
//
// Can be used to separate logs by category.
// If the "key" is string then it's used as prefix,
// which is appended to the current prefix one.
func Child(key interface{}) *Logger {
	return Default.Child(key)
}

// SetChildPrefix same as `SetPrefix` but it does NOT
// override the existing, instead the given "s"
// is appended to the current one. It's useful
// to chian loggers with their own names/prefixes.
// It does add the ": " in the end of "s" if it's missing.
// It returns itself.
func SetChildPrefix(s string) *Logger {
	return Default.SetChildPrefix(s)
}

// LastChild returns the last registered child Logger.
func LastChild() *Logger {
	return Default.LastChild()
}
