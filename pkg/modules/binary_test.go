// Copyright (c) TFG Co. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
package modules

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
	}
	for _, table := range tables {
		t.Run(table.name, func(t *testing.T) {
			b := NewBinary(table.binPath, table.args, table.gracefulShutdownInterval)
			err := b.Init()
			assert.NoError(t, err)
			err = b.Shutdown()
			assert.NoError(t, err)
		})
	}
}
