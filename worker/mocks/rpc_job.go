// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/long12310225/pitaya/v2/worker (interfaces: RPCJob)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	gomock "github.com/golang/mock/gomock"
	proto "github.com/golang/protobuf/proto"
	reflect "reflect"
)

// MockRPCJob is a mock of RPCJob interface
type MockRPCJob struct {
	ctrl     *gomock.Controller
	recorder *MockRPCJobMockRecorder
}

// MockRPCJobMockRecorder is the mock recorder for MockRPCJob
type MockRPCJobMockRecorder struct {
	mock *MockRPCJob
}

// NewMockRPCJob creates a new mock instance
func NewMockRPCJob(ctrl *gomock.Controller) *MockRPCJob {
	mock := &MockRPCJob{ctrl: ctrl}
	mock.recorder = &MockRPCJobMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockRPCJob) EXPECT() *MockRPCJobMockRecorder {
	return m.recorder
}

// GetArgReply mocks base method
func (m *MockRPCJob) GetArgReply(arg0 string) (proto.Message, proto.Message, error) {
	ret := m.ctrl.Call(m, "GetArgReply", arg0)
	ret0, _ := ret[0].(proto.Message)
	ret1, _ := ret[1].(proto.Message)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetArgReply indicates an expected call of GetArgReply
func (mr *MockRPCJobMockRecorder) GetArgReply(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetArgReply", reflect.TypeOf((*MockRPCJob)(nil).GetArgReply), arg0)
}

// RPC mocks base method
func (m *MockRPCJob) RPC(arg0 context.Context, arg1, arg2 string, arg3, arg4 proto.Message) error {
	ret := m.ctrl.Call(m, "RPC", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].(error)
	return ret0
}

// RPC indicates an expected call of RPC
func (mr *MockRPCJobMockRecorder) RPC(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RPC", reflect.TypeOf((*MockRPCJob)(nil).RPC), arg0, arg1, arg2, arg3, arg4)
}

// ServerDiscovery mocks base method
func (m *MockRPCJob) ServerDiscovery(arg0 string, arg1 map[string]interface{}) (string, error) {
	ret := m.ctrl.Call(m, "ServerDiscovery", arg0, arg1)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ServerDiscovery indicates an expected call of ServerDiscovery
func (mr *MockRPCJobMockRecorder) ServerDiscovery(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ServerDiscovery", reflect.TypeOf((*MockRPCJob)(nil).ServerDiscovery), arg0, arg1)
}
