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

package util

import (
	"bytes"
	"encoding/gob"
	"errors"
	"os"
	"reflect"
	"runtime"
	"strings"

	"github.com/topfreegames/pitaya/internal/message"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/serialize"
)

var log = logger.Log

// Pcall calls a method that returns an interface and an error and recovers in case of panic
func Pcall(method reflect.Method, args []reflect.Value) (rets interface{}, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Errorf("pitaya/dispatch: %v", rec)
			log.Error(Stack())
			if s, ok := rec.(string); ok {
				err = errors.New(s)
			} else {
				err = errors.New("rpc call internal error")
			}
		}
	}()

	r := method.Func.Call(args)
	// r can have 0 length in case of notify handlers
	// otherwise it will have 1 (an interface) or 2 outputs: an interface and an error
	if len(r) == 1 {
		rets = r[0].Interface()
	}
	if len(r) == 2 {
		if v := r[1].Interface(); v != nil {
			err = v.(error)
			if err != nil {
				log.Error(err.Error())
			}
		} else {
			rets = r[0].Interface()
		}
	}
	return
}

// Pinvoke call handler with protected
func Pinvoke(fn func()) {
	defer func() {
		if err := recover(); err != nil {
			logger.Log.Errorf("pitaya/invoke: %v", err)
			logger.Log.Error(Stack())
		}
	}()

	fn()
}

// SliceContainsString returns true if a slice contains the string
func SliceContainsString(slice []string, str string) bool {
	for _, value := range slice {
		if value == str {
			return true
		}
	}
	return false
}

// SerializeOrRaw serializes the interface if its not an array of bytes already
func SerializeOrRaw(serializer serialize.Serializer, v interface{}) ([]byte, error) {
	if data, ok := v.([]byte); ok {
		return data, nil
	}
	data, err := serializer.Marshal(v)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// GobEncode encodes interfaces with gob
func GobEncode(args ...interface{}) ([]byte, error) {
	buf := bytes.NewBuffer([]byte(nil))
	if err := gob.NewEncoder(buf).Encode(args); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobEncodeSingle encodes a single value with goc
func GobEncodeSingle(arg interface{}) ([]byte, error) {
	buf := bytes.NewBuffer([]byte(nil))
	if err := gob.NewEncoder(buf).Encode(arg); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GobDecode decodes a gob encoded binary
func GobDecode(reply interface{}, data []byte) error {
	return gob.NewDecoder(bytes.NewReader(data)).Decode(reply)
}

// FileExists tells if a file exists
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

// Stack prints the stack trace
func Stack() string {
	buf := make([]byte, 10000)
	n := runtime.Stack(buf, false)
	buf = buf[:n]

	s := string(buf)

	// skip pitaya frames lines
	const skip = 7
	count := 0
	index := strings.IndexFunc(s, func(c rune) bool {
		if c != '\n' {
			return false
		}
		count++
		return count == skip
	})
	return s[index+1:]
}

// GetErrorPayload creates and serializes an error payload
func GetErrorPayload(serializer serialize.Serializer, err error) ([]byte, error) {
	errPayload := &protos.ErrorPayload{
		Code:   500,
		Reason: err.Error(),
	}
	return SerializeOrRaw(serializer, errPayload)
}

// ConvertProtoToMessageType converts a protos.MsgType to a message.Type
func ConvertProtoToMessageType(protoMsgType protos.MsgType) message.Type {
	var msgType message.Type
	switch protoMsgType {
	case protos.MsgType_MsgRequest:
		msgType = message.Request
	case protos.MsgType_MsgNotify:
		msgType = message.Notify
	}
	return msgType
}
