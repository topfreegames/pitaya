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

import "errors"

// Errors that could be occurred during message handling.
var (
	ErrBrokenPipe                     = errors.New("broken low-level pipe")
	ErrBufferExceed                   = errors.New("session send buffer exceed")
	ErrCloseClosedGroup               = errors.New("close closed group")
	ErrCloseClosedSession             = errors.New("close closed session")
	ErrClosedGroup                    = errors.New("group closed")
	ErrFrontSessionCantPushToFront    = errors.New("frontend session can't push to front")
	ErrMemberNotFound                 = errors.New("member not found in the group")
	ErrNotifyOnRequest                = errors.New("tried to notify a request route")
	ErrRPCLocal                       = errors.New("RPC must be to a different server type")
	ErrRPCServerNotInitialized        = errors.New("RPC server is not running")
	ErrReplyShouldBePtr               = errors.New("reply must be a pointer")
	ErrRequestOnNotify                = errors.New("tried to request a notify route")
	ErrServiceDiscoveryNotInitialized = errors.New("service discovery client is not initialized")
	ErrSessionAlreadyBound            = errors.New("session is already bound to an uid")
	ErrSessionDuplication             = errors.New("session has existed in the current group")
	ErrSessionNotFound                = errors.New("session not found")
	ErrSessionOnNotify                = errors.New("current session working on notify mode")
	ErrNoServerTypeChosenForRPC       = errors.New("no server type chosen for sending RPC, send a full route in the format server.service.component")
	ErrServerNotFound                 = errors.New("server not found")
	ErrNonsenseRPC                    = errors.New("you are making a rpc that may be processed locally, either specify a different server type or specify a server id")
	ErrNoUIDBind                      = errors.New("you have to bind an UID to the session to use groups")
	ErrOnCloseBackend                 = errors.New("onclose callbacks are not allowed on backend servers")
)
