package main

import (
    "crypto/tls"
    "fmt"
    "log"
    "time"
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

	err := c.ConnectToQUIC("localhost:3250", quicConf, tlsConf)

    if(err != nil) {
        log.Fatalf("Failed to connect to server: %v", err)
    }

   
    id, err := c.SendRequest("connector.getsessiondata", []byte{});

    if(err != nil) {
        log.Fatalf("Request failed: %v", err)
    }
    
    fmt.Printf("Request was sent to server: reqUid %v!\n", id)

    id, err = c.SendRequest("connector.setsessiondata", []byte("{\"Data\":{\"ipversion\":\"ipv4\", \"novoDado\":\"QuicClient\"}}"));

    if(err != nil) {
        log.Fatalf("Request failed: %v", err)
    }
    
    fmt.Printf("Request was sent to server: reqUid %v!\n", id)

    time.Sleep(2 * time.Second);

    id, err = c.SendRequest("connector.getsessiondata", []byte{});

    if(err != nil) {
        log.Fatalf("Request failed: %v", err)
    }
    
    fmt.Printf("Request was sent to server: reqUid %v!\n", id)

    id, err = c.SendRequest("requestor.room.entry", []byte{});

    if(err != nil) {
        log.Fatalf("Request failed: %v", err)
    }
    
    fmt.Printf("Request was sent to server: reqUid %v!\n", id)

    time.Sleep(2 * time.Second)

    id, err = c.SendRequest("requestor.room.join", []byte{});

    if(err != nil) {
        log.Fatalf("Request failed: %v", err)
    }
    
    fmt.Printf("Request was sent to server: reqUid %v!\n", id)

    time.Sleep(65 * time.Second);

    c.Disconnect();

}
