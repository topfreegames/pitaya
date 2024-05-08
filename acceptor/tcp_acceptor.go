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
	"fmt"
	"io"
	"net"

	"github.com/topfreegames/pitaya/v2/conn/codec"
	"github.com/topfreegames/pitaya/v2/constants"
	"github.com/topfreegames/pitaya/v2/logger"
)

// TCPAcceptor struct
type TCPAcceptor struct {
	addr          string
	connChan      chan PlayerConn
	listener      net.Listener
	running       bool
	certs         []tls.Certificate
	proxyProtocol bool
}

type tcpPlayerConn struct {
	net.Conn
	remoteAddr net.Addr
}

func (t *tcpPlayerConn) RemoteAddr() net.Addr {
	return t.remoteAddr
}

// GetNextMessage reads the next message available in the stream
func (t *tcpPlayerConn) GetNextMessage() (b []byte, err error) {
	header, err := io.ReadAll(io.LimitReader(t.Conn, codec.HeadLength))
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
	msgData, err := io.ReadAll(io.LimitReader(t.Conn, int64(msgSize)))
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
	certificates := []tls.Certificate{}
	if len(certs) != 2 && len(certs) != 0 {
		panic(constants.ErrIncorrectNumberOfCertificates)
	} else if len(certs) == 2 && certs[0] != "" && certs[1] != "" {
		cert, err := tls.LoadX509KeyPair(certs[0], certs[1])
		if err != nil {
			panic(fmt.Errorf("%w: %v", constants.ErrInvalidCertificates, err))
		}
		certificates = append(certificates, cert)
	}

	return NewTLSAcceptor(addr, certificates...)
}

func NewTLSAcceptor(addr string, certs ...tls.Certificate) *TCPAcceptor {
	return &TCPAcceptor{
		addr:          addr,
		connChan:      make(chan PlayerConn),
		running:       false,
		certs:         certs,
		proxyProtocol: false,
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
	return len(a.certs) > 0
}

// ListenAndServe using tcp acceptor
func (a *TCPAcceptor) ListenAndServe() {

	listener := a.createBaseListener()

	if a.hasTLSCertificates() {
		listener = a.listenAndServeTLS(listener)
	}

	a.listener = listener
	a.running = true
	a.serve()
}

// ListenAndServeTLS listens using tls
func (a *TCPAcceptor) ListenAndServeTLS(cert, key string) {
	listener := a.createBaseListener()

	crt, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		logger.Log.Fatalf("Failed to listen: %s", err.Error())
	}

	a.certs = append(a.certs, crt)

	a.listener = a.listenAndServeTLS(listener)
	a.running = true
	a.serve()
}

// Create base listener
func (a *TCPAcceptor) createBaseListener() net.Listener {
	// Create raw listener
	baseListener, err := net.Listen("tcp", a.addr)
	if err != nil {
		logger.Log.Fatalf("Failed to listen: %s", err.Error())
	}

	// Wrap listener in ProxyProto
	baseListener = &ProxyProtocolListener{Listener: baseListener, proxyProtocolEnabled: &a.proxyProtocol}

	return baseListener
}

// ListenAndServeTLS listens using tls
func (a *TCPAcceptor) listenAndServeTLS(listener net.Listener) net.Listener {

	tlsCfg := &tls.Config{Certificates: a.certs}
	tlsListener := tls.NewListener(listener, tlsCfg)

	return tlsListener
}

func (a *TCPAcceptor) EnableProxyProtocol() {
	a.proxyProtocol = true
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
			Conn:       conn,
			remoteAddr: conn.RemoteAddr(),
		}
	}
}

func (a *TCPAcceptor) IsRunning() bool {
	return a.running
}

func (a *TCPAcceptor) GetConfiguredAddress() string {
	return a.addr
}
