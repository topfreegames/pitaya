package modules

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/constants"
)

func TestNewBinary(t *testing.T) {
	tables := []struct {
		name                     string
		binPath                  string
		args                     []string
		gracefulShutdownInterval time.Duration
	}{
		{"cmd1", "go", []string{"some", "arg"}, 2 * time.Second},
		{"cmd2", "go", []string{}, 2 * time.Second},
	}

	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			b := NewBinary(table.binPath, table.args, table.gracefulShutdownInterval)
			assert.NotNil(t, b)
			assert.Equal(t, table.binPath, b.binPath)
			assert.Equal(t, table.args, b.args)
			assert.Equal(t, table.gracefulShutdownInterval, b.gracefulShutdownInterval)
		})
	}
}

func TestInit(t *testing.T) {
	tables := []struct {
		name                     string
		binPath                  string
		args                     []string
		gracefulShutdownInterval time.Duration
	}{
		{"cmd1", "go", []string{"version"}, 2 * time.Second},
		{"cmd2", "go", []string{}, 2 * time.Second},
	}
	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			b := NewBinary(table.binPath, table.args, table.gracefulShutdownInterval)
			err := b.Init()
			assert.NoError(t, err)
		})
	}
}

func TestShutdown(t *testing.T) {
	tables := []struct {
		name                     string
		binPath                  string
		args                     []string
		gracefulShutdownInterval time.Duration
		err                      error
	}{
		{"cmd1", "tail", []string{"-f", "/dev/null"}, 2 * time.Second, nil},
		{"cmd2", "tail", []string{"-f", "/dev/null"}, 0 * time.Nanosecond, constants.ErrTimeoutTerminatingBinaryModule},
	}
	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			b := NewBinary(table.binPath, table.args, table.gracefulShutdownInterval)
			err := b.Init()
			assert.NoError(t, err)
			err = b.Shutdown()
			if table.err != nil {
				assert.EqualError(t, table.err, err.Error())
			} else {

				assert.NoError(t, err)
			}
		})
	}
}
