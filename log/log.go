// Package log is mainly going to be used as a kludge to transition
// to a simpler logging setup in kaeter. Currently we often pass
// the logger instance around or expose it as a global. This
// will expose it as a singleton with a similar interface as
// the standard logger and other loggers we might want to use later.
// Transition plan:
// 0. [x] Implement logger interface
// 1. [x] Transition all kaeter code to use it
// 2. [x] Replace it with another logger
// 3. [ ] Remove this wrapper and use github.com/charmbracelet/log directly
package log

import (
	"github.com/charmbracelet/log"
)

// IsDebug returns true only when the log level is debug
func IsDebug() bool {
	return log.GetLevel() == log.DebugLevel
}

// Wrappers... added as needed
//revive:disable:exported

func Debug(message string, args ...any) {
	log.Helper()
	log.Debugf(message, args...)
}

func Debugln(msg any, keyvals ...any) {
	log.Helper()
	// no *ln in charm map to regular
	log.Debug(msg, keyvals...)
}

func Debugf(message string, args ...any) {
	log.Helper()
	log.Debugf(message, args...)
}

func Info(msg any, keyvals ...any) {
	log.Helper()
	log.Info(msg, keyvals...)
}

func Infoln(msg any, keyvals ...any) {
	log.Helper()
	// no *ln in charm map to regular
	log.Info(msg, keyvals...)
}

func Infof(message string, args ...any) {
	log.Helper()

	log.Infof(message, args...)
}

func Warnf(message string, args ...any) {
	log.Helper()
	log.Warnf(message, args...)
}

func Error(msg any, keyvals ...any) {
	log.Helper()
	log.Error(msg, keyvals...)
}

func Errorln(msg any, keyvals ...any) {
	log.Helper()
	// no *ln in charm map to regular
	log.Error(msg, keyvals...)
}

func Errorf(message string, args ...any) {
	log.Helper()
	log.Errorf(message, args...)
}

func Fatal(msg any, keyvals ...any) {
	log.Helper()
	log.Fatal(msg, keyvals...)
}

func Fatalln(msg any, keyvals ...any) {
	log.Helper()
	// no *ln in charm map to regular
	log.Fatal(msg, keyvals...)
}

func Fatalf(message string, args ...any) {
	log.Helper()
	log.Fatalf(message, args...)
}
