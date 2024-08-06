// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/topfreegames/pitaya/v3/pkg/worker (interfaces: RPCJob)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
)

// MockRPCJob is a mock of RPCJob interface.
type MockRPCJob struct {
	ctrl     *gomock.Controller
	recorder *MockRPCJobMockRecorder
}

// MockRPCJobMockRecorder is the mock recorder for MockRPCJob.
type MockRPCJobMockRecorder struct {
	mock *MockRPCJob
}

// NewMockRPCJob creates a new mock instance.
func NewMockRPCJob(ctrl *gomock.Controller) *MockRPCJob {
	mock := &MockRPCJob{ctrl: ctrl}
	mock.recorder = &MockRPCJobMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRPCJob) EXPECT() *MockRPCJobMockRecorder {
	return m.recorder
}

// GetArgReply mocks base method.
func (m *MockRPCJob) GetArgReply(arg0 string) (protoreflect.ProtoMessage, protoreflect.ProtoMessage, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetArgReply", arg0)
	ret0, _ := ret[0].(protoreflect.ProtoMessage)
	ret1, _ := ret[1].(protoreflect.ProtoMessage)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetArgReply indicates an expected call of GetArgReply.
func (mr *MockRPCJobMockRecorder) GetArgReply(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetArgReply", reflect.TypeOf((*MockRPCJob)(nil).GetArgReply), arg0)
}

// RPC mocks base method.
func (m *MockRPCJob) RPC(arg0 context.Context, arg1, arg2 string, arg3, arg4 protoreflect.ProtoMessage) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RPC", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].(error)
	return ret0
}

// RPC indicates an expected call of RPC.
func (mr *MockRPCJobMockRecorder) RPC(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RPC", reflect.TypeOf((*MockRPCJob)(nil).RPC), arg0, arg1, arg2, arg3, arg4)
}

// ServerDiscovery mocks base method.
func (m *MockRPCJob) ServerDiscovery(arg0 string, arg1 map[string]interface{}) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ServerDiscovery", arg0, arg1)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ServerDiscovery indicates an expected call of ServerDiscovery.
func (mr *MockRPCJobMockRecorder) ServerDiscovery(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ServerDiscovery", reflect.TypeOf((*MockRPCJob)(nil).ServerDiscovery), arg0, arg1)
}
