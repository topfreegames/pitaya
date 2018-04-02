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
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/topfreegames/pitaya/logger"
)

// WSAcceptor struct
type WSAcceptor struct {
	addr     string
	wsPath   string
	server   *http.Server
	connChan chan net.Conn
}

// NewWSAcceptor returns a new instance of WSAcceptor
func NewWSAcceptor(addr string, wsPath ...string) *WSAcceptor {
	w := &WSAcceptor{
		addr:     addr,
		connChan: make(chan net.Conn),
		wsPath:   "",
	}
	if len(wsPath) > 0 {
		w.wsPath = wsPath[0]
	}
	return w
}

// GetAddr returns the addr the acceptor will listen on
func (w *WSAcceptor) GetAddr() string {
	if w.server != nil {
		return w.server.Addr
	}
	return ""
}

// GetConnChan gets a connection channel
func (w *WSAcceptor) GetConnChan() chan net.Conn {
	return w.connChan
}

// ListenAndServe listens and serve in the specified addr
func (w *WSAcceptor) ListenAndServe() {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	http.HandleFunc("/"+strings.TrimPrefix(w.wsPath, "/"), func(rw http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(rw, r, nil)
		if err != nil {
			logger.Log.Errorf("Upgrade failure, URI=%s, Error=%s", r.RequestURI, err.Error())
			return
		}

		c, err := newWSConn(conn)
		if err != nil {
			logger.Log.Error(err)
			return
		}

		w.connChan <- c
	})

	w.server = &http.Server{
		Addr: w.addr,
	}

	defer w.Stop()

	if err := w.server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

// Stop stops the acceptor
func (w *WSAcceptor) Stop() {
	w.server.Close()
}

// wsConn is an adapter to t.Conn, which implements all t.Conn
// interface base on *websocket.Conn
type wsConn struct {
	conn   *websocket.Conn
	typ    int // message type
	reader io.Reader
}

// newWSConn return an initialized *wsConn
func newWSConn(conn *websocket.Conn) (*wsConn, error) {
	c := &wsConn{conn: conn}

	t, r, err := conn.NextReader()
	if err != nil {
		return nil, err
	}

	c.typ = t
	c.reader = r

	return c, nil
}

// Read reads data from the connection.
// Read can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
func (c *wsConn) Read(b []byte) (int, error) {
	n, err := c.reader.Read(b)
	if err != nil && err != io.EOF {
		return n, err
	} else if err == io.EOF {
		_, r, err := c.conn.NextReader()
		if err != nil {
			return 0, err
		}
		c.reader = r
	}

	return n, nil
}

// Write writes data to the connection.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
func (c *wsConn) Write(b []byte) (int, error) {
	err := c.conn.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		return 0, err
	}

	return len(b), nil
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (c *wsConn) Close() error {
	return c.conn.Close()
}

// LocalAddr returns the local network address.
func (c *wsConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (c *wsConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// SetDeadline sets the read and write deadlines associated
// with the connection. It is equivalent to calling both
// SetReadDeadline and SetWriteDeadline.
//
// A deadline is an absolute time after which I/O operations
// fail with a timeout (see type Error) instead of
// blocking. The deadline applies to all future and pending
// I/O, not just the immediately following call to Read or
// Write. After a deadline has been exceeded, the connection
// can be refreshed by setting a deadline in the future.
//
// An idle timeout can be implemented by repeatedly extending
// the deadline after successful Read or Write calls.
//
// A zero value for t means I/O operations will not time out.
func (c *wsConn) SetDeadline(t time.Time) error {
	if err := c.conn.SetReadDeadline(t); err != nil {
		return err
	}

	return c.conn.SetWriteDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
// A zero value for t means Read will not time out.
func (c *wsConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (c *wsConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}
