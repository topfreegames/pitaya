package main

import (
	"crypto/tls"
	"fmt"
	"log"

	"github.com/quic-go/quic-go"
	"github.com/topfreegames/pitaya/v3/pkg/acceptor"
	"github.com/topfreegames/pitaya/v3/pkg/conn/codec"
	"github.com/topfreegames/pitaya/v3/pkg/conn/message"
	"github.com/topfreegames/pitaya/v3/pkg/conn/packet"
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

	packetDecoder := codec.NewPomeloPacketDecoder()
	packetEncoder := codec.NewPomeloPacketEncoder()
	messageEncoder := message.NewMessagesEncoder(false)

	// Aceitar um stream
	c := acceptor.NewQuicConnWrapper(conn)

	// Ler dados do cliente
	for {
		buff := make([]byte, 1024)
		n, err := c.Read(buff)
		if err != nil {
			log.Printf("Erro ao ler do stream: %v", err)
			return
		}

		packets, err := packetDecoder.Decode(buff[:n])
		if err != nil {
			fmt.Printf("%v", err)
			break
		}

		for _, p := range packets {
			switch p.Type {
			case packet.Data:
				m, err := message.Decode(p.Data)
				if err != nil {
					fmt.Printf("error decoding msg from sv: %v\n", string(m.Data))
				}
				fmt.Printf("got data from client: %v\n", string(m.Data))
			case packet.Heartbeat:
				fmt.Printf("got heartbeat from client!\n")
			}
		}

		// Enviar resposta ao cliente
		response := "Olá, cliente QUIC!"

		// buildPacket
		m := message.Message{
			Type:  message.Response,
			ID:    2,
			Route: "",
			Data:  []byte(response),
			Err:   false,
		}

		encMsg, err := messageEncoder.Encode(&m)

		fmt.Printf("message encoded\n")
		if err != nil {
			return
		}
		p, err := packetEncoder.Encode(packet.Data, encMsg)

		fmt.Printf("packet encoded\n")
		if err != nil {
			return
		}

		_, err = c.Write([]byte(p))
		if err != nil {
			log.Printf("Erro ao enviar resposta: %v", err)
			return
		}
		fmt.Println("Resposta enviada ao cliente")
	}
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