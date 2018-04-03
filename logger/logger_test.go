package logger

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestInitLogger(t *testing.T) {
	initLogger()
	assert.NotNil(t, Log)
}

func TestSetLogger(t *testing.T) {
	l := logrus.New()
	SetLogger(l)
	assert.Equal(t, Log, l)
}
