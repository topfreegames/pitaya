// Copyright (c) nano Author and TFG Co. All Rights Reserved.
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

package acceptor

import (
    "context"
    "crypto/tls"
    "net"

    "github.com/quic-go/quic-go"
    "github.com/quic-go/quic-go/http3"
    "github.com/topfreegames/pitaya/v3/pkg/conn/codec"
    "github.com/topfreegames/pitaya/v3/pkg/constants"
    "github.com/topfreegames/pitaya/v3/pkg/logger"
)

// QUICAcceptor struct
type QUICAcceptor struct {
    addr       string
    connChan   chan PlayerConn
    listener   quic.Listener
    running    bool
    tlsConfig  *tls.Config
    quicConfig *quic.Config
}

type quicPlayerConn struct {
    quic.Connection
    remoteAddr net.Addr
}

func (q *quicPlayerConn) RemoteAddr() net.Addr {
    return q.remoteAddr
}

// GetNextMessage reads the next message available in the stream
func (q *quicPlayerConn) GetNextMessage() (b []byte, err error) {
    stream, err := q.OpenStreamSync(context.Background())
    if err != nil {
        return nil, err
    }

    header, err := codec.ReadBytes(stream, codec.HeadLength)
    if err != nil {
        return nil, err
    }

    if len(header) == 0 {
        return nil, constants.ErrConnectionClosed
    }

    msgSize, _, err := codec.ParseHeader(header)
    if err != nil {
        return nil, err
    }

    msgData, err := codec.ReadBytes(stream, int(msgSize))
    if err != nil {
        return nil, err
    }

    if len(msgData) < msgSize {
        return nil, constants.ErrReceivedMsgSmallerThanExpected
    }

    return append(header, msgData...), nil
}

// NewQUICAcceptor creates a new instance of QUIC acceptor
func NewQUICAcceptor(addr string, tlsConfig *tls.Config, quicConfig *quic.Config) *QUICAcceptor {
    return &QUICAcceptor{
        addr:       addr,
        connChan:   make(chan PlayerConn),
        running:    false,
        tlsConfig:  tlsConfig,
        quicConfig: quicConfig,
    }
}

// GetAddr returns the address the acceptor will listen on
func (q *QUICAcceptor) GetAddr() string {
    if q.listener != nil {
        return q.listener.Addr().String()
    }
    return ""
}

// GetConnChan gets a connection channel
func (q *QUICAcceptor) GetConnChan() chan PlayerConn {
    return q.connChan
}

// Stop stops the acceptor
func (q *QUICAcceptor) Stop() {
    q.running = false
    q.listener.Close()
}

// ListenAndServe starts the QUIC listener
func (q *QUICAcceptor) ListenAndServe() {
    listener, err := quic.ListenAddr(q.addr, q.tlsConfig, q.quicConfig)
    if err != nil {
        logger.Log.Fatalf("Failed to listen: %s", err.Error())
    }

    q.listener = listener
    q.running = true
    q.serve()
}

func (q *QUICAcceptor) serve() {
    defer q.Stop()
    for q.running {
        session, err := q.listener.Accept(context.Background())
        if err != nil {
            logger.Log.Errorf("Failed to accept QUIC connection: %s", err.Error())
            continue
        }

        q.connChan <- &quicPlayerConn{
            Connection: session,
            remoteAddr: session.RemoteAddr(),
        }
    }
}

// IsRunning checks if the acceptor is running
func (q *QUICAcceptor) IsRunning() bool {
    return q.running
}

func (q *QUICAcceptor) GetConfiguredAddress() string {
    return q.addr
}

// TODO
// func (q *QUICAcceptor) ServeHTTP3(certFile, keyFile string, handler http.Handler) {
// }

