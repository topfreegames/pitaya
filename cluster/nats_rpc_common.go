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
	"os"
	"time"

	nats "github.com/nats-io/nats.go"
	"github.com/topfreegames/pitaya/v2/logger"
)

func getChannel(serverType, serverID string) string {
	return fmt.Sprintf("pitaya/servers/%s/%s", serverType, serverID)
}

func drainAndClose(nc *nats.Conn) error {
	if nc == nil {
		return nil
	}
	// Drain connection (this will flush any pending messages and prevent new ones)
	err := nc.Drain()
	if err != nil {
		logger.Log.Warnf("error draining nats connection: %v", err)
		// Even if drain fails, try to close
		nc.Close()
		return err
	}

	// Wait for drain to complete with timeout
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for nc.IsDraining() {
		select {
		case <-ticker.C:
			continue
		case <-timeout:
			logger.Log.Warn("drain timeout exceeded, forcing close")
			nc.Close()
			return fmt.Errorf("drain timeout exceeded")
		}
	}

	// Close will happen automatically after drain completes
	return nil
}

func setupNatsConn(connectString string, appDieChan chan bool, options ...nats.Option) (*nats.Conn, error) {
	connectedCh := make(chan bool)
	initialConnectErrorCh := make(chan error)
	natsOptions := append(
		options,
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.Log.Warnf("disconnected from nats (%s)! Reason: %q\n", nc.ConnectedAddr(), err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Log.Warnf("reconnected to nats server %s with address %s in cluster %s!", nc.ConnectedServerName(), nc.ConnectedAddr(), nc.ConnectedClusterName())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			err := nc.LastError()
			if err == nil {
				logger.Log.Warn("nats connection closed with no error.")
				return
			}

			logger.Log.Errorf("nats connection closed. reason: %q", nc.LastError())
			if appDieChan != nil {
				select {
				case appDieChan <- true:
					return
				case initialConnectErrorCh <- nc.LastError():
					logger.Log.Warnf("appDieChan not ready, sending error in initialConnectCh")
				default:
					logger.Log.Warnf("no termination channel available, sending termination signal to app")

					p, err := os.FindProcess(os.Getpid())
					if err != nil {
						logger.Log.Errorf("could not find current process: %v", err)
						os.Exit(1)
					}

					// On Windows, Signal() with Interrupt works
					// On Unix-like systems, this is equivalent to SIGINT
					err = p.Signal(os.Interrupt)
					if err != nil {
						logger.Log.Errorf("could not send interrupt signal to the application: %v", err)
						os.Exit(1)
					}
				}
			}
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			if err == nats.ErrSlowConsumer {
				dropped, _ := sub.Dropped()
				logger.Log.Warn("nats slow consumer on subject %q: dropped %d messages\n",
					sub.Subject, dropped)
			} else {
				logger.Log.Errorf(err.Error())
			}
		}),
		nats.ConnectHandler(func(nc *nats.Conn) {
			logger.Log.Infof("connected to nats on %s", nc.ConnectedAddr())
			connectedCh <- true
		}),
	)

	nc, err := nats.Connect(connectString, natsOptions...)
	if err != nil {
		return nil, err
	}
	maxConnTimeout := nc.Opts.Timeout
	if nc.Opts.RetryOnFailedConnect {
		// This is non-deterministic becase jitter TLS is different and we need to simplify
		// the calculations. What we want to do is simply not block forever the call while
		// we don't set a timeout so low that hinders our own reconnect config:
		// 		maxReconnectTimeout = reconnectWait + reconnectJitter + reconnectTimeout
		// 		connectionTimeout + (maxReconnectionAttemps * maxReconnectTimeout)
		// Thus, the time.After considers 2 times this value
		maxReconnectionTimeout := nc.Opts.ReconnectWait + nc.Opts.ReconnectJitter + nc.Opts.Timeout
		maxConnTimeout += time.Duration(nc.Opts.MaxReconnect) * maxReconnectionTimeout
	}

	logger.Log.Debugf("attempting nats connection for a max of %v", maxConnTimeout)
	select {
	case <-connectedCh:
		return nc, nil
	case err := <-initialConnectErrorCh:
		drainErr := drainAndClose(nc)
		if drainErr != nil {
			logger.Log.Warnf("failed to drain and close: %s", drainErr)
		}
		return nil, err
	case <-time.After(maxConnTimeout * 2):
		drainErr := drainAndClose(nc)
		if drainErr != nil {
			logger.Log.Warnf("failed to drain and close: %s", drainErr)
		}
		return nil, fmt.Errorf("timeout setting up nats connection")
	}
}
