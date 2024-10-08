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
	"net"
	"time"

	"github.com/quic-go/quic-go"
)

var (
	// ErrListenerNotInitialized é retornado se o listener QUIC ainda não foi inicializado
	ErrListenerNotInitialized = errors.New("listener not initialized")
	// ErrConnClosed é retornado quando a conexão QUIC está fechada
	ErrConnClosed = errors.New("connection is closed")
)

// QuicAcceptor estrutura que representa um acceptor QUIC
type QuicAcceptor struct {
	addr     string
	listener *quic.Listener
	tlsConf  *tls.Config
	quicConf *quic.Config
}

// NewQuicAcceptor cria um novo QuicAcceptor
func NewQuicAcceptor(addr string, tlsConf *tls.Config, quicConf *quic.Config) *QuicAcceptor {
	return &QuicAcceptor{
		addr:    addr,
		tlsConf: tlsConf,
		quicConf: quicConf,
	}
}

// Listen inicia o listener QUIC para aceitar novas conexões
func (a *QuicAcceptor) Listen() error {
	// Criando um listener QUIC no endereço especificado
	listener, err := quic.ListenAddr(a.addr, a.tlsConf, a.quicConf)
	if err != nil {
		return err
	}
	a.listener = listener
	return nil
}

// Accept aceita novas conexões QUIC
func (a *QuicAcceptor) Accept() (quic.Connection, error) {
	if a.listener == nil {
		return nil, ErrListenerNotInitialized
	}

	// Contexto com timeout para aceitar conexões
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := a.listener.Accept(ctx)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// Close fecha o listener QUIC
func (a *QuicAcceptor) Close() error {
	if a.listener != nil {
		return a.listener.Close()
	}
	return nil
}

// QuicConnWrapper é um wrapper para uma conexão QUIC, permitindo o uso de deadlines
type QuicConnWrapper struct {
	conn quic.Connection
}

// NewQuicConnWrapper cria um novo wrapper para uma conexão QUIC
func NewQuicConnWrapper(conn quic.Connection) *QuicConnWrapper {
	return &QuicConnWrapper{conn: conn}
}

// Read lê dados da conexão com um deadline definido
func (q *QuicConnWrapper) Read(p []byte, readTimeout time.Duration) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), readTimeout)
	defer cancel()

	stream, err := q.conn.AcceptStream(ctx)
	if err != nil {
		return 0, err
	}

	return stream.Read(p)
}

// Write escreve dados na conexão com um deadline definido
func (q *QuicConnWrapper) Write(p []byte, writeTimeout time.Duration) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
	defer cancel()

	stream, err := q.conn.OpenStreamSync(ctx)
	if err != nil {
		return 0, err
	}

	return stream.Write(p)
}

// Close fecha a conexão QUIC
func (q *QuicConnWrapper) Close() error {
	return q.conn.CloseWithError(0, "closed")
}

// LocalAddr retorna o endereço local da conexão
func (q *QuicConnWrapper) LocalAddr() net.Addr {
	return q.conn.LocalAddr()
}

// RemoteAddr retorna o endereço remoto da conexão
func (q *QuicConnWrapper) RemoteAddr() net.Addr {
	return q.conn.RemoteAddr()
}
