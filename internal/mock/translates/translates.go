// Code generated by MockGen. DO NOT EDIT.
// Source: internal/translates/translates.go
//
// Generated by this command:
//
//	mockgen -package mockt -destination=internal/mock/translates/translates.go -source=internal/translates/translates.go
//

// Package mockt is a generated GoMock package.
package mockt

import (
	context "context"
	reflect "reflect"

	texttospeechpb "cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	translate "cloud.google.com/go/translate"
	gax "github.com/googleapis/gax-go/v2"
	echo "github.com/labstack/echo/v4"
	gomock "go.uber.org/mock/gomock"
	language "golang.org/x/text/language"
	models "talkliketv.click/tltv/internal/models"
)

// MockTranslateClientX is a mock of TranslateClientX interface.
type MockTranslateClientX struct {
	ctrl     *gomock.Controller
	recorder *MockTranslateClientXMockRecorder
	isgomock struct{}
}

// MockTranslateClientXMockRecorder is the mock recorder for MockTranslateClientX.
type MockTranslateClientXMockRecorder struct {
	mock *MockTranslateClientX
}

// NewMockTranslateClientX creates a new mock instance.
func NewMockTranslateClientX(ctrl *gomock.Controller) *MockTranslateClientX {
	mock := &MockTranslateClientX{ctrl: ctrl}
	mock.recorder = &MockTranslateClientXMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTranslateClientX) EXPECT() *MockTranslateClientXMockRecorder {
	return m.recorder
}

// Translate mocks base method.
func (m *MockTranslateClientX) Translate(arg0 context.Context, arg1 []string, arg2 language.Tag, arg3 *translate.Options) ([]translate.Translation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Translate", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].([]translate.Translation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Translate indicates an expected call of Translate.
func (mr *MockTranslateClientXMockRecorder) Translate(arg0, arg1, arg2, arg3 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Translate", reflect.TypeOf((*MockTranslateClientX)(nil).Translate), arg0, arg1, arg2, arg3)
}

// MockTTSClientX is a mock of TTSClientX interface.
type MockTTSClientX struct {
	ctrl     *gomock.Controller
	recorder *MockTTSClientXMockRecorder
	isgomock struct{}
}

// MockTTSClientXMockRecorder is the mock recorder for MockTTSClientX.
type MockTTSClientXMockRecorder struct {
	mock *MockTTSClientX
}

// NewMockTTSClientX creates a new mock instance.
func NewMockTTSClientX(ctrl *gomock.Controller) *MockTTSClientX {
	mock := &MockTTSClientX{ctrl: ctrl}
	mock.recorder = &MockTTSClientXMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTTSClientX) EXPECT() *MockTTSClientXMockRecorder {
	return m.recorder
}

// SynthesizeSpeech mocks base method.
func (m *MockTTSClientX) SynthesizeSpeech(arg0 context.Context, arg1 *texttospeechpb.SynthesizeSpeechRequest, arg2 ...gax.CallOption) (*texttospeechpb.SynthesizeSpeechResponse, error) {
	m.ctrl.T.Helper()
	varargs := []any{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "SynthesizeSpeech", varargs...)
	ret0, _ := ret[0].(*texttospeechpb.SynthesizeSpeechResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SynthesizeSpeech indicates an expected call of SynthesizeSpeech.
func (mr *MockTTSClientXMockRecorder) SynthesizeSpeech(arg0, arg1 any, arg2 ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SynthesizeSpeech", reflect.TypeOf((*MockTTSClientX)(nil).SynthesizeSpeech), varargs...)
}

// MockTranslateX is a mock of TranslateX interface.
type MockTranslateX struct {
	ctrl     *gomock.Controller
	recorder *MockTranslateXMockRecorder
	isgomock struct{}
}

// MockTranslateXMockRecorder is the mock recorder for MockTranslateX.
type MockTranslateXMockRecorder struct {
	mock *MockTranslateX
}

// NewMockTranslateX creates a new mock instance.
func NewMockTranslateX(ctrl *gomock.Controller) *MockTranslateX {
	mock := &MockTranslateX{ctrl: ctrl}
	mock.recorder = &MockTranslateXMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTranslateX) EXPECT() *MockTranslateXMockRecorder {
	return m.recorder
}

// CreateTTS mocks base method.
func (m *MockTranslateX) CreateTTS(arg0 echo.Context, arg1 models.Title, arg2 int, arg3 string) ([]models.Phrase, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateTTS", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].([]models.Phrase)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateTTS indicates an expected call of CreateTTS.
func (mr *MockTranslateXMockRecorder) CreateTTS(arg0, arg1, arg2, arg3 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateTTS", reflect.TypeOf((*MockTranslateX)(nil).CreateTTS), arg0, arg1, arg2, arg3)
}

// TranslatePhrases mocks base method.
func (m *MockTranslateX) TranslatePhrases(arg0 echo.Context, arg1 []models.Phrase, arg2 models.Language) ([]models.Phrase, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TranslatePhrases", arg0, arg1, arg2)
	ret0, _ := ret[0].([]models.Phrase)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// TranslatePhrases indicates an expected call of TranslatePhrases.
func (mr *MockTranslateXMockRecorder) TranslatePhrases(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TranslatePhrases", reflect.TypeOf((*MockTranslateX)(nil).TranslatePhrases), arg0, arg1, arg2)
}
