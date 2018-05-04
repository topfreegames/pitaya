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
	ErrChangeDictionaryWhileRunning   = errors.New("you shouldn't change the dictionary while the app is already running")
	ErrChangeRouteWhileRunning        = errors.New("you shouldn't change routes while app is already running")
	ErrCloseClosedGroup               = errors.New("close closed group")
	ErrCloseClosedSession             = errors.New("close closed session")
	ErrClosedGroup                    = errors.New("group closed")
	ErrFrontSessionCantPushToFront    = errors.New("frontend session can't push to front")
	ErrIllegalUID                     = errors.New("illegal uid")
	ErrMemberNotFound                 = errors.New("member not found in the group")
	ErrNatsMessagesBufferSizeZero     = errors.New("pitaya.buffer.cluster.rpc.server.messages cant be zero")
	ErrNatsNoRequestTimeout           = errors.New("pitaya.cluster.rpc.client.nats.requesttimeout cant be empty")
	ErrNatsPushBufferSizeZero         = errors.New("pitaya.buffer.cluster.rpc.server.push cant be zero")
	ErrNilCondition                   = errors.New("pitaya/timer: nil condition")
	ErrNoNatsConnectionString         = errors.New("you have to provide a nats url")
	ErrNoServerTypeChosenForRPC       = errors.New("no server type chosen for sending RPC, send a full route in the format server.service.component")
	ErrNoServerWithID                 = errors.New("can't find any server with the provided ID")
	ErrNoServersAvailableOfType       = errors.New("no servers available of this type")
	ErrNoUIDBind                      = errors.New("you have to bind an UID to the session to do that")
	ErrNonsenseRPC                    = errors.New("you are making a rpc that may be processed locally, either specify a different server type or specify a server id")
	ErrNotifyOnRequest                = errors.New("tried to notify a request route")
	ErrOnCloseBackend                 = errors.New("onclose callbacks are not allowed on backend servers")
	ErrRPCClientNotInitialized        = errors.New("RPC client is not running")
	ErrRPCLocal                       = errors.New("RPC must be to a different server type")
	ErrRPCServerNotInitialized        = errors.New("RPC server is not running")
	ErrReplyShouldBeNotNull           = errors.New("reply must not be null")
	ErrReplyShouldBePtr               = errors.New("reply must be a pointer")
	ErrRequestOnNotify                = errors.New("tried to request a notify route")
	ErrRouterNotInitialized           = errors.New("router is not initialized")
	ErrServerNotFound                 = errors.New("server not found")
	ErrServiceDiscoveryNotInitialized = errors.New("service discovery client is not initialized")
	ErrSessionAlreadyBound            = errors.New("session is already bound to an uid")
	ErrSessionDuplication             = errors.New("session exists in the current group")
	ErrSessionNotFound                = errors.New("session not found")
	ErrSessionOnNotify                = errors.New("current session working on notify mode")
	ErrWrongValueType                 = errors.New("protobuf: convert on wrong type value")
	ErrInvalidCertificates            = errors.New("certificates must be exactly two")
	ErrTimeoutTerminatingBinaryModule = errors.New("timeout waiting to binary module to die")
	ErrFrontendTypeNotSpecified       = errors.New("for using SendPushToUsers from a backend server you have to specify a valid frontendType")
)
