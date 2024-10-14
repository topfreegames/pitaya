package main

import (
	"crypto/tls"
	"fmt"
	"log"

	"github.com/quic-go/quic-go"
	"github.com/topfreegames/pitaya/v3/pkg/acceptor"
)

func main() {
	tlsConf := &tls.Config{
		Certificates: []tls.Certificate{
			loadTLSCertificates(),
		},
	}

	quicConf := &quic.Config{
		Allow0RTT: true,
	}

	// Inicializar o QuicAcceptor
	quicAcceptor := acceptor.NewQuicAcceptor(":3250", tlsConf, quicConf)
	err := quicAcceptor.Listen()
	if err != nil {
		log.Fatalf("Falha ao iniciar o servidor QUIC: %v", err)
	}
	defer quicAcceptor.Close()

	fmt.Println("Servidor QUIC escutando na porta 3250")

	// Loop de aceitação de conexões
	for {
		conn, err := quicAcceptor.Accept()
		if err != nil {
			log.Printf("Erro ao aceitar conexão: %v", err)
			continue
		}

		// Manusear conexão em uma goroutine separada
		go handleConnection(conn)
	}
}

func handleConnection(conn quic.Connection) {

	// Aceitar um stream
	c := acceptor.NewQuicConnWrapper(conn)

	// Ler dados do cliente
	buff := make([]byte, 1024)
	n, err := c.Read(buff)
	if err != nil {
		log.Printf("Erro ao ler do stream: %v", err)
		return
	}
	fmt.Printf("Mensagem recebida: %s\n", string(buff[:n]))

	// Enviar resposta ao cliente
	response := "Olá, cliente QUIC!"
	_, err = c.Write([]byte(response))
	if err != nil {
		log.Printf("Erro ao enviar resposta: %v", err)
		return
	}
	fmt.Println("Resposta enviada ao cliente")
}

func loadTLSCertificates() tls.Certificate {
	certPath := "../../../pkg/acceptor/fixtures/server.crt"
	keyPath := "../../../pkg/acceptor/fixtures/server.key"
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		panic(fmt.Sprintf("Erro ao carregar certificados TLS: %v", err))
	}
	return cert
}