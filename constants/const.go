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

package constants

const (
	_ int32 = iota
	// StatusStart status
	StatusStart
	// StatusHandshake status
	StatusHandshake
	// StatusWorking status
	StatusWorking
	// StatusClosed status
	StatusClosed
)

const (
	// SessionPushRoute is the route used for updating session
	SessionPushRoute = "sys.pushsession"

	// SessionBindRoute is the route used for binding session
	SessionBindRoute = "sys.bindsession"

	// KickRoute is the route used for kicking an user
	KickRoute = "sys.kick"
)

// SessionCtxKey is the context key where the session will be set
var SessionCtxKey = "session"

type propagateKey struct{}

// PropagateCtxKey is the context key where the content that will be
// propagated through rpc calls is set
var PropagateCtxKey = propagateKey{}

// SpanPropagateCtxKey is the key holding the opentracing spans inside
// the propagate key
var SpanPropagateCtxKey = "opentracing-span"

// PeerIdKey is the key holding the peer id to be sent over the context
var PeerIdKey = "peer.id"

// PeerServiceKey is the key holding the peer service to be sent over the context
var PeerServiceKey = "peer.service"
