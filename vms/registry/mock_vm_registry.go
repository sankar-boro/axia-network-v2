// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Code generated by MockGen. DO NOT EDIT.
// Source: vms/registry/vm_registry.go

// Package registry is a generated GoMock package.
package registry

import (
	reflect "reflect"

	ids "github.com/sankar-boro/avalanchego/ids"
	gomock "github.com/golang/mock/gomock"
)

// MockVMRegistry is a mock of VMRegistry interface.
type MockVMRegistry struct {
	ctrl     *gomock.Controller
	recorder *MockVMRegistryMockRecorder
}

// MockVMRegistryMockRecorder is the mock recorder for MockVMRegistry.
type MockVMRegistryMockRecorder struct {
	mock *MockVMRegistry
}

// NewMockVMRegistry creates a new mock instance.
func NewMockVMRegistry(ctrl *gomock.Controller) *MockVMRegistry {
	mock := &MockVMRegistry{ctrl: ctrl}
	mock.recorder = &MockVMRegistryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockVMRegistry) EXPECT() *MockVMRegistryMockRecorder {
	return m.recorder
}

// Reload mocks base method.
func (m *MockVMRegistry) Reload() ([]ids.ID, map[ids.ID]error, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Reload")
	ret0, _ := ret[0].([]ids.ID)
	ret1, _ := ret[1].(map[ids.ID]error)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Reload indicates an expected call of Reload.
func (mr *MockVMRegistryMockRecorder) Reload() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Reload", reflect.TypeOf((*MockVMRegistry)(nil).Reload))
}

// ReloadWithReadLock mocks base method.
func (m *MockVMRegistry) ReloadWithReadLock() ([]ids.ID, map[ids.ID]error, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReloadWithReadLock")
	ret0, _ := ret[0].([]ids.ID)
	ret1, _ := ret[1].(map[ids.ID]error)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// ReloadWithReadLock indicates an expected call of ReloadWithReadLock.
func (mr *MockVMRegistryMockRecorder) ReloadWithReadLock() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReloadWithReadLock", reflect.TypeOf((*MockVMRegistry)(nil).ReloadWithReadLock))
}
