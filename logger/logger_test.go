package logger

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

func TestSetLogger(t *testing.T) {
	logger, hook := test.NewNullLogger()
	SetLogger(logger)
	Log.Info("test1")

	if len(hook.Entries) != 1 {
		t.Fail()
	}
	if hook.LastEntry().Level != logrus.InfoLevel {
		t.Fail()
	}
	hook.Reset()
}
