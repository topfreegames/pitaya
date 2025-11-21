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

package repl

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/topfreegames/pitaya/v3/pkg/client"
)

func connect(logger Log, addr string, onMessageCallback func([]byte)) (err error) {
	if pClient != nil && pClient.ConnectedStatus() {
		return errors.New("already connected")
	}

	switch {
	case docsString != "":
		err = protoClient(logger, addr)
	default:
		logger.Println("Using json client")
		pClient = client.New(logrus.InfoLevel)
	}
	pClient.SetClientHandshakeData(handshake)

	if err != nil {
		return err
	}

	if err = tryConnect(addr); err != nil {
		logger.Println("Failed to connect!")
		return err
	}

	logger.Println("connected!")
	disconnectedCh = make(chan bool, 1)
	go readServerMessages(onMessageCallback)

	return nil
}

func setHandshake(logger Log, args []string) error {

	// Try first unserializing arguments as a json.
	err := json.Unmarshal([]byte(strings.Join(args[:], "")), &handshake)
	if err == nil {
		return nil
	}

	if len(args) == 2 {
		if args[0] != "version" && args[0] != "platform" && args[0] != "buildNumber" {
			return errors.New("invalid argument to sethandshake, expected version, platform or buildNumber")
		}
		switch arg := args[0]; arg {
		case "version":
			handshake.Sys.Version = args[1]
		case "platform":
			handshake.Sys.Platform = args[1]
		case "buildNumber":
			handshake.Sys.BuildNumber = args[1]
		}
		return nil
	}

	return errors.New("invalid number of arguments, expected 1 argument <json> or 2 arguments <variable> <value>")
}

func push(logger Log, args []string) error {
	if pClient != nil {
		return errors.New("use this command before connect")
	}

	if len(args) != 2 {
		return errors.New(`push should be in the format: push {route} {type}`)
	}

	route := args[0]
	pushType := args[1]

	if docsString == "" {
		logger.Println("Only for probuffer servers")
		return nil
	}

	pushInfo[route] = pushType

	return nil
}

func request(logger Log, args []string) error {
	if pClient == nil {
		return errors.New("not connected")
	}

	if !pClient.ConnectedStatus() {
		return errors.New("not connected")
	}

	if len(args) < 1 {
		return errors.New(`request should be in the format: request {route} [data]`)
	}

	route := args[0]

	var data []byte
	if len(args) > 1 {
		data = []byte(strings.Join(args[1:], ""))
	}

	_, err := pClient.SendRequest(route, data)
	if err != nil {
		return err
	}

	return nil
}

func notify(logger Log, args []string) error {
	if pClient == nil {
		return errors.New("not connected")
	}

	if !pClient.ConnectedStatus() {
		return errors.New("not connected")
	}

	if len(args) < 1 {
		return errors.New(`notify should be in the format: notify {route} [data]`)
	}

	route := args[0]
	var data []byte
	if len(args) > 1 {
		data = []byte(strings.Join(args[1:], ""))
	}

	if err := pClient.SendNotify(route, data); err != nil {
		return err
	}

	return nil
}

func disconnect() {
	if pClient.ConnectedStatus() {
		disconnectedCh <- true
		pClient.Disconnect()
	}
}

func routes(logger Log) error {
	if pClient == nil {
		return errors.New("client is not initialized")
	}

	if !pClient.ConnectedStatus() {
		return errors.New("not connected")
	}

	if protoClient, ok := pClient.(*client.ProtoClient); ok {
		info := protoClient.ExportInformation()
		if info != nil {
			for k, _ := range info.Commands {
				logger.Println(k)
			}
		}

	} else {
		return errors.New("only ProtoClient implements the command `routes`")
	}

	return nil
}
