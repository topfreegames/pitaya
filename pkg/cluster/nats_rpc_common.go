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
	"syscall"
	"time"

	nats "github.com/nats-io/nats.go"
	"github.com/topfreegames/pitaya/v3/pkg/logger"
)

func getChannel(serverType, serverID string) string {
	return fmt.Sprintf("pitaya/servers/%s/%s", serverType, serverID)
}

func setupNatsConn(connectString string, appDieChan chan bool, options ...nats.Option) (*nats.Conn, error) {
	connectedCh := make(chan bool)
	initialConnectErrorCh := make(chan error)
	natsOptions := append(
		options,
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			logger.Log.Warnf("disconnected from nats! Reason: %q\n", err)
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
					logger.Log.Warnf("no termination channel available, sending SIGTERM to app")
					err := syscall.Kill(os.Getpid(), syscall.SIGTERM)
					if err != nil {
						logger.Log.Errorf("could not kill the application via SIGTERM, exiting", err)
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
		nats.ConnectHandler(func(*nats.Conn) {
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
		return nil, err
	case <-time.After(maxConnTimeout * 2):
		return nil, fmt.Errorf("timeout setting up nats connection")
	}
}
