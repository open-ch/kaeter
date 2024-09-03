// Package log is mainly going to be used as a kludge to transition
// to a simpler logging setup in kaeter. Currently we often pass
// the logger instance around or expose it as a global. This
// will expose it as a singleton with a similar interface as
// the standard logger and other loggers we might want to use later.
// Transition plan:
// 0. [x] Implement logger interface
// 1. [x] Transition all kaeter code to use it
// 2. [x] Replace it with another logger
// 3. [ ] Remove this wrapper and use github.com/charmbracelet/log or log/slog directly
package log

import (
	"log/slog"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
)

// Initialize configures the default logger reading settings from viper, ideally
// this is called after viper is initialized
func Initialize() {
	charmLogger := log.New(os.Stderr)
	setLevelUsingConfig(charmLogger)

	logger := slog.New(charmLogger)
	slog.SetDefault(logger)
}

func setLevelUsingConfig(charmLogger *log.Logger) {
	charmLogger.SetReportTimestamp(false)
	if viper.GetBool("debug") {
		charmLogger.SetLevel(log.DebugLevel)
		charmLogger.SetReportCaller(true)
	} else if viper.GetString("log-level") != "" {
		switch viper.GetString("log-level") {
		case "debug":
			charmLogger.SetLevel(log.DebugLevel)
		case "info":
			charmLogger.SetLevel(log.InfoLevel)
		case "warn":
			charmLogger.SetLevel(log.WarnLevel)
		case "error":
			charmLogger.SetLevel(log.ErrorLevel)
		default:
			charmLogger.Warn("Unknown log level (supported levels: debug, info, warn, error)", "log-level", viper.GetString("log-level"))
		}
	}
}

// Wrappers... added as needed
//revive:disable:exported

func Debug(message string, args ...any) {
	log.Helper()
	slog.Debug(message, args...)
}

func Info(message string, keyvals ...any) {
	log.Helper()
	slog.Info(message, keyvals...)
}

func Infof(message string, args ...any) {
	log.Helper()
	// TODO use slog
	log.Infof(message, args...)
}

func Warn(message string, keyvals ...any) {
	log.Helper()
	slog.Warn(message, keyvals...)
}

func Error(message string, keyvals ...any) {
	log.Helper()
	slog.Error(message, keyvals...)
}
