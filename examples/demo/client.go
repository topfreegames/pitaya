package main

import (
    "crypto/tls"
    "fmt"
    "log"
    "github.com/sirupsen/logrus"

    "github.com/quic-go/quic-go"
    "github.com/topfreegames/pitaya/v3/pkg/client"
)

func main() {
    // Define TLS configurations
    tlsConf := &tls.Config{
        InsecureSkipVerify: true, // Only for testing, do not use in production
    }

    // Define QUIC configurations (can be nil if not needed)
    quicConf := &quic.Config{}

    c := client.New(logrus.InfoLevel)

	conn, err := c.ConnectToQUIC("localhost:3250", tlsConf, quicConf)

    if(err != nil) {
        log.Fatalf("Failed to connect to server: %v", err)
    }
    
    msg := "Hello from QUIC client!"
    fmt.Printf("Sending message: %v\n", msg)
    _, err = conn.Write([]byte(msg))

    if(err != nil) {
        log.Fatalf("Failed to send data: %v\n", err)
    }

    fmt.Printf("it's all ok\n")

    buffer := make([]byte, 1024)
    n, err := conn.Read(buffer)

    if err != nil {
        log.Fatalf("Failed to read response data: %v\n", err)
    }

    fmt.Printf("Response from server: %s\n", string(buffer[:n]))
}
