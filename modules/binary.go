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
		binPath: binPath,
		args:    args,
		gracefulShutdownInterval: gracefulTime,
		exitCh: make(chan struct{}),
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

// AfterInit runs after initialization tasks
func (b *Binary) AfterInit() {}

// BeforeShutdown runs tasks before shutting down the binary module
func (b *Binary) BeforeShutdown() {}

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
