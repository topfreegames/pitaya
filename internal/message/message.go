// Copyright (c) nano Author and TFG Co. All Rights Reserved.
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

package message

import (
	"errors"
	"fmt"
	"strings"
)

// Type represents the type of message, which could be Request/Notify/Response/Push
type Type byte

// Message types
const (
	Request  Type = 0x00
	Notify   Type = 0x01
	Response Type = 0x02
	Push     Type = 0x03
)

const (
	errorMask            = 0x20
	gzipMask             = 0x10
	msgRouteCompressMask = 0x01
	msgTypeMask          = 0x07
	msgRouteLengthMask   = 0xFF
	msgHeadLength        = 0x02
)

var types = map[Type]string{
	Request:  "Request",
	Notify:   "Notify",
	Response: "Response",
	Push:     "Push",
}

var (
	routes = make(map[string]uint16) // route map to code
	codes  = make(map[uint16]string) // code map to route
)

// Errors that could be occurred in message codec
var (
	ErrWrongMessageType  = errors.New("wrong message type")
	ErrInvalidMessage    = errors.New("invalid message")
	ErrRouteInfoNotFound = errors.New("route info not found in dictionary")
)

// Message represents a unmarshaled message or a message which to be marshaled
type Message struct {
	Type       Type   // message type
	ID         uint   // unique id, zero while notify mode
	Route      string // route for locating service
	Data       []byte // payload
	compressed bool   // is message compressed
	Err        bool   // is an error message
}

// New returns a new message instance
func New(err ...bool) *Message {
	m := &Message{}
	if len(err) > 0 {
		m.Err = err[0]
	}
	return m
}

// String, implementation of fmt.Stringer interface
func (m *Message) String() string {
	return fmt.Sprintf("Type: %s, ID: %d, Route: %s, Compressed: %t, Error: %t, Data: %v, BodyLength: %d",
		types[m.Type],
		m.ID,
		m.Route,
		m.compressed,
		m.Err,
		m.Data,
		len(m.Data))
}

func routable(t Type) bool {
	return t == Request || t == Notify || t == Push
}

func invalidType(t Type) bool {
	return t < Request || t > Push

}

// SetDictionary set routes map which be used to compress route.
func SetDictionary(dict map[string]uint16) error {
	if dict == nil {
		return nil
	}

	for route, code := range dict {
		r := strings.TrimSpace(route)

		// duplication check
		if _, ok := routes[r]; ok {
			return fmt.Errorf("duplicated route(route: %s, code: %d)", r, code)
		}

		if _, ok := codes[code]; ok {
			return fmt.Errorf("duplicated route(route: %s, code: %d)", r, code)
		}

		// update map, using last value when key duplicated
		routes[r] = code
		codes[code] = r
	}

	return nil
}

// GetDictionary gets the routes map which is used to compress route.
func GetDictionary() map[string]uint16 {
	return routes
}

func (t *Type) String() string {
	return types[*t]
}
