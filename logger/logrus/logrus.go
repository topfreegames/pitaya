package logrus

import (
	"github.com/sirupsen/logrus"
	"github.com/topfreegames/pitaya/v2/logger/interfaces"
)

type logrusImpl struct {
	logrus.FieldLogger
}

// New returns a new interfaces.Logger implementation based on logrus
func New() interfaces.Logger {
	log := logrus.New()
	return NewWithLogger(log)
}

// NewWithEntry returns a new interfaces.Logger implementation based on a provided logrus entry instance
// Deprecated: NewWithEntry is deprecated.
func NewWithEntry(logger *logrus.Entry) interfaces.Logger {
	return &logrusImpl{FieldLogger: logger}
}

// NewWithLogger returns a new interfaces.Logger implementation based on a provided logrus instance
// Deprecated: NewWithLogger is deprecated.
func NewWithLogger(logger *logrus.Logger) interfaces.Logger {
	return &logrusImpl{FieldLogger: logrus.NewEntry(logger)}
}

// NewWithFieldLogger returns a new interfaces.Logger implementation based on a provided logrus instance
func NewWithFieldLogger(logger logrus.FieldLogger) interfaces.Logger {
	return &logrusImpl{FieldLogger: logger}
}

func (l *logrusImpl) WithFields(fields map[string]interface{}) interfaces.Logger {
	return &logrusImpl{FieldLogger: l.FieldLogger.WithFields(fields)}
}

func (l *logrusImpl) WithField(key string, value interface{}) interfaces.Logger {
	return &logrusImpl{FieldLogger: l.FieldLogger.WithField(key, value)}
}

func (l *logrusImpl) WithError(err error) interfaces.Logger {
	return &logrusImpl{FieldLogger: l.FieldLogger.WithError(err)}
}

func (l *logrusImpl) LogrusLogger() logrus.FieldLogger {
	return l.FieldLogger
}
