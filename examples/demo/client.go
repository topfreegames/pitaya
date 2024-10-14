package main

import (
    "context"
    "crypto/tls"
    "fmt"
    "log"
    "time"

    "github.com/quic-go/quic-go"
)

func main() {
    // Define TLS configurations
    tlsConf := &tls.Config{
        InsecureSkipVerify: true, // Only for testing, do not use in production
    }

    // Create context
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // Define QUIC configurations (can be nil if not needed)
    quicConf := &quic.Config{}

    // Connect to the server using QUIC
    addr := "localhost:3250"
    session, err := quic.DialAddr(ctx, addr, tlsConf, quicConf)
    if err != nil {
        log.Fatalf("Failed to connect to server: %v", err)
    }
    defer session.CloseWithError(0, "connection closed")

    // Open a stream to send data
    stream, err := session.OpenStreamSync(ctx)
    if err != nil {
        log.Fatalf("Failed to open stream: %v", err)
    }
    defer stream.Close()

    // Send message to the server
    message := "Hello from QUIC client!"
    fmt.Printf("Sending message: %s\n", message)
    _, err = stream.Write([]byte(message))
    if err != nil {
        log.Fatalf("Failed to send data: %v", err)
    }

    // Receive response from the server
    buffer := make([]byte, 1024)
    n, err := stream.Read(buffer)
    if err != nil {
        log.Fatalf("Failed to read response: %v", err)
    }

    fmt.Printf("Response from server: %s\n", string(buffer[:n]))
}
