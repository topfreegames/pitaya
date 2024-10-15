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

	// Initialize the QuicAcceptor
	quicAcceptor := acceptor.NewQuicAcceptor(":3250", tlsConf, quicConf)
	err := quicAcceptor.Listen()
	if err != nil {
		log.Fatalf("Failed to start QUIC server: %v", err)
	}
	defer quicAcceptor.Close()

	fmt.Println("QUIC server listening on port 3250")

	// Connection acceptance loop
	for {
		conn, err := quicAcceptor.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		// Handle connection in a separate goroutine
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
		// Read data from the client
		if err != nil {
			log.Printf("Error reading from stream: %v", err)
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
				fmt.Printf("message received from client: %v\n", string(m.Data))
			case packet.Heartbeat:
				fmt.Printf("got heartbeat from client!\n")
			}
		}

		// Send response to the client
		response := "Hello, QUIC client!"

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
			log.Printf("Error sending response: %v", err)
			return
		}
		fmt.Println("Response sent to client")
	}
	
}

func loadTLSCertificates() tls.Certificate {
	certPath := "../../../pkg/acceptor/fixtures/server.crt"
	keyPath := "../../../pkg/acceptor/fixtures/server.key"
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		panic(fmt.Sprintf("Error loading TLS certificates: %v", err))
	}
	return cert
}
