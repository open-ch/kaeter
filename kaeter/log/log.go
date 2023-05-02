// Package log is mainly going to be used as a kludge to transition
// to a simpler logging setup in kaeter. Currently we often pass
// the logger instance around or expose it as a global. This
// will expose it as a singleton with a similar interface as
// the standard logger and other loggers we might want to use later.
// Transition plan:
// 0. Implement logger interface
// 1. Transition all kaeter code to use it
// 2. Replace it with another logger
package log

import (
	"github.com/sirupsen/logrus"
)

var logger = logrus.New()

// SetLogger updates the current logger instance to a new one
func SetLogger(l *logrus.Logger) {
	logger = l
}

// GetLogger returns the current logger instance
func GetLogger() *logrus.Logger {
	return logger
}

// IsDebug returns true only when the log level is debug
func IsDebug() bool {
	return logger.GetLevel() == logrus.DebugLevel
}

// Wrappers... added as needed
//revive:disable:exported

func SetLevel(l logrus.Level) {
	logger.SetLevel(l)
}

func Debugln(args ...any) {
	logger.Debugln(args...)
}

func Debugf(message string, args ...any) {
	logger.Debugf(message, args...)
}

func Info(args ...any) {
	logger.Info(args...)
}

func Infoln(args ...any) {
	logger.Info(args...)
}

func Infof(message string, args ...any) {
	logger.Infof(message, args...)
}

func Warnf(message string, args ...any) {
	logger.Warnf(message, args...)
}

func Errorln(args ...any) {
	logger.Errorln(args...)
}

func Errorf(message string, args ...any) {
	logger.Errorf(message, args...)
}

func Fatal(args ...any) {
	logger.Fatal(args...)
}

func Fatalln(args ...any) {
	logger.Fatalln(args...)
}

func Fatalf(message string, args ...any) {
	logger.Fatalf(message, args...)
}
