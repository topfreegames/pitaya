package logrus

import (
	"github.com/sirupsen/logrus"
	"github.com/topfreegames/pitaya/v2/logger/interfaces"
)

type logrusImpl struct {
	impl *logrus.Entry
}

// New returns a new interfaces.Logger implementation based on logrus
func New() interfaces.Logger {
	log := logrus.New()
	return NewWithLogger(log)
}

// NewWithEntry returns a new interfaces.Logger implementation based on a provided logrus entry instance
func NewWithEntry(logger *logrus.Entry) interfaces.Logger {
	return &logrusImpl{impl: logger}
}

// NewWithLogger returns a new interfaces.Logger implementation based on a provided logrus instance
func NewWithLogger(logger *logrus.Logger) interfaces.Logger {
	return &logrusImpl{impl: logrus.NewEntry(logger)}
}

func (l *logrusImpl) Fatal(format ...interface{}) {
	l.impl.Fatal(format...)
}

func (l *logrusImpl) Fatalf(format string, args ...interface{}) {
	l.impl.Fatalf(format, args...)
}

func (l *logrusImpl) Fatalln(args ...interface{}) {
	l.impl.Fatalln(args...)
}

func (l *logrusImpl) Debug(args ...interface{}) {
	l.impl.Debug(args...)
}

func (l *logrusImpl) Debugf(format string, args ...interface{}) {
	l.impl.Debugf(format, args...)
}

func (l *logrusImpl) Debugln(args ...interface{}) {
	l.impl.Debugln(args...)
}

func (l *logrusImpl) Error(args ...interface{}) {
	l.impl.Error(args...)
}

func (l *logrusImpl) Errorf(format string, args ...interface{}) {
	l.impl.Errorf(format, args...)
}

func (l *logrusImpl) Errorln(args ...interface{}) {
	l.impl.Errorln(args...)
}

func (l *logrusImpl) Info(args ...interface{}) {
	l.impl.Info(args...)
}

func (l *logrusImpl) Infof(format string, args ...interface{}) {
	l.impl.Infof(format, args...)
}

func (l *logrusImpl) Infoln(args ...interface{}) {
	l.impl.Infoln(args...)
}

func (l *logrusImpl) Warn(args ...interface{}) {
	l.impl.Warn(args...)
}

func (l *logrusImpl) Warnf(format string, args ...interface{}) {
	l.impl.Warnf(format, args...)
}

func (l *logrusImpl) Warnln(args ...interface{}) {
	l.impl.Warnln(args...)
}

func (l *logrusImpl) WithFields(fields map[string]interface{}) interfaces.Logger {
	return &logrusImpl{impl: l.impl.WithFields(fields)}

}
func (l *logrusImpl) WithField(key string, value interface{}) interfaces.Logger {
	return &logrusImpl{impl: l.impl.WithField(key, value)}
}

func (l *logrusImpl) WithError(err error) interfaces.Logger {
	return &logrusImpl{impl: l.impl.WithError(err)}
}
