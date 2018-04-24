// Copyright (c) TFG Co. All Rights Reserved.
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

package client

import (
	"encoding/json"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/topfreegames/pitaya/internal/codec"
	"github.com/topfreegames/pitaya/internal/message"
	"github.com/topfreegames/pitaya/internal/packet"
	"github.com/topfreegames/pitaya/logger"
)

var (
	handshakeBuffer = `
{
    "sys": {
      "type": "golang-tcp",
      "version": "0.0.1",
      "rsa": {}
    },
    "user": {
    }
  };
`
)

// HandshakeSys struct
type HandshakeSys struct {
	Dict      map[string]uint16 `json:"dict"`
	Heartbeat int               `json:"heartbeat"`
}

// HandshakeData struct
type HandshakeData struct {
	Code int          `json:"code"`
	Sys  HandshakeSys `json:"sys"`
}

// Client struct
type Client struct {
	conn            net.Conn
	Connected       bool
	packetEncoder   codec.PacketEncoder
	packetDecoder   codec.PacketDecoder
	packetChan      chan *packet.Packet
	IncomingMsgChan chan *message.Message
	pendingChan     chan bool
	closeChan       chan struct{}
	nextID          uint32
	dataCompression bool
}

// New returns a new client
func New(logLevel logrus.Level) *Client {
	l := logrus.New()
	l.Formatter = &logrus.TextFormatter{}
	l.SetLevel(logLevel)

	logger.Log = l

	return &Client{
		Connected:       false,
		packetEncoder:   codec.NewPomeloPacketEncoder(),
		packetDecoder:   codec.NewPomeloPacketDecoder(),
		packetChan:      make(chan *packet.Packet, 10),
		IncomingMsgChan: make(chan *message.Message, 10),
		// 30 here is the limit of inflight messages
		// TODO this should probably be configurable
		pendingChan: make(chan bool, 30),
		dataCompression: true,
	}
}

func (c *Client) sendHandshakeRequest() error {
	p, err := c.packetEncoder.Encode(packet.Handshake, []byte(handshakeBuffer))
	if err != nil {
		return err
	}
	_, err = c.conn.Write(p)
	return err
}

func (c *Client) handleHandshakeResponse() error {
	buf := make([]byte, 2048)
	packets, err := c.readPackets(buf)
	if err != nil {
		return err
	}

	handshakePacket := packets[0]
	if handshakePacket.Type != packet.Handshake {
		return fmt.Errorf("got first packet from server that is not a handshake, aborting")
	}

	handshake := &HandshakeData{}
	err = json.Unmarshal(handshakePacket.Data, handshake)
	if err != nil {
		return err
	}
	logger.Log.Debug("got handshake from sv, data: %v", handshake)

	if handshake.Sys.Dict != nil {
		message.SetDictionary(handshake.Sys.Dict)
	}
	p, err := c.packetEncoder.Encode(packet.HandshakeAck, []byte{})
	if err != nil {
		return err
	}
	_, err = c.conn.Write(p)
	if err != nil {
		return err
	}

	go c.sendHeartbeats(handshake.Sys.Heartbeat)
	go c.handleServerMessages()
	go c.handlePackets()

	return nil
}

func (c *Client) handlePackets() {
	for {
		select {
		case p := <-c.packetChan:
			switch p.Type {
			case packet.Data:
				//handle data
				logger.Log.Debug("got data: %s", string(p.Data))
				m, err := message.Decode(p.Data)
				if err != nil {
					logger.Log.Errorf("error decoding msg from sv: %s", string(m.Data))
				}
				c.IncomingMsgChan <- m
				if m.Type == message.Response {
					<-c.pendingChan
				}
			}
		case <-c.closeChan:
			return
		}
	}
}

func (c *Client) readPackets(buf []byte) ([]*packet.Packet, error) {
	// listen for sv messages
	n, err := c.conn.Read(buf)
	if err != nil {
		return nil, err
	}
	data := buf[:n]
	packets, err := c.packetDecoder.Decode(data)
	if err != nil {
		logger.Log.Errorf("error decoding packet from server: %s", err.Error())
	}

	return packets, nil
}

func (c *Client) handleServerMessages() {
	buf := make([]byte, 2048)
	defer c.Disconnect()
	for c.Connected {
		packets, err := c.readPackets(buf)
		if err != nil {
			logger.Log.Error(err)
			break
		}

		for _, p := range packets {
			c.packetChan <- p
		}
	}
}

func (c *Client) sendHeartbeats(interval int) {
	t := time.NewTicker(time.Duration(interval) * time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			p, err := c.packetEncoder.Encode(packet.Heartbeat, []byte{})
			if err != nil {
				logger.Log.Errorf("error encoding heartbeat package: %s", err.Error())
			}
			_, err = c.conn.Write(p)
			if err != nil {
				logger.Log.Errorf("error sending heartbeat to sv: %s", err.Error())
			}
		case <-c.closeChan:
			return
		}
	}
}

// Disconnect disconnects the client
func (c *Client) Disconnect() {
	if c.Connected {
		c.Connected = false
		close(c.closeChan)
		c.conn.Close()
	}
}

// ConnectTo connects to the server at addr, for now the only supported protocol is tcp
// this methods blocks as it also handles the messages from the server
func (c *Client) ConnectTo(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	c.conn = conn

	err = c.sendHandshakeRequest()
	if err != nil {
		return err
	}

	err = c.handleHandshakeResponse()
	if err != nil {
		return err
	}

	c.closeChan = make(chan struct{})
	c.Connected = true

	return nil
}

// SendRequest sends a request to the server
func (c *Client) SendRequest(route string, data []byte) error {
	c.pendingChan <- true
	return c.sendMsg(message.Request, route, data)
}

// SendNotify sends a notify to the server
func (c *Client) SendNotify(route string, data []byte) error {
	return c.sendMsg(message.Notify, route, data)
}

// sendMsg sends the request to the server
func (c *Client) sendMsg(msgType message.Type, route string, data []byte) error {
	atomic.AddUint32(&c.nextID, 1)
	// TODO mount msg and encode
	m := message.Message{
		Type:  msgType,
		ID:    uint(c.nextID),
		Route: route,
		Data:  data,
		Err:   false,
	}
	encMsg, err := m.Encode(c.dataCompression)
	if err != nil {
		return err
	}
	p, err := c.packetEncoder.Encode(packet.Data, encMsg)
	if err != nil {
		return err
	}
	_, err = c.conn.Write(p)
	return err
}
