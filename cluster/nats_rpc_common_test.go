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

package cluster

import (
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/test"
	nats "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/topfreegames/pitaya/v2/helpers"
)

func getServer() *Server {
	return &Server{
		ID:       "id1",
		Type:     "type1",
		Frontend: true,
	}
}

func TestNatsRPCCommonGetChannel(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "pitaya/servers/type1/sv1", getChannel("type1", "sv1"))
	assert.Equal(t, "pitaya/servers/2type1/2sv1", getChannel("2type1", "2sv1"))
}

func TestNatsRPCCommonSetupNatsConn(t *testing.T) {
	t.Parallel()
	s := helpers.GetTestNatsServer(t)
	defer s.Shutdown()
	conn, err := setupNatsConn(fmt.Sprintf("nats://%s", s.Addr()), nil)
	assert.NoError(t, err)
	assert.NotNil(t, conn)
}

func TestNatsRPCCommonSetupNatsConnShouldError(t *testing.T) {
	t.Parallel()
	conn, err := setupNatsConn("nats://localhost:1234", nil)
	assert.Error(t, err)
	assert.Nil(t, conn)
}

func TestNatsRPCCommonCloseHandler(t *testing.T) {
	t.Parallel()
	s := helpers.GetTestNatsServer(t)

	dieChan := make(chan bool)

	conn, err := setupNatsConn(fmt.Sprintf("nats://%s", s.Addr()), dieChan, nats.MaxReconnects(1),
		nats.ReconnectWait(1*time.Millisecond))
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	s.Shutdown()

	value, ok := <-dieChan
	assert.True(t, ok)
	assert.True(t, value)
}

func TestSetupNatsConnReconnection(t *testing.T) {
	t.Run("waits for reconnection on initial failure", func(t *testing.T) {
		// Use an invalid address first to force initial connection failure
		invalidAddr := "nats://invalid:4222"
		validAddr := "nats://localhost:4222"

		urls := fmt.Sprintf("%s,%s", invalidAddr, validAddr)

		go func() {
			time.Sleep(50 * time.Millisecond)
			ts := test.RunDefaultServer()
			defer ts.Shutdown()
			<-time.After(200 * time.Millisecond)
		}()

		// Setup connection with retry enabled
		appDieCh := make(chan bool)
		conn, err := setupNatsConn(
			urls,
			appDieCh,
			nats.ReconnectWait(10*time.Millisecond),
			nats.MaxReconnects(5),
			nats.RetryOnFailedConnect(true),
		)

		assert.NoError(t, err)
		assert.NotNil(t, conn)
		assert.True(t, conn.IsConnected())

		conn.Close()
	})

	t.Run("does not block indefinitely if all connect attempts fail", func(t *testing.T) {
		invalidAddr := "nats://invalid:4222"

		appDieCh := make(chan bool)
		done := make(chan any)

		ts := test.RunDefaultServer()
		defer ts.Shutdown()

		go func() {
			conn, err := setupNatsConn(
				invalidAddr,
				appDieCh,
				nats.ReconnectWait(10*time.Millisecond),
				nats.MaxReconnects(2),
				nats.RetryOnFailedConnect(true),
			)
			assert.Error(t, err)
			assert.Nil(t, conn)
			close(done)
			close(appDieCh)
		}()

		select {
		case <-appDieCh:
		case <-done:
		case <-time.After(250 * time.Millisecond):
			t.Fail()
		}
	})

	t.Run("if it fails to connect, exit with error even if appDieChan is not ready to listen", func(t *testing.T) {
		invalidAddr := "nats://invalid:4222"

		appDieCh := make(chan bool)
		done := make(chan any)

		ts := test.RunDefaultServer()
		defer ts.Shutdown()

		go func() {
			conn, err := setupNatsConn(invalidAddr, appDieCh)
			assert.Error(t, err)
			assert.Nil(t, conn)
			close(done)
			close(appDieCh)
		}()

		select {
		case <-done:
		case <-time.After(50 * time.Millisecond):
			t.Fail()
		}
	})

	t.Run("if connection takes too long, exit with error after waiting maxReconnTimeout", func(t *testing.T) {
		invalidAddr := "nats://invalid:4222"

		appDieCh := make(chan bool)
		done := make(chan any)

		initialConnectionTimeout := time.Nanosecond
		maxReconnectionAtetmpts := 1
		reconnectWait := time.Nanosecond
		reconnectJitter := time.Nanosecond
		maxReconnectionTimeout := reconnectWait + reconnectJitter + initialConnectionTimeout
		maxReconnTimeout := initialConnectionTimeout + (time.Duration(maxReconnectionAtetmpts) * maxReconnectionTimeout)

		maxTestTimeout := 100 * time.Millisecond

		// Assert that if it fails because of connection timeout the test will capture
		assert.Greater(t, maxTestTimeout, maxReconnTimeout)

		ts := test.RunDefaultServer()
		defer ts.Shutdown()

		go func() {
			conn, err := setupNatsConn(
				invalidAddr,
				appDieCh,
				nats.Timeout(initialConnectionTimeout),
				nats.ReconnectWait(reconnectWait),
				nats.MaxReconnects(maxReconnectionAtetmpts),
				nats.ReconnectJitter(reconnectJitter, reconnectJitter),
				nats.RetryOnFailedConnect(true),
			)
			assert.Error(t, err)
			assert.ErrorContains(t, err, "timeout setting up nats connection")
			assert.Nil(t, conn)
			close(done)
			close(appDieCh)
		}()

		select {
		case <-done:
		case <-time.After(maxTestTimeout):
			t.Fail()
		}
	})
}
