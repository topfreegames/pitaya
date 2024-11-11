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
	"errors"
	"fmt"
	"net"
	"time"
	"io"

	"github.com/quic-go/quic-go"
	"github.com/topfreegames/pitaya/v3/pkg/conn/codec"
	"github.com/topfreegames/pitaya/v3/pkg/constants"
)

var (
	// ErrListenerNotInitialized is returned if the QUIC listener is not initialized
	ErrListenerNotInitialized = errors.New("listener not initialized")
	// ErrConnClosed is returned when the QUIC connection is closed
	ErrConnClosed = errors.New("connection is closed")
)

// QuicAcceptor represents a QUIC acceptor
type QuicAcceptor struct {
	addr     string
	connChan chan PlayerConn
	listener *quic.Listener
	running  bool
	tlsConf  *tls.Config
	quicConf *quic.Config
}

// NewQuicAcceptor creates a new QuicAcceptor
func NewQuicAcceptor(addr string, tlsConf *tls.Config, quicConf *quic.Config) *QuicAcceptor {
	return &QuicAcceptor{
		addr:     addr,
		tlsConf:  tlsConf,
		quicConf: quicConf,
		connChan: make(chan PlayerConn),
	}
}

func (a *QuicAcceptor) GetAddr() string {
	if a.listener != nil {
		return a.listener.Addr().String()
	}
	return ""
}

// GetConnChan gets a connection channel
func (a *QuicAcceptor) GetConnChan() chan PlayerConn {
	return a.connChan
}

// Listen starts the QUIC listener to accept new connections
func (a *QuicAcceptor) Listen() error {
	// Creating a QUIC listener at the specified address
	a.running = true
	listener, err := quic.ListenAddr(a.addr, a.tlsConf, a.quicConf)
	if err != nil {
		return err
	}
	a.listener = listener
	return nil
}

// Accept accepts new QUIC connections
func (a *QuicAcceptor) Accept() (quic.Connection, error) {
	if a.listener == nil { // Correct comparison
		return nil, ErrListenerNotInitialized
	}

	conn, err := a.listener.Accept(context.Background())
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// Close closes the QUIC listener
func (a *QuicAcceptor) Close() error {
	if a.listener != nil { // Correct comparison
		return a.listener.Close()
	}
	return nil
}

// EnableProxyProtocol not implemented for QUIC, keep as No-op or implement as needed
func (a *QuicAcceptor) EnableProxyProtocol() {
	// No-op: Implement this method if needed for Proxy Protocol support
}

func (a *QuicAcceptor) IsRunning() bool {
	return a.running
}

func (a *QuicAcceptor) GetConfiguredAddress() string {
	return a.addr
}

func (a *QuicAcceptor) ListenAndServe() {
	// Start the QUIC listener
	if err := a.Listen(); err != nil {
		fmt.Printf("Failed to start QUIC listener: %s\n", err)
		return
	}

	// Loop to continuously accept connections
	for a.IsRunning() {
		conn, err := a.Accept()
		if err != nil {
			if errors.Is(err, ErrListenerNotInitialized) {
				fmt.Println("Listener not initialized")
				continue
			}
			fmt.Printf("Failed to accept connection: %s\n", err)
			continue
		}

		// Send the connection to the channel for further processing
		go func(c quic.Connection) {
			playerConn := NewQuicConnWrapper(c, 100*time.Millisecond, 100*time.Millisecond)
			a.connChan <- playerConn
		}(conn)
	}
}

// QuicConnWrapper is a wrapper for a QUIC connection, allowing the use of deadlines
type QuicConnWrapper struct {
	conn quic.Connection
	stream quic.Stream
	writeTimeout time.Duration
	readTimeout time.Duration
}

// NewQuicConnWrapper creates a new wrapper for a QUIC connection
func NewQuicConnWrapper(conn quic.Connection, writeTimeout, readTimeout time.Duration) *QuicConnWrapper {
	return &QuicConnWrapper{conn: conn,
		writeTimeout: writeTimeout,
		readTimeout:  readTimeout,
	}
}

// Read reads data from the QUIC connection with a timeout
func (q *QuicConnWrapper) Read(b []byte) (n int, err error) {
	if q.stream == nil {
		ctx, cancel := context.WithTimeout(context.Background(), q.readTimeout)
		defer cancel()
		stream, err := q.conn.AcceptStream(ctx)
		if err != nil {
			return 0, err
		}
		q.stream = stream
	}

	n, err = q.stream.Read(b)
	if err == io.EOF {
		q.stream = nil
	} else if err != nil {
		return n, err
	}

	return n, err
}

// Write writes data to the connection with a defined deadline
func (q *QuicConnWrapper) Write(b []byte) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), q.writeTimeout)
	defer cancel()

	stream, err := q.conn.OpenStreamSync(ctx)
	if err != nil {
		return 0, err
	}

	return stream.Write(b)
}

// Close closes the QUIC connection
func (q *QuicConnWrapper) Close() error {
	return q.conn.CloseWithError(0, "closed")
}

// LocalAddr returns the local address of the connection
func (q *QuicConnWrapper) LocalAddr() net.Addr {
	return q.conn.LocalAddr()
}

// RemoteAddr returns the remote address of the connection
func (q *QuicConnWrapper) RemoteAddr() net.Addr {
	return q.conn.RemoteAddr()
}

// GetNextMessage reads the next message available in the stream
func (q *QuicConnWrapper) GetNextMessage() (b []byte, err error) {
    stream, err := q.conn.AcceptStream(context.Background())
    if err != nil {
        return nil, err
    }
    header, err := io.ReadAll(io.LimitReader(stream, codec.HeadLength))
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
    msgData, err := io.ReadAll(io.LimitReader(stream, int64(msgSize)))
    if err != nil {
        return nil, err
    }
    if len(msgData) < msgSize {
        return nil, constants.ErrReceivedMsgSmallerThanExpected
    }
    return append(header, msgData...), nil
}

func (q *QuicConnWrapper) SetDeadline(t time.Time) error {
	return nil
}

func (q *QuicConnWrapper) SetReadDeadline(t time.Time) error {
	return nil
}

func (q *QuicConnWrapper) SetWriteDeadline(t time.Time) error {
	return nil
}

func (a *QuicAcceptor) Stop() {
	a.running = false
	a.listener.Close()
}
