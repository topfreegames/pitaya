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
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/topfreegames/pitaya"
	"github.com/topfreegames/pitaya/internal/codec"
	"github.com/topfreegames/pitaya/internal/message"
	"github.com/topfreegames/pitaya/internal/packet"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/util/compression"
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

type pendingRequest struct {
	msg    *message.Message
	sentAt time.Time
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
	pendingRequests map[uint]*pendingRequest
	pendingReqMutex sync.Mutex
	requestTimeout  time.Duration
	closeChan       chan struct{}
	nextID          uint32
	messageEncoder  message.MessageEncoder
}

// New returns a new client
func New(logLevel logrus.Level, requestTimeout ...time.Duration) *Client {
	l := logrus.New()
	l.Formatter = &logrus.TextFormatter{}
	l.SetLevel(logLevel)

	logger.Log = l

	reqTimeout := 5 * time.Second
	if len(requestTimeout) > 0 {
		reqTimeout = requestTimeout[0]
	}

	return &Client{
		Connected:       false,
		packetEncoder:   codec.NewPomeloPacketEncoder(),
		packetDecoder:   codec.NewPomeloPacketDecoder(),
		packetChan:      make(chan *packet.Packet, 10),
		IncomingMsgChan: make(chan *message.Message, 10),
		pendingRequests: make(map[uint]*pendingRequest),
		requestTimeout:  reqTimeout,
		// 30 here is the limit of inflight messages
		// TODO this should probably be configurable
		pendingChan:    make(chan bool, 30),
		messageEncoder: message.NewEncoder(true),
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
	if compression.IsCompressed(handshakePacket.Data) {
		handshakePacket.Data, err = compression.InflateData(handshakePacket.Data)
		if err != nil {
			return err
		}
	}

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
	go c.pendingRequestsReaper()

	return nil
}

// pendingRequestsReaper delete timedout requests
func (c *Client) pendingRequestsReaper() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			toDelete := make([]*pendingRequest, 0)
			c.pendingReqMutex.Lock()
			for _, v := range c.pendingRequests {
				if time.Now().Sub(v.sentAt) > c.requestTimeout {
					toDelete = append(toDelete, v)
				}
			}
			for _, pendingReq := range toDelete {
				err := pitaya.Error(errors.New("request timeout"), "PIT-504")
				errMarshalled, _ := json.Marshal(err)
				// send a timeout to incoming msg chan
				m := &message.Message{
					Type:  message.Response,
					ID:    pendingReq.msg.ID,
					Route: pendingReq.msg.Route,
					Data:  errMarshalled,
					Err:   true,
				}
				delete(c.pendingRequests, pendingReq.msg.ID)
				<-c.pendingChan
				c.IncomingMsgChan <- m
			}
			c.pendingReqMutex.Unlock()
		case <-c.closeChan:
			return
		}
	}
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
				if m.Type == message.Response {
					c.pendingReqMutex.Lock()
					if _, ok := c.pendingRequests[m.ID]; ok {
						delete(c.pendingRequests, m.ID)
						<-c.pendingChan
					} else {
						continue // do not process msg for already timedout request
					}
					c.pendingReqMutex.Unlock()
				}
				c.IncomingMsgChan <- m
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
	return c.sendMsg(message.Request, route, data)
}

// SendNotify sends a notify to the server
func (c *Client) SendNotify(route string, data []byte) error {
	return c.sendMsg(message.Notify, route, data)
}

func (c *Client) buildPacket(msg message.Message) ([]byte, error) {
	encMsg, err := c.messageEncoder.Encode(&msg)
	if err != nil {
		return nil, err
	}
	p, err := c.packetEncoder.Encode(packet.Data, encMsg)
	if err != nil {
		return nil, err
	}

	return p, nil
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
	p, err := c.buildPacket(m)
	if msgType == message.Request {
		c.pendingReqMutex.Lock()
		if _, ok := c.pendingRequests[uint(c.nextID)]; !ok {
			c.pendingRequests[uint(c.nextID)] = &pendingRequest{
				msg:    &m,
				sentAt: time.Now(),
			}
			c.pendingChan <- true
		}
		c.pendingReqMutex.Unlock()
	}

	if err != nil {
		return err
	}
	_, err = c.conn.Write(p)
	return err
}
