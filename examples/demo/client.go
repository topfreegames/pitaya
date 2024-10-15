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

    c := client.New(logrus.DebugLevel)

	err := c.ConnectToQUIC("localhost:3250", tlsConf, quicConf)

    if(err != nil) {
        log.Fatalf("Failed to connect to server: %v", err)
    }

   
    id, err := c.SendRequest("", []byte("Olá, server!"));

    if(err != nil) {
        log.Fatalf("Request failed: %v", err)
    }
    
    fmt.Printf("It's all ok: id %v!\n", id)

    for {}

}
