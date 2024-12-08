// Code generated by MockGen. DO NOT EDIT.
// Source: internal/models/models.go
//
// Generated by this command:
//
//	mockgen -package mockm -destination=internal/mock/models/models.go -source=internal/models/models.go
//

// Package mockm is a generated GoMock package.
package mockm

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
	models "talkliketv.click/tltv/internal/models"
)

// MockModelsX is a mock of ModelsX interface.
type MockModelsX struct {
	ctrl     *gomock.Controller
	recorder *MockModelsXMockRecorder
	isgomock struct{}
}

// MockModelsXMockRecorder is the mock recorder for MockModelsX.
type MockModelsXMockRecorder struct {
	mock *MockModelsX
}

// NewMockModelsX creates a new mock instance.
func NewMockModelsX(ctrl *gomock.Controller) *MockModelsX {
	mock := &MockModelsX{ctrl: ctrl}
	mock.recorder = &MockModelsXMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockModelsX) EXPECT() *MockModelsXMockRecorder {
	return m.recorder
}

// GetLanguage mocks base method.
func (m *MockModelsX) GetLanguage(arg0 int) (models.Language, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLanguage", arg0)
	ret0, _ := ret[0].(models.Language)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLanguage indicates an expected call of GetLanguage.
func (mr *MockModelsXMockRecorder) GetLanguage(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLanguage", reflect.TypeOf((*MockModelsX)(nil).GetLanguage), arg0)
}

// GetVoice mocks base method.
func (m *MockModelsX) GetVoice(arg0 int) (models.Voice, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVoice", arg0)
	ret0, _ := ret[0].(models.Voice)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVoice indicates an expected call of GetVoice.
func (mr *MockModelsXMockRecorder) GetVoice(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVoice", reflect.TypeOf((*MockModelsX)(nil).GetVoice), arg0)
}