package test

import (
	tests "github.com/sirupsen/logrus/hooks/test"
	"github.com/topfreegames/pitaya/v2/logger/interfaces"
	lwrapper "github.com/topfreegames/pitaya/v2/logger/logrus"
	"io"
)

// NewNullLogger creates a discarding logger and installs the test hook.
func NewNullLogger() (interfaces.Logger, *tests.Hook) {
	logger, hook := tests.NewNullLogger()
	logger.Out = io.Discard
	return lwrapper.NewWithFieldLogger(logger), hook
}
