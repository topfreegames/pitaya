package acceptor

import (
	"net"
	"sync"

	"github.com/mailgun/proxyproto"
	"github.com/topfreegames/pitaya/v2/logger"
)

// Listener is used to wrap an underlying listener,
// whose connections may be using the HAProxy Proxy Protocol.
// If the connection is using the protocol, the RemoteAddr() will return
// the correct client address.
type ProxyProtocolListener struct {
	net.Listener
	proxyProtocolEnabled *bool
}

// Accept waits for and returns the next connection to the listener.
func (p *ProxyProtocolListener) Accept() (net.Conn, error) {
	// Get the underlying connection
	conn, err := p.Listener.Accept()
	if err != nil {
		return nil, err
	}
	connP := &Conn{Conn: conn, proxyProtocolEnabled: p.proxyProtocolEnabled}
	if *p.proxyProtocolEnabled {
		err = connP.checkPrefix()
		if err != nil {
			return connP, err
		}
	}
	return connP, nil
}

// Conn is used to wrap and underlying connection which
// may be speaking the Proxy Protocol. If it is, the RemoteAddr() will
// return the address of the client instead of the proxy address.
type Conn struct {
	net.Conn
	dstAddr              *net.Addr
	srcAddr              *net.Addr
	once                 sync.Once
	proxyProtocolEnabled *bool
}

func (p *Conn) LocalAddr() net.Addr {
	if p.dstAddr != nil {
		return *p.dstAddr
	}
	return p.Conn.LocalAddr()
}

// RemoteAddr returns the address of the client if the proxy
// protocol is being used, otherwise just returns the address of
// the socket peer. If there is an error parsing the header, the
// address of the client is not returned, and the socket is closed.
// Once implication of this is that the call could block if the
// client is slow. Using a Deadline is recommended if this is called
// before Read()
func (p *Conn) RemoteAddr() net.Addr {
	if p.srcAddr != nil {
		return *p.srcAddr
	}
	return p.Conn.RemoteAddr()
}

func (p *Conn) checkPrefix() error {

	h, err := proxyproto.ReadHeader(p)
	if err != nil {
		logger.Log.Errorf("Failed to read Proxy Protocol TCP header: %s", err.Error())
		p.Close()
		return err

	} else if h.Source == nil {
		p.Close()
	} else {
		p.srcAddr = &h.Source
		p.dstAddr = &h.Destination
	}

	return nil
}
