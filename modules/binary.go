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
	"bufio"
	"os/exec"
	"syscall"
	"time"

	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/logger"
)

// Binary is a pitaya module that starts a binary as a child process and
// pipes its stdout
type Binary struct {
	Base
	binPath                  string
	args                     []string
	gracefulShutdownInterval time.Duration
	cmd                      *exec.Cmd
	exitCh                   chan struct{}
}

// NewBinary creates a new binary module with the given path
func NewBinary(binPath string, args []string, gracefulShutdownInterval ...time.Duration) *Binary {
	gracefulTime := 15 * time.Second
	if len(gracefulShutdownInterval) > 0 {
		gracefulTime = gracefulShutdownInterval[0]
	}
	return &Binary{
		binPath:                  binPath,
		args:                     args,
		gracefulShutdownInterval: gracefulTime,
		exitCh:                   make(chan struct{}),
	}
}

// GetExitChannel gets a channel that is closed when the binary dies
func (b *Binary) GetExitChannel() chan struct{} {
	return b.exitCh
}

// Init initializes the binary
func (b *Binary) Init() error {
	b.cmd = exec.Command(b.binPath, b.args...)
	stdout, _ := b.cmd.StdoutPipe()
	stdOutScanner := bufio.NewScanner(stdout)
	stderr, _ := b.cmd.StderrPipe()
	stdErrScanner := bufio.NewScanner(stderr)
	go func() {
		for stdOutScanner.Scan() {
			logger.Log.Info(stdOutScanner.Text())
		}
	}()
	go func() {
		for stdErrScanner.Scan() {
			logger.Log.Error(stdErrScanner.Text())
		}
	}()
	err := b.cmd.Start()
	go func() {
		b.cmd.Wait()
		close(b.exitCh)
	}()
	return err
}

// Shutdown shutdowns the binary module
func (b *Binary) Shutdown() error {
	err := b.cmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		return err
	}
	timeout := time.After(b.gracefulShutdownInterval)
	select {
	case <-b.exitCh:
		return nil
	case <-timeout:
		b.cmd.Process.Kill()
		return constants.ErrTimeoutTerminatingBinaryModule
	}
}
