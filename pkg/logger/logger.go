package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

type Logger struct {
	logger *logrus.Logger
	entry  *logrus.Entry
	hook   func() (key, value string, ok bool)
}

func parseLevel(level string) logrus.Level {
	switch level {
	case "debug":
		return logrus.DebugLevel
	case "info":
		return logrus.InfoLevel
	case "warn":
		return logrus.WarnLevel
	case "error":
		return logrus.ErrorLevel
	case "fatal":
		return logrus.FatalLevel
	case "panic":
		return logrus.PanicLevel
	default:
		return logrus.InfoLevel
	}
}

func NewLogger(level string) *Logger {
	formatter := new(logrus.TextFormatter)
	formatter.TimestampFormat = "2006-02-01 15:04:05"
	formatter.FullTimestamp = true
	logger := logrus.New()
	logger.SetLevel(parseLevel(level))
	logger.Formatter = formatter
	entry := logger.WithFields(logrus.Fields{
		"pid": os.Getpid(),
	},
	)

	return &Logger{
		logger: logger,
		entry:  entry,
		hook:   func() (string, string, bool) { return "", "", false },
	}
}

func (l *Logger) SetHook(hook func() (key, value string, ok bool)) {
	l.hook = hook
}

// Debug logs a message at level Debug on the standard logger.
func (l *Logger) Debug(args ...interface{}) {
	key, value, ok := l.hook()
	if !ok {
		callstack(l.entry).Debug(args...)
		return
	}
	callstack(l.entry.WithField(key, value)).Debug(args...)
}

// Debugln logs a message at level Debug on the standard logger.
func (l *Logger) Debugln(args ...interface{}) {
	key, value, ok := l.hook()
	if !ok {
		callstack(l.entry).Debugln(args...)
		return
	}
	callstack(l.entry.WithField(key, value)).Debugln(args...)
}

// Debugf logs a message at level Debug on the standard logger.
func (l *Logger) Debugf(format string, args ...interface{}) {
	callstack(l.entry).Debugf(format, args...)
}

// Info logs a message at level Info on the standard logger.
func (l *Logger) Info(args ...interface{}) {
	callstack(l.entry).Info(args...)
}

// Infoln logs a message at level Info on the standard logger.
func (l *Logger) Infoln(args ...interface{}) {
	key, value, ok := l.hook()
	if !ok {
		callstack(l.entry).Infoln(args...)
		return
	}
	callstack(l.entry.WithField(key, value)).Infoln(args...)
}

// Infof logs a message at level Info on the standard logger.
func (l *Logger) Infof(format string, args ...interface{}) {
	key, value, ok := l.hook()
	if !ok {
		callstack(l.entry).Infof(format, args...)
		return
	}
	callstack(l.entry.WithField(key, value)).Infof(format, args...)
}

// Warn logs a message at level Warn on the standard logger.
func (l *Logger) Warn(args ...interface{}) {
	key, value, ok := l.hook()
	if !ok {
		callstack(l.entry).Warn(args...)
		return
	}
	callstack(l.entry.WithField(key, value)).Warn(args...)
}

// Warnln logs a message at level Warn on the standard logger.
func (l *Logger) Warnln(args ...interface{}) {
	key, value, ok := l.hook()
	if !ok {
		callstack(l.entry).Warnln(args...)
		return
	}
	callstack(l.entry.WithField(key, value)).Warnln(args...)
}

// Warnf logs a message at level Warn on the standard logger.
func (l *Logger) Warnf(format string, args ...interface{}) {
	key, value, ok := l.hook()
	if !ok {
		callstack(l.entry).Warnf(format, args...)
		return
	}
	callstack(l.entry.WithField(key, value)).Warnf(format, args...)
}

// Error logs a message at level Error on the standard logger.
func (l *Logger) Error(args ...interface{}) {
	callstack(l.entry).Error(args...)
}

// Errorln logs a message at level Error on the standard logger.
func (l *Logger) Errorln(args ...interface{}) {
	callstack(l.entry).Errorln(args...)
}

// Errorf logs a message at level Error on the standard logger.
func (l *Logger) Errorf(format string, args ...interface{}) {
	callstack(l.entry).Errorf(format, args...)
}

// Fatal logs a message at level Fatal on the standard logger.
func (l *Logger) Fatal(args ...interface{}) {
	callstack(l.entry).Fatal(args...)
}

// Fatalln logs a message at level Fatal on the standard logger.
func (l *Logger) Fatalln(args ...interface{}) {
	callstack(l.entry).Fatalln(args...)
}

// Fatalf logs a message at level Fatal on the standard logger.
func (l *Logger) Fatalf(format string, args ...interface{}) {
	callstack(l.entry).Fatalf(format, args...)
}
