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

package component

import (
	"errors"
	"reflect"

	"github.com/topfreegames/pitaya/conn/message"
	"github.com/topfreegames/pitaya/constants"
)

type (
	//Handler represents a message.Message's handler's meta information.
	Handler struct {
		Receiver    reflect.Value  // receiver of method
		Method      reflect.Method // method stub
		Type        reflect.Type   // low-level type of method
		IsRawArg    bool           // whether the data need to serialize
		MessageType message.Type   // handler allowed message type (either request or notify)
	}

	//Remote represents remote's meta information.
	Remote struct {
		Receiver reflect.Value  // receiver of method
		Method   reflect.Method // method stub
		HasArgs  bool           // if remote has no args we won't try to serialize received data into arguments
		Type     reflect.Type   // low-level type of method
	}

	// Service implements a specific service, some of it's methods will be
	// called when the correspond events is occurred.
	Service struct {
		Name     string              // name of service
		Type     reflect.Type        // type of the receiver
		Receiver reflect.Value       // receiver of methods for the service
		Handlers map[string]*Handler // registered methods
		Remotes  map[string]*Remote  // registered remote methods
		Options  options             // options
	}
)

// NewService creates a new service
func NewService(comp Component, opts []Option) *Service {
	s := &Service{
		Type:     reflect.TypeOf(comp),
		Receiver: reflect.ValueOf(comp),
	}

	// apply options
	for i := range opts {
		opt := opts[i]
		opt(&s.Options)
	}
	if name := s.Options.name; name != "" {
		s.Name = name
	} else {
		s.Name = reflect.Indirect(s.Receiver).Type().Name()
	}

	return s
}

// ExtractHandler extract the set of methods from the
// receiver value which satisfy the following conditions:
// - exported method of exported type
// - one or two arguments
// - the first argument is context.Context
// - the second argument (if it exists) is []byte or a pointer
// - zero or two outputs
// - the first output is [] or a pointer
// - the second output is an error
func (s *Service) ExtractHandler() error {
	typeName := reflect.Indirect(s.Receiver).Type().Name()
	if typeName == "" {
		return errors.New("no service name for type " + s.Type.String())
	}
	if !isExported(typeName) {
		return errors.New("type " + typeName + " is not exported")
	}

	// Install the methods
	s.Handlers = suitableHandlerMethods(s.Type, s.Options.nameFunc)

	if len(s.Handlers) == 0 {
		str := ""
		// To help the user, see if a pointer receiver would work.
		method := suitableHandlerMethods(reflect.PtrTo(s.Type), s.Options.nameFunc)
		if len(method) != 0 {
			str = "type " + s.Name + " has no exported methods of handler type (hint: pass a pointer to value of that type)"
		} else {
			str = "type " + s.Name + " has no exported methods of handler type"
		}
		return errors.New(str)
	}

	for i := range s.Handlers {
		s.Handlers[i].Receiver = s.Receiver
	}

	return nil
}

// ExtractRemote extract the set of methods from the
// receiver value which satisfy the following conditions:
// - exported method of exported type
// - two return values
// - the first return implements protobuf interface
// - the second return is an error
func (s *Service) ExtractRemote() error {
	typeName := reflect.Indirect(s.Receiver).Type().Name()
	if typeName == "" {
		return errors.New("no service name for type " + s.Type.String())
	}
	if !isExported(typeName) {
		return errors.New("type " + typeName + " is not exported")
	}

	// Install the methods
	s.Remotes = suitableRemoteMethods(s.Type, s.Options.nameFunc)

	if len(s.Remotes) == 0 {
		str := ""
		// To help the user, see if a pointer receiver would work.
		method := suitableRemoteMethods(reflect.PtrTo(s.Type), s.Options.nameFunc)
		if len(method) != 0 {
			str = "type " + s.Name + " has no exported methods of remote type (hint: pass a pointer to value of that type)"
		} else {
			str = "type " + s.Name + " has no exported methods of remote type"
		}
		return errors.New(str)
	}

	for i := range s.Remotes {
		s.Remotes[i].Receiver = s.Receiver
	}
	return nil
}

// ValidateMessageType validates a given message type against the handler's one
// and returns an error if it is a mismatch and a boolean indicating if the caller should
// exit in the presence of this error or not.
func (h *Handler) ValidateMessageType(msgType message.Type) (exitOnError bool, err error) {
	if h.MessageType != msgType {
		switch msgType {
		case message.Request:
			err = constants.ErrRequestOnNotify
			exitOnError = true

		case message.Notify:
			err = constants.ErrNotifyOnRequest
		}
	}
	return
}
