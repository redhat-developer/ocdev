// Code generated by MockGen. DO NOT EDIT.
// Source: localConfigProvider.go

// Package mock_localConfigProvider is a generated GoMock package.
package localConfigProvider

import (
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockLocalConfigProvider is a mock of LocalConfigProvider interface
type MockLocalConfigProvider struct {
	ctrl     *gomock.Controller
	recorder *MockLocalConfigProviderMockRecorder
}

// MockLocalConfigProviderMockRecorder is the mock recorder for MockLocalConfigProvider
type MockLocalConfigProviderMockRecorder struct {
	mock *MockLocalConfigProvider
}

// NewMockLocalConfigProvider creates a new mock instance
func NewMockLocalConfigProvider(ctrl *gomock.Controller) *MockLocalConfigProvider {
	mock := &MockLocalConfigProvider{ctrl: ctrl}
	mock.recorder = &MockLocalConfigProviderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockLocalConfigProvider) EXPECT() *MockLocalConfigProviderMockRecorder {
	return m.recorder
}

// GetApplication mocks base method
func (m *MockLocalConfigProvider) GetApplication() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetApplication")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetApplication indicates an expected call of GetApplication
func (mr *MockLocalConfigProviderMockRecorder) GetApplication() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetApplication", reflect.TypeOf((*MockLocalConfigProvider)(nil).GetApplication))
}

// GetName mocks base method
func (m *MockLocalConfigProvider) GetName() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetName")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetName indicates an expected call of GetName
func (mr *MockLocalConfigProviderMockRecorder) GetName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetName", reflect.TypeOf((*MockLocalConfigProvider)(nil).GetName))
}

// GetNamespace mocks base method
func (m *MockLocalConfigProvider) GetNamespace() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNamespace")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetNamespace indicates an expected call of GetNamespace
func (mr *MockLocalConfigProviderMockRecorder) GetNamespace() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNamespace", reflect.TypeOf((*MockLocalConfigProvider)(nil).GetNamespace))
}

// GetDebugPort mocks base method
func (m *MockLocalConfigProvider) GetDebugPort() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDebugPort")
	ret0, _ := ret[0].(int)
	return ret0
}

// GetDebugPort indicates an expected call of GetDebugPort
func (mr *MockLocalConfigProviderMockRecorder) GetDebugPort() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDebugPort", reflect.TypeOf((*MockLocalConfigProvider)(nil).GetDebugPort))
}

// GetContainers mocks base method
func (m *MockLocalConfigProvider) GetContainers() ([]LocalContainer, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetContainers")
	ret0, _ := ret[0].([]LocalContainer)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetContainers indicates an expected call of GetContainers
func (mr *MockLocalConfigProviderMockRecorder) GetContainers() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetContainers", reflect.TypeOf((*MockLocalConfigProvider)(nil).GetContainers))
}

// GetURL mocks base method
func (m *MockLocalConfigProvider) GetURL(name string) (*LocalURL, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetURL", name)
	ret0, _ := ret[0].(*LocalURL)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetURL indicates an expected call of GetURL
func (mr *MockLocalConfigProviderMockRecorder) GetURL(name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetURL", reflect.TypeOf((*MockLocalConfigProvider)(nil).GetURL), name)
}

// CompleteURL mocks base method
func (m *MockLocalConfigProvider) CompleteURL(url *LocalURL) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CompleteURL", url)
	ret0, _ := ret[0].(error)
	return ret0
}

// CompleteURL indicates an expected call of CompleteURL
func (mr *MockLocalConfigProviderMockRecorder) CompleteURL(url interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CompleteURL", reflect.TypeOf((*MockLocalConfigProvider)(nil).CompleteURL), url)
}

// ValidateURL mocks base method
func (m *MockLocalConfigProvider) ValidateURL(url LocalURL) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ValidateURL", url)
	ret0, _ := ret[0].(error)
	return ret0
}

// ValidateURL indicates an expected call of ValidateURL
func (mr *MockLocalConfigProviderMockRecorder) ValidateURL(url interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ValidateURL", reflect.TypeOf((*MockLocalConfigProvider)(nil).ValidateURL), url)
}

// CreateURL mocks base method
func (m *MockLocalConfigProvider) CreateURL(url LocalURL) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateURL", url)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateURL indicates an expected call of CreateURL
func (mr *MockLocalConfigProviderMockRecorder) CreateURL(url interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateURL", reflect.TypeOf((*MockLocalConfigProvider)(nil).CreateURL), url)
}

// DeleteURL mocks base method
func (m *MockLocalConfigProvider) DeleteURL(name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteURL", name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteURL indicates an expected call of DeleteURL
func (mr *MockLocalConfigProviderMockRecorder) DeleteURL(name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteURL", reflect.TypeOf((*MockLocalConfigProvider)(nil).DeleteURL), name)
}

// GetContainerPorts mocks base method
func (m *MockLocalConfigProvider) GetContainerPorts(container string) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetContainerPorts", container)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetContainerPorts indicates an expected call of GetContainerPorts
func (mr *MockLocalConfigProviderMockRecorder) GetContainerPorts(container interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetContainerPorts", reflect.TypeOf((*MockLocalConfigProvider)(nil).GetContainerPorts), container)
}

// GetComponentPorts mocks base method
func (m *MockLocalConfigProvider) GetComponentPorts() ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetComponentPorts")
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetComponentPorts indicates an expected call of GetComponentPorts
func (mr *MockLocalConfigProviderMockRecorder) GetComponentPorts() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetComponentPorts", reflect.TypeOf((*MockLocalConfigProvider)(nil).GetComponentPorts))
}

// ListURLs mocks base method
func (m *MockLocalConfigProvider) ListURLs() ([]LocalURL, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListURLs")
	ret0, _ := ret[0].([]LocalURL)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListURLs indicates an expected call of ListURLs
func (mr *MockLocalConfigProviderMockRecorder) ListURLs() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListURLs", reflect.TypeOf((*MockLocalConfigProvider)(nil).ListURLs))
}

// GetStorage mocks base method
func (m *MockLocalConfigProvider) GetStorage(name string) (*LocalStorage, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStorage", name)
	ret0, _ := ret[0].(*LocalStorage)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetStorage indicates an expected call of GetStorage
func (mr *MockLocalConfigProviderMockRecorder) GetStorage(name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStorage", reflect.TypeOf((*MockLocalConfigProvider)(nil).GetStorage), name)
}

// CompleteStorage mocks base method
func (m *MockLocalConfigProvider) CompleteStorage(storage *LocalStorage) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "CompleteStorage", storage)
}

// CompleteStorage indicates an expected call of CompleteStorage
func (mr *MockLocalConfigProviderMockRecorder) CompleteStorage(storage interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CompleteStorage", reflect.TypeOf((*MockLocalConfigProvider)(nil).CompleteStorage), storage)
}

// ValidateStorage mocks base method
func (m *MockLocalConfigProvider) ValidateStorage(storage LocalStorage) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ValidateStorage", storage)
	ret0, _ := ret[0].(error)
	return ret0
}

// ValidateStorage indicates an expected call of ValidateStorage
func (mr *MockLocalConfigProviderMockRecorder) ValidateStorage(storage interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ValidateStorage", reflect.TypeOf((*MockLocalConfigProvider)(nil).ValidateStorage), storage)
}

// CreateStorage mocks base method
func (m *MockLocalConfigProvider) CreateStorage(storage LocalStorage) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateStorage", storage)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateStorage indicates an expected call of CreateStorage
func (mr *MockLocalConfigProviderMockRecorder) CreateStorage(storage interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateStorage", reflect.TypeOf((*MockLocalConfigProvider)(nil).CreateStorage), storage)
}

// DeleteStorage mocks base method
func (m *MockLocalConfigProvider) DeleteStorage(name string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteStorage", name)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteStorage indicates an expected call of DeleteStorage
func (mr *MockLocalConfigProviderMockRecorder) DeleteStorage(name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteStorage", reflect.TypeOf((*MockLocalConfigProvider)(nil).DeleteStorage), name)
}

// ListStorage mocks base method
func (m *MockLocalConfigProvider) ListStorage() ([]LocalStorage, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListStorage")
	ret0, _ := ret[0].([]LocalStorage)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListStorage indicates an expected call of ListStorage
func (mr *MockLocalConfigProviderMockRecorder) ListStorage() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListStorage", reflect.TypeOf((*MockLocalConfigProvider)(nil).ListStorage))
}

// GetStorageMountPath mocks base method
func (m *MockLocalConfigProvider) GetStorageMountPath(storageName string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStorageMountPath", storageName)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetStorageMountPath indicates an expected call of GetStorageMountPath
func (mr *MockLocalConfigProviderMockRecorder) GetStorageMountPath(storageName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStorageMountPath", reflect.TypeOf((*MockLocalConfigProvider)(nil).GetStorageMountPath), storageName)
}

// Exists mocks base method
func (m *MockLocalConfigProvider) Exists() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Exists")
	ret0, _ := ret[0].(bool)
	return ret0
}

// Exists indicates an expected call of Exists
func (mr *MockLocalConfigProviderMockRecorder) Exists() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Exists", reflect.TypeOf((*MockLocalConfigProvider)(nil).Exists))
}
