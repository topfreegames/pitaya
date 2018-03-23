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
	ErrMemberNotFound                 = errors.New("member not found in the group")
	ErrReplyShouldBePtr               = errors.New("reply must be a pointer")
	ErrRPCLocal                       = errors.New("RPC must be to a different server type")
	ErrSessionDuplication             = errors.New("session has existed in the current group")
	ErrSessionOnNotify                = errors.New("current session working on notify mode")
	ErrSessionNotFound                = errors.New("session not found")
	ErrSessionAlreadyBound            = errors.New("session is already bound to an uid")
	ErrRPCServerNotInitialized        = errors.New("rpc server is not running")
	ErrFrontSessionCantPushToFront    = errors.New("frontend session can't push to front")
	ErrServiceDiscoveryNotInitialized = errors.New("service discovery client is not initialized")
)
