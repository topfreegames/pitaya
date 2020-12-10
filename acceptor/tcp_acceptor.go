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
	"crypto/tls"
	"io"
	"io/ioutil"
	"net"

	"github.com/topfreegames/pitaya/conn/codec"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/logger"
)

// TCPAcceptor struct
type TCPAcceptor struct {
	addr     string
	connChan chan PlayerConn
	listener net.Listener
	running  bool
	certFile string
	keyFile  string
}

type tcpPlayerConn struct {
	net.Conn
}

// GetNextMessage reads the next message available in the stream
func (t *tcpPlayerConn) GetNextMessage() (b []byte, err error) {
	header, err := ioutil.ReadAll(io.LimitReader(t.Conn, codec.HeadLength))
	if err != nil {
		return nil, err
	}
	// if the header has no data, we can consider it as a closed connection
	if len(header) == 0 {
		return nil, constants.ErrConnectionClosed
	}
	msgSize, _, err := codec.ParseHeader(header)
	if err != nil {
		return nil, err
	}
	msgData, err := ioutil.ReadAll(io.LimitReader(t.Conn, int64(msgSize)))
	if err != nil {
		return nil, err
	}
	if len(msgData) < msgSize {
		return nil, constants.ErrReceivedMsgSmallerThanExpected
	}
	return append(header, msgData...), nil
}

// NewTCPAcceptor creates a new instance of tcp acceptor
func NewTCPAcceptor(addr string, certs ...string) *TCPAcceptor {
	keyFile := ""
	certFile := ""
	if len(certs) != 2 && len(certs) != 0 {
		panic(constants.ErrInvalidCertificates)
	} else if len(certs) == 2 {
		certFile = certs[0]
		keyFile = certs[1]
	}

	return &TCPAcceptor{
		addr:     addr,
		connChan: make(chan PlayerConn),
		running:  false,
		certFile: certFile,
		keyFile:  keyFile,
	}
}

// GetAddr returns the addr the acceptor will listen on
func (a *TCPAcceptor) GetAddr() string {
	if a.listener != nil {
		return a.listener.Addr().String()
	}
	return ""
}

// GetConnChan gets a connection channel
func (a *TCPAcceptor) GetConnChan() chan PlayerConn {
	return a.connChan
}

// Stop stops the acceptor
func (a *TCPAcceptor) Stop() {
	a.running = false
	a.listener.Close()
}

func (a *TCPAcceptor) hasTLSCertificates() bool {
	return a.certFile != "" && a.keyFile != ""
}

// ListenAndServe using tcp acceptor
func (a *TCPAcceptor) ListenAndServe() {
	if a.hasTLSCertificates() {
		a.ListenAndServeTLS(a.certFile, a.keyFile)
		return
	}

	listener, err := net.Listen("tcp", a.addr)
	if err != nil {
		logger.Log.Fatalf("Failed to listen: %s", err.Error())
	}
	a.listener = listener
	a.running = true
	a.serve()
}

// ListenAndServeTLS listens using tls
func (a *TCPAcceptor) ListenAndServeTLS(cert, key string) {
	crt, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		logger.Log.Fatalf("Failed to listen: %s", err.Error())
	}

	tlsCfg := &tls.Config{Certificates: []tls.Certificate{crt}}

	listener, err := tls.Listen("tcp", a.addr, tlsCfg)
	a.listener = listener
	a.running = true
	a.serve()
}

func (a *TCPAcceptor) serve() {
	defer a.Stop()
	for a.running {
		conn, err := a.listener.Accept()
		if err != nil {
			logger.Log.Errorf("Failed to accept TCP connection: %s", err.Error())
			continue
		}

		a.connChan <- &tcpPlayerConn{
			Conn: conn,
		}
	}
}
