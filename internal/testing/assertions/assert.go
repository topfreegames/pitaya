package assertions

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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
