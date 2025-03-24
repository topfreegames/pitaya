package assertions

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func ShouldEventuallyReturn(t testing.TB, wg *sync.WaitGroup, timeout time.Duration) {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return
	case <-time.After(timeout):
		assert.Fail(t, "timed out waiting for sync.WaitGroup to finish")
	}
}

func ShouldEventuallyClose(t testing.TB, channel chan bool, timeout time.Duration) {
	c := make(chan struct{})
	select {
	case <-channel:
		close(c)
		return
	case <-time.After(timeout):
		assert.Fail(t, "timed out waiting for channel to close")
	}
}
