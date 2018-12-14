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
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io/ioutil"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	protobuf "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/sirupsen/logrus"
	"github.com/topfreegames/pitaya/conn/message"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/protos"
)

type ClientInterface interface {
	ConnectTo(addr string) error
	ConnectToTLS(addr string, skipVerify bool) error
	Disconnect()
	SendNotify(route string, data []byte) error
	SendRequest(route string, data []byte) (uint, error)
	ConnectedStatus() bool
	MsgChannel() chan *message.Message
}

type Command struct {
	input     string // input command name
	output    string // output command name
	inputMsg  *dynamic.Message
	outputMsg *dynamic.Message
}

type ProtoBufferInfo struct {
	Commands map[string]*Command
}

type ProtoClient struct {
	Client
	descriptorsNames map[string]bool
	info             ProtoBufferInfo
	docsRoute        string
	descriptorsRoute string
	IncomingMsgChan  chan *message.Message
	expectedInput    *dynamic.Message
	ready            bool
	closeChan        chan bool
}

func (pc *ProtoClient) MsgChannel() chan *message.Message {
	return pc.IncomingMsgChan
}

func unpackDescriptor(compressedDescriptor []byte) (*protobuf.FileDescriptorProto, error) {
	r, err := gzip.NewReader(bytes.NewReader(compressedDescriptor))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var fileDescriptorProto protobuf.FileDescriptorProto

	if err = proto.Unmarshal(b, &fileDescriptorProto); err != nil {
		return nil, err
	}

	return &fileDescriptorProto, nil
}

func (pc *ProtoClient) buildProtosFromDescriptor(descriptorArray []*protobuf.FileDescriptorProto) error {

	descriptorsMap := make(map[string]*dynamic.Message)

	desc, err := desc.CreateFileDescriptors(descriptorArray)
	if err != nil {
		return err
	}
	// log

	for name := range pc.descriptorsNames {
		for _, v := range desc {
			menssage := v.FindMessage(name)
			if menssage != nil {
				descriptorsMap[name] = dynamic.NewMessage(menssage)
			}
		}
	}

	for name, cmd := range pc.info.Commands {
		if msg, ok := descriptorsMap[cmd.input]; ok {
			pc.info.Commands[name].inputMsg = msg
		}
		if msg, ok := descriptorsMap[cmd.output]; ok {
			pc.info.Commands[name].outputMsg = msg
		}
	}

	return nil
}

func getOutputInputNames(command map[string]interface{}) (string, string, error) {
	outputName := ""
	inputName := ""

	in := command["input"]
	inputDocs, ok := in.(map[string]interface{})
	if ok {
		for k := range inputDocs {
			if strings.Contains(k, "proto") {
				inputName = strings.Replace(k, "*", "", 1)
			}
		}
	}

	out := command["output"]
	outputDocsArr := out.([]interface{})
	outputDocs, ok := outputDocsArr[0].(map[string]interface{})
	if ok {
		for k := range outputDocs {
			if strings.Contains(k, "proto") {
				outputName = strings.Replace(k, "*", "", 1)
			}
		}
	}

	return inputName, outputName, nil
}

// get recursivily all protos needed in a Unmarshal json
func getKeys(info map[string]interface{}, keysSet map[string]bool) {
	for k, v := range info {
		if strings.Contains(k, "*proto") {
			kew := strings.Replace(k, "*", "", 1)
			keysSet[kew] = true
		}

		listofouts, ok := v.([]interface{})
		if ok {
			for i := range listofouts {
				aux, ok := listofouts[i].(map[string]interface{})
				if !ok {
					continue
				}
				getKeys(aux, keysSet)
			}
		}

		aux, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		getKeys(aux, keysSet)
	}
}

func (pc *ProtoClient) getDescriptors(data string) error {
	d := []byte(data)
	var jsonmap interface{}
	if err := json.Unmarshal(d, &jsonmap); err != nil {
		return err
	}
	m := jsonmap.(map[string]interface{})
	keysSet := make(map[string]bool)
	getKeys(m, keysSet)

	// load predefined protos
	for _, commands := range pc.info.Commands {
		if commands.input != "" {
			keysSet[commands.input] = true
		}
		if commands.output != "" {
			keysSet[commands.output] = true
		}
	}

	// build commands reference
	handlers := m["handlers"].(map[string]interface{})
	for k, v := range handlers {
		cmdInfo := v.(map[string]interface{})
		in, out, err := getOutputInputNames(cmdInfo)
		if err != nil {
			return err
		}

		var command Command
		command.input = in
		command.output = out

		pc.info.Commands[k] = &command
		if pc.descriptorsRoute == "" && in == "protos.ProtoName" && out == "protos.ProtoDescriptor" {
			pc.descriptorsRoute = k
		}
		// c.Println(k);
	}

	remotes := m["remotes"].(map[string]interface{})
	for k, v := range remotes {
		cmdInfo := v.(map[string]interface{})
		in, out, err := getOutputInputNames(cmdInfo)
		if err != nil {
			return err
		}

		var command Command
		command.input = in
		command.output = out

		pc.info.Commands[k] = &command
		// c.Println(k);
	}

	// get all proto types
	descriptorArray := make([]*protobuf.FileDescriptorProto, 0)

	for key := range keysSet {
		protname := &protos.ProtoName{
			Name: key,
		}
		data, err := proto.Marshal(protname)
		if err != nil {
			return err
		}
		_, err = pc.SendRequest(pc.descriptorsRoute, data)
		if err != nil {
			return err
		}

		response := <-pc.Client.IncomingMsgChan

		protodecrip := &protos.ProtoDescriptor{}

		if err := proto.Unmarshal(response.Data, protodecrip); err != nil {
			return err
		}

		fileDescriptorProto, err := unpackDescriptor(protodecrip.Desc)
		if err != nil {
			return err
		}

		descriptorArray = append(descriptorArray, fileDescriptorProto)
		pc.descriptorsNames[key] = true
	}

	err := pc.buildProtosFromDescriptor(descriptorArray)
	if err != nil {
		return err
	}

	return nil
}

func new(docslogLevel logrus.Level, requestTimeout ...time.Duration) *ProtoClient {
	return &ProtoClient{
		Client:           *New(docslogLevel, requestTimeout...),
		descriptorsNames: make(map[string]bool),
		info: ProtoBufferInfo{
			Commands: make(map[string]*Command),
		},
		docsRoute:        "",
		descriptorsRoute: "",
		IncomingMsgChan:  make(chan *message.Message, 3),
		closeChan:        make(chan bool),
	}
}

func NewProto(docsRoute string, docslogLevel logrus.Level, requestTimeout ...time.Duration) *ProtoClient {
	newclient := new(docslogLevel, requestTimeout...)
	newclient.docsRoute = docsRoute
	return newclient
}

func NewWithDescriptor(descriptorsRoute string, docsRoute string, docslogLevel logrus.Level, requestTimeout ...time.Duration) *ProtoClient {
	newclient := new(docslogLevel, requestTimeout...)
	newclient.docsRoute = docsRoute
	newclient.descriptorsRoute = descriptorsRoute
	return newclient
}

// Load commands information form the server. Names is a list of protos names.
func (pc *ProtoClient) LoadServoInfo(addr string) error {
	pc.ready = false

	if err := pc.ConnectToTLS(addr, true); err != nil {
		if err.Error() == "EOF" {
			if err := pc.ConnectTo(addr); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// request doc info
	_, err := pc.SendRequest(pc.docsRoute, make([]byte, 0))
	if err != nil {
		return err
	}
	response := <-pc.Client.IncomingMsgChan

	docs := &protos.Doc{}
	if err := proto.Unmarshal(response.Data, docs); err != nil {
		return err
	}

	if err := pc.getDescriptors(docs.Doc); err != nil {
		return err
	}

	pc.Disconnect()
	pc.ready = true

	return nil
}

// Disconnect the client
func (pc *ProtoClient) Disconnect() {
	pc.Client.Disconnect()
	if pc.ready {
		pc.closeChan <- true
	}
}

func (pc *ProtoClient) waitForData() {
	if !pc.ready {
		return
	}
	for {
		select {
		case response := <-pc.Client.IncomingMsgChan:

			inputMsg := pc.expectedInput

			msg, ok := pc.info.Commands[response.Route]
			if ok {
				inputMsg = msg.outputMsg
				// fmt.Println(msg)
			} else {
				pc.expectedInput = nil
			}

			if inputMsg == nil {
				logger.Log.Errorf("Not expected data: %s", string(response.Data))
				continue
			}

			err := inputMsg.Unmarshal(response.Data)
			if err != nil {
				logger.Log.Errorf("Erro decode data: %s", string(response.Data))
				continue
			}

			data, err2 := inputMsg.MarshalJSON()
			if err2 != nil {
				logger.Log.Errorf("Erro encode data to json: %s", string(response.Data))
				continue
			}

			response.Data = data
			pc.IncomingMsgChan <- response
		case <-pc.closeChan:
			return
		}
	}
}

// ConnectToTLS connects to the server at addr using TLS, for now the only supported protocol is tcp
// this methods blocks as it also handles the messages from the server
func (pc *ProtoClient) ConnectToTLS(addr string, skipVerify bool) error {
	err := pc.Client.ConnectToTLS(addr, skipVerify)
	if err != nil {
		return err
	}
	go pc.waitForData()
	return nil
}

// ConnectTo connects to the server at addr, for now the only supported protocol is tcp
// this methods blocks as it also handles the messages from the server
func (pc *ProtoClient) ConnectTo(addr string) error {
	err := pc.Client.ConnectTo(addr)
	if err != nil {
		return err
	}
	go pc.waitForData()
	return nil
}

// Export sup[orted commands information
func (pc *ProtoClient) ExportInformation() *ProtoBufferInfo {
	if !pc.ready {
		return nil
	}
	return &pc.info
}

// Load commands information form ProtoBufferInfo
func (pc *ProtoClient) LoadInfo(info *ProtoBufferInfo) error {
	if info == nil {
		return errors.New("Protobuffer information invalid.")
	}
	pc.info = *info
	pc.ready = true
	return nil
}

// Add a push response. Must be ladded before LoadInfo.
func (pc *ProtoClient) AddPushResponse(route string, protoName string) {
	if route != "" && protoName != "" {
		var command Command
		command.input = ""
		command.output = protoName

		pc.info.Commands[route] = &command
	}
}

// SendRequest sends a request to the server
func (pc *ProtoClient) SendRequest(route string, data []byte) (uint, error) {

	if !pc.ready {
		return pc.Client.SendRequest(route, data)
	}

	if cmd, ok := pc.info.Commands[route]; ok {
		if len(data) < 0 || string(data) == "{}" || cmd.inputMsg == nil {
			pc.expectedInput = cmd.outputMsg
			data = data[:0]
			return pc.Client.SendRequest(route, data)
		}
		if err := cmd.inputMsg.UnmarshalJSON(data); err != nil {
			return 0, err
		}
		realdata, err := cmd.inputMsg.Marshal()
		if err != nil {
			return 0, err
		}
		pc.expectedInput = cmd.outputMsg
		return pc.Client.SendRequest(route, realdata)
	}

	return 0, errors.New("Invalid Route: " + route)
}

// SendNotify sends a notify to the server
func (pc *ProtoClient) SendNotify(route string, data []byte) error {

	if cmd, ok := pc.info.Commands[route]; ok {
		err := cmd.inputMsg.UnmarshalJSON(data)
		if err != nil {
			return err
		}
		realdata, err := cmd.inputMsg.Marshal()
		if err != nil {
			return err
		}
		return pc.Client.SendNotify(route, realdata)
	}

	return errors.New("Invalid Route.")
}
