package pitaya

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/dop251/goja"
	pitayaclient "github.com/topfreegames/pitaya/v2/client"
	pitayamessage "github.com/topfreegames/pitaya/v2/conn/message"
	"github.com/topfreegames/pitaya/v2/session"
	"go.k6.io/k6/js/modules"
)

// Response is the type of the response returned by the server
type Response interface{}

// Client is the pitaya client
// It is used to connect to a pitaya server and send requests and notifies
// It is also used to consume pushes
type Client struct {
	vu             modules.VU
	client         pitayaclient.PitayaClient
	handshake      *session.HandshakeData
	responsesMutex sync.Mutex
	responses      map[uint]chan []byte
	pushesMutex    sync.Mutex
	pushes         map[string]chan []byte
	timeout        time.Duration
}

// Connect connects to the server
// addr is the address of the server to connect to
func (c *Client) Connect(addr string) error { //TODO: tls Options
	vuState := c.vu.State()

	if vuState == nil {
		return errors.New("connecting to a pitaya server in the init context is not supported")
	}

	err := c.client.ConnectTo(addr)
	if err != nil {
		return err
	}
	go c.listen()

	return err
}

// IsConnected returns true if the client is connected to the server
func (c *Client) IsConnected() bool {
	res := reflect.ValueOf(c.client).Elem().FieldByName("Connected")
	return res.Bool()
}

// ConsumePush will return a promise that will be resolved when a push is received on the given route.
// The promise will be rejected if the timeout is reached before a push is received.
// The promise will be resolved with the push data.
func (c *Client) ConsumePush(route string, timeoutMs int) *goja.Promise {
	promise, resolve, reject := c.makeHandledPromise()
	ch := c.getPushChannelForRoute(route)
	go func() {
		select {
		case data := <-ch:
			var ret Response
			if err := json.Unmarshal(data, &ret); err != nil {
				err = fmt.Errorf("Error unmarshaling response: %s", err)
				reject(err)
				return
			}
			resolve(ret)
			return
		case <-time.After(time.Duration(timeoutMs) * time.Millisecond):
			reject(fmt.Errorf("Timeout waiting for push on route %s", route))
			return
		}
	}()
	return promise
}

// Notify sends a notify to the server
// route is the route to send the notify to
// msg is the message to send
// returns an error if the notify could not be sent
func (c *Client) Notify(route string, msg interface{}) error {
	m := msg
	if m == nil {
		m = map[string]interface{}{}
	}
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return c.client.SendNotify(route, data)
}

// Request sends a request to the server
// route is the route to send the request to
// msg is the message to send
// returns a promise that will be resolved when the response is received
// the promise will be rejected if the timeout is reached before a response is received
func (c *Client) Request(route string, msg interface{}) *goja.Promise { // TODO: add custom timeout
	m := msg
	if m == nil {
		m = map[string]interface{}{}
	}
	promise, resolve, reject := c.makeHandledPromise()
	data, err := json.Marshal(m)
	if err != nil {
		reject(err)
		return promise
	}

	mid, err := c.client.SendRequest(route, data)
	if err != nil {
		reject(err)
		return promise
	}
	responseChan := c.getResponseChannelForID(mid)
	go func() {
		select {
		case responseData := <-responseChan:
			var ret Response
			if err := json.Unmarshal(responseData, &ret); err != nil {
				resolve(responseData)
				return
			}
			resolve(ret)
			return
		case <-time.After(c.timeout):
			reject(fmt.Errorf("Timeout waiting for response on route %s", route))
		}
	}()
	return promise
}

// Disconnect disconnects from the server
func (c *Client) Disconnect() {
	c.client.Disconnect()
}

func (c *Client) listen() {
	channel := c.client.MsgChannel()
	go func() {
		for m := range channel {
			switch m.Type {
			case pitayamessage.Response:
				ch := c.getResponseChannelForID(m.ID)
				ch <- m.Data
				c.removeResponseChannelForID(m.ID)
			case pitayamessage.Push:
				ch := c.getPushChannelForRoute(m.Route)
				ch <- m.Data
			default:
				panic("Unknown message type")
			}
		}
	}()
}

func (c *Client) getResponseChannelForID(id uint) chan []byte {
	c.responsesMutex.Lock()
	defer c.responsesMutex.Unlock()
	if _, ok := c.responses[id]; !ok {
		c.responses[id] = make(chan []byte, 100)
	}

	return c.responses[id]
}

func (c *Client) removeResponseChannelForID(id uint) {
	c.responsesMutex.Lock()
	defer c.responsesMutex.Unlock()

	delete(c.responses, id)
}

func (c *Client) getPushChannelForRoute(route string) chan []byte {
	c.pushesMutex.Lock()
	defer c.pushesMutex.Unlock()
	if _, ok := c.pushes[route]; !ok {
		c.pushes[route] = make(chan []byte, 100)
	}

	return c.pushes[route]
}

// makeHandledPromise will create a promise and return its resolve and reject methods,
// wrapped in such a way that it will block the eventloop from exiting before they are
// called even if the promise isn't resolved by the time the current script ends executing.
func (c *Client) makeHandledPromise() (*goja.Promise, func(interface{}), func(interface{})) {
	runtime := c.vu.Runtime()
	callback := c.vu.RegisterCallback()
	p, resolve, reject := runtime.NewPromise()

	return p, func(i interface{}) {
			// more stuff
			callback(func() error {
				resolve(i)
				return nil
			})
		}, func(i interface{}) {
			// more stuff
			callback(func() error {
				reject(i)
				return nil
			})
		}
}
