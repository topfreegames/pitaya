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
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false, // Skip certificate verification for testing
	}

	quicConfig := &quic.Config{
		// QUIC specific settings can be placed here
	}

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start the QUIC connection
	conn, err := quic.DialAddr(ctx, "127.0.0.1:3250", tlsConfig, quicConfig)
	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}
	// Defer closing the connection using CloseWithError
	defer conn.CloseWithError(0, "client done")

	// Open a stream
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		log.Fatalf("Failed to open stream: %v", err)
	}

	// Send a message
	message := []byte("Hello from QUIC client!")
	_, err = stream.Write(message)
	if err != nil {
		log.Fatalf("Failed to write to stream: %v", err)
	}

	// Read the response
	response := make([]byte, 1024) // Adjust buffer size as needed
	n, err := stream.Read(response)
	if err != nil {
		log.Fatalf("Failed to read from stream: %v", err)
	}

	// Print the response
	fmt.Printf("Received response: %s\n", string(response[:n]))
}
