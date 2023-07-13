package logrus

import (
	"github.com/sirupsen/logrus"
	"github.com/topfreegames/pitaya/v2/logger/interfaces"
)

type logrusImpl struct {
	Impl logrus.FieldLogger
}

// New returns a new interfaces.Logger Implementation based on logrus
func New() interfaces.Logger {
	log := logrus.New()
	return NewWithLogger(log)
}

// NewWithEntry returns a new interfaces.Logger Implementation based on a provided logrus entry instance
// Deprecated: NewWithEntry is deprecated.
func NewWithEntry(logger *logrus.Entry) interfaces.Logger {
	return &logrusImpl{Impl: logger}
}

// NewWithLogger returns a new interfaces.Logger Implementation based on a provided logrus instance
// Deprecated: NewWithLogger is deprecated.
func NewWithLogger(logger *logrus.Logger) interfaces.Logger {
	return &logrusImpl{Impl: logrus.NewEntry(logger)}
}

// NewWithFieldLogger returns a new interfaces.Logger Implementation based on a provided logrus instance
func NewWithFieldLogger(logger logrus.FieldLogger) interfaces.Logger {
	return &logrusImpl{Impl: logger}
}

func (l *logrusImpl) Fatal(format ...interface{}) {
	l.Impl.Fatal(format...)
}

func (l *logrusImpl) Fatalf(format string, args ...interface{}) {
	l.Impl.Fatalf(format, args...)
}

func (l *logrusImpl) Fatalln(args ...interface{}) {
	l.Impl.Fatalln(args...)
}

func (l *logrusImpl) Debug(args ...interface{}) {
	l.Impl.Debug(args...)
}

func (l *logrusImpl) Debugf(format string, args ...interface{}) {
	l.Impl.Debugf(format, args...)
}

func (l *logrusImpl) Debugln(args ...interface{}) {
	l.Impl.Debugln(args...)
}

func (l *logrusImpl) Error(args ...interface{}) {
	l.Impl.Error(args...)
}

func (l *logrusImpl) Errorf(format string, args ...interface{}) {
	l.Impl.Errorf(format, args...)
}

func (l *logrusImpl) Errorln(args ...interface{}) {
	l.Impl.Errorln(args...)
}

func (l *logrusImpl) Info(args ...interface{}) {
	l.Impl.Info(args...)
}

func (l *logrusImpl) Infof(format string, args ...interface{}) {
	l.Impl.Infof(format, args...)
}

func (l *logrusImpl) Infoln(args ...interface{}) {
	l.Impl.Infoln(args...)
}

func (l *logrusImpl) Warn(args ...interface{}) {
	l.Impl.Warn(args...)
}

func (l *logrusImpl) Warnf(format string, args ...interface{}) {
	l.Impl.Warnf(format, args...)
}

func (l *logrusImpl) Warnln(args ...interface{}) {
	l.Impl.Warnln(args...)
}

func (l *logrusImpl) Panic(args ...interface{}) {
	l.Impl.Panic(args...)
}

func (l *logrusImpl) Panicf(format string, args ...interface{}) {
	l.Impl.Panicf(format, args...)
}

func (l *logrusImpl) Panicln(args ...interface{}) {
	l.Impl.Panicln(args...)
}

func (l *logrusImpl) WithFields(fields map[string]interface{}) interfaces.Logger {
	return &logrusImpl{Impl: l.Impl.WithFields(fields)}
}

func (l *logrusImpl) WithField(key string, value interface{}) interfaces.Logger {
	return &logrusImpl{Impl: l.Impl.WithField(key, value)}
}

func (l *logrusImpl) WithError(err error) interfaces.Logger {
	return &logrusImpl{Impl: l.Impl.WithError(err)}
}
