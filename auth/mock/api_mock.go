// Code generated by MockGen. DO NOT EDIT.
// Source: api.go

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	auth "github.com/polarismesh/polaris/auth"
	api "github.com/polarismesh/polaris/cache/api"
	model "github.com/polarismesh/polaris/common/model"
	store "github.com/polarismesh/polaris/store"
	security "github.com/polarismesh/specification/source/go/api/v1/security"
	service_manage "github.com/polarismesh/specification/source/go/api/v1/service_manage"
)

// MockAuthChecker is a mock of AuthChecker interface.
type MockAuthChecker struct {
	ctrl     *gomock.Controller
	recorder *MockAuthCheckerMockRecorder
}

// MockAuthCheckerMockRecorder is the mock recorder for MockAuthChecker.
type MockAuthCheckerMockRecorder struct {
	mock *MockAuthChecker
}

// NewMockAuthChecker creates a new mock instance.
func NewMockAuthChecker(ctrl *gomock.Controller) *MockAuthChecker {
	mock := &MockAuthChecker{ctrl: ctrl}
	mock.recorder = &MockAuthCheckerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAuthChecker) EXPECT() *MockAuthCheckerMockRecorder {
	return m.recorder
}

// AllowResourceOperate mocks base method.
func (m *MockAuthChecker) AllowResourceOperate(ctx *model.AcquireContext, opInfo *model.ResourceOpInfo) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AllowResourceOperate", ctx, opInfo)
	ret0, _ := ret[0].(bool)
	return ret0
}

// AllowResourceOperate indicates an expected call of AllowResourceOperate.
func (mr *MockAuthCheckerMockRecorder) AllowResourceOperate(ctx, opInfo interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AllowResourceOperate", reflect.TypeOf((*MockAuthChecker)(nil).AllowResourceOperate), ctx, opInfo)
}

// CheckClientPermission mocks base method.
func (m *MockAuthChecker) CheckClientPermission(preCtx *model.AcquireContext) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckClientPermission", preCtx)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CheckClientPermission indicates an expected call of CheckClientPermission.
func (mr *MockAuthCheckerMockRecorder) CheckClientPermission(preCtx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckClientPermission", reflect.TypeOf((*MockAuthChecker)(nil).CheckClientPermission), preCtx)
}

// CheckConsolePermission mocks base method.
func (m *MockAuthChecker) CheckConsolePermission(preCtx *model.AcquireContext) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckConsolePermission", preCtx)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CheckConsolePermission indicates an expected call of CheckConsolePermission.
func (mr *MockAuthCheckerMockRecorder) CheckConsolePermission(preCtx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckConsolePermission", reflect.TypeOf((*MockAuthChecker)(nil).CheckConsolePermission), preCtx)
}

// IsOpenClientAuth mocks base method.
func (m *MockAuthChecker) IsOpenClientAuth() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsOpenClientAuth")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsOpenClientAuth indicates an expected call of IsOpenClientAuth.
func (mr *MockAuthCheckerMockRecorder) IsOpenClientAuth() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsOpenClientAuth", reflect.TypeOf((*MockAuthChecker)(nil).IsOpenClientAuth))
}

// IsOpenConsoleAuth mocks base method.
func (m *MockAuthChecker) IsOpenConsoleAuth() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsOpenConsoleAuth")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsOpenConsoleAuth indicates an expected call of IsOpenConsoleAuth.
func (mr *MockAuthCheckerMockRecorder) IsOpenConsoleAuth() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsOpenConsoleAuth", reflect.TypeOf((*MockAuthChecker)(nil).IsOpenConsoleAuth))
}

// MockStrategyServer is a mock of StrategyServer interface.
type MockStrategyServer struct {
	ctrl     *gomock.Controller
	recorder *MockStrategyServerMockRecorder
}

// MockStrategyServerMockRecorder is the mock recorder for MockStrategyServer.
type MockStrategyServerMockRecorder struct {
	mock *MockStrategyServer
}

// NewMockStrategyServer creates a new mock instance.
func NewMockStrategyServer(ctrl *gomock.Controller) *MockStrategyServer {
	mock := &MockStrategyServer{ctrl: ctrl}
	mock.recorder = &MockStrategyServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStrategyServer) EXPECT() *MockStrategyServerMockRecorder {
	return m.recorder
}

// AfterResourceOperation mocks base method.
func (m *MockStrategyServer) AfterResourceOperation(afterCtx *model.AcquireContext) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AfterResourceOperation", afterCtx)
	ret0, _ := ret[0].(error)
	return ret0
}

// AfterResourceOperation indicates an expected call of AfterResourceOperation.
func (mr *MockStrategyServerMockRecorder) AfterResourceOperation(afterCtx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AfterResourceOperation", reflect.TypeOf((*MockStrategyServer)(nil).AfterResourceOperation), afterCtx)
}

// CreateStrategy mocks base method.
func (m *MockStrategyServer) CreateStrategy(ctx context.Context, strategy *security.AuthStrategy) *service_manage.Response {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateStrategy", ctx, strategy)
	ret0, _ := ret[0].(*service_manage.Response)
	return ret0
}

// CreateStrategy indicates an expected call of CreateStrategy.
func (mr *MockStrategyServerMockRecorder) CreateStrategy(ctx, strategy interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateStrategy", reflect.TypeOf((*MockStrategyServer)(nil).CreateStrategy), ctx, strategy)
}

// DeleteStrategies mocks base method.
func (m *MockStrategyServer) DeleteStrategies(ctx context.Context, reqs []*security.AuthStrategy) *service_manage.BatchWriteResponse {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteStrategies", ctx, reqs)
	ret0, _ := ret[0].(*service_manage.BatchWriteResponse)
	return ret0
}

// DeleteStrategies indicates an expected call of DeleteStrategies.
func (mr *MockStrategyServerMockRecorder) DeleteStrategies(ctx, reqs interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteStrategies", reflect.TypeOf((*MockStrategyServer)(nil).DeleteStrategies), ctx, reqs)
}

// GetAuthChecker mocks base method.
func (m *MockStrategyServer) GetAuthChecker() auth.AuthChecker {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAuthChecker")
	ret0, _ := ret[0].(auth.AuthChecker)
	return ret0
}

// GetAuthChecker indicates an expected call of GetAuthChecker.
func (mr *MockStrategyServerMockRecorder) GetAuthChecker() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAuthChecker", reflect.TypeOf((*MockStrategyServer)(nil).GetAuthChecker))
}

// GetPrincipalResources mocks base method.
func (m *MockStrategyServer) GetPrincipalResources(ctx context.Context, query map[string]string) *service_manage.Response {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPrincipalResources", ctx, query)
	ret0, _ := ret[0].(*service_manage.Response)
	return ret0
}

// GetPrincipalResources indicates an expected call of GetPrincipalResources.
func (mr *MockStrategyServerMockRecorder) GetPrincipalResources(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPrincipalResources", reflect.TypeOf((*MockStrategyServer)(nil).GetPrincipalResources), ctx, query)
}

// GetStrategies mocks base method.
func (m *MockStrategyServer) GetStrategies(ctx context.Context, query map[string]string) *service_manage.BatchQueryResponse {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStrategies", ctx, query)
	ret0, _ := ret[0].(*service_manage.BatchQueryResponse)
	return ret0
}

// GetStrategies indicates an expected call of GetStrategies.
func (mr *MockStrategyServerMockRecorder) GetStrategies(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStrategies", reflect.TypeOf((*MockStrategyServer)(nil).GetStrategies), ctx, query)
}

// GetStrategy mocks base method.
func (m *MockStrategyServer) GetStrategy(ctx context.Context, strategy *security.AuthStrategy) *service_manage.Response {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStrategy", ctx, strategy)
	ret0, _ := ret[0].(*service_manage.Response)
	return ret0
}

// GetStrategy indicates an expected call of GetStrategy.
func (mr *MockStrategyServerMockRecorder) GetStrategy(ctx, strategy interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStrategy", reflect.TypeOf((*MockStrategyServer)(nil).GetStrategy), ctx, strategy)
}

// Initialize mocks base method.
func (m *MockStrategyServer) Initialize(options *auth.Config, storage store.Store, cacheMgr api.CacheManager, userSvr auth.UserServer) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Initialize", options, storage, cacheMgr, userSvr)
	ret0, _ := ret[0].(error)
	return ret0
}

// Initialize indicates an expected call of Initialize.
func (mr *MockStrategyServerMockRecorder) Initialize(options, storage, cacheMgr, userSvr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Initialize", reflect.TypeOf((*MockStrategyServer)(nil).Initialize), options, storage, cacheMgr, userSvr)
}

// Name mocks base method.
func (m *MockStrategyServer) Name() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

// Name indicates an expected call of Name.
func (mr *MockStrategyServerMockRecorder) Name() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Name", reflect.TypeOf((*MockStrategyServer)(nil).Name))
}

// UpdateStrategies mocks base method.
func (m *MockStrategyServer) UpdateStrategies(ctx context.Context, reqs []*security.ModifyAuthStrategy) *service_manage.BatchWriteResponse {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateStrategies", ctx, reqs)
	ret0, _ := ret[0].(*service_manage.BatchWriteResponse)
	return ret0
}

// UpdateStrategies indicates an expected call of UpdateStrategies.
func (mr *MockStrategyServerMockRecorder) UpdateStrategies(ctx, reqs interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateStrategies", reflect.TypeOf((*MockStrategyServer)(nil).UpdateStrategies), ctx, reqs)
}

// MockUserServer is a mock of UserServer interface.
type MockUserServer struct {
	ctrl     *gomock.Controller
	recorder *MockUserServerMockRecorder
}

// MockUserServerMockRecorder is the mock recorder for MockUserServer.
type MockUserServerMockRecorder struct {
	mock *MockUserServer
}

// NewMockUserServer creates a new mock instance.
func NewMockUserServer(ctrl *gomock.Controller) *MockUserServer {
	mock := &MockUserServer{ctrl: ctrl}
	mock.recorder = &MockUserServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUserServer) EXPECT() *MockUserServerMockRecorder {
	return m.recorder
}

// CheckCredential mocks base method.
func (m *MockUserServer) CheckCredential(authCtx *model.AcquireContext) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckCredential", authCtx)
	ret0, _ := ret[0].(error)
	return ret0
}

// CheckCredential indicates an expected call of CheckCredential.
func (mr *MockUserServerMockRecorder) CheckCredential(authCtx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckCredential", reflect.TypeOf((*MockUserServer)(nil).CheckCredential), authCtx)
}

// CreateGroup mocks base method.
func (m *MockUserServer) CreateGroup(ctx context.Context, group *security.UserGroup) *service_manage.Response {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateGroup", ctx, group)
	ret0, _ := ret[0].(*service_manage.Response)
	return ret0
}

// CreateGroup indicates an expected call of CreateGroup.
func (mr *MockUserServerMockRecorder) CreateGroup(ctx, group interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateGroup", reflect.TypeOf((*MockUserServer)(nil).CreateGroup), ctx, group)
}

// CreateUsers mocks base method.
func (m *MockUserServer) CreateUsers(ctx context.Context, users []*security.User) *service_manage.BatchWriteResponse {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateUsers", ctx, users)
	ret0, _ := ret[0].(*service_manage.BatchWriteResponse)
	return ret0
}

// CreateUsers indicates an expected call of CreateUsers.
func (mr *MockUserServerMockRecorder) CreateUsers(ctx, users interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateUsers", reflect.TypeOf((*MockUserServer)(nil).CreateUsers), ctx, users)
}

// DeleteGroups mocks base method.
func (m *MockUserServer) DeleteGroups(ctx context.Context, group []*security.UserGroup) *service_manage.BatchWriteResponse {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteGroups", ctx, group)
	ret0, _ := ret[0].(*service_manage.BatchWriteResponse)
	return ret0
}

// DeleteGroups indicates an expected call of DeleteGroups.
func (mr *MockUserServerMockRecorder) DeleteGroups(ctx, group interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteGroups", reflect.TypeOf((*MockUserServer)(nil).DeleteGroups), ctx, group)
}

// DeleteUsers mocks base method.
func (m *MockUserServer) DeleteUsers(ctx context.Context, users []*security.User) *service_manage.BatchWriteResponse {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteUsers", ctx, users)
	ret0, _ := ret[0].(*service_manage.BatchWriteResponse)
	return ret0
}

// DeleteUsers indicates an expected call of DeleteUsers.
func (mr *MockUserServerMockRecorder) DeleteUsers(ctx, users interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteUsers", reflect.TypeOf((*MockUserServer)(nil).DeleteUsers), ctx, users)
}

// GetGroup mocks base method.
func (m *MockUserServer) GetGroup(ctx context.Context, req *security.UserGroup) *service_manage.Response {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGroup", ctx, req)
	ret0, _ := ret[0].(*service_manage.Response)
	return ret0
}

// GetGroup indicates an expected call of GetGroup.
func (mr *MockUserServerMockRecorder) GetGroup(ctx, req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGroup", reflect.TypeOf((*MockUserServer)(nil).GetGroup), ctx, req)
}

// GetGroupToken mocks base method.
func (m *MockUserServer) GetGroupToken(ctx context.Context, group *security.UserGroup) *service_manage.Response {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGroupToken", ctx, group)
	ret0, _ := ret[0].(*service_manage.Response)
	return ret0
}

// GetGroupToken indicates an expected call of GetGroupToken.
func (mr *MockUserServerMockRecorder) GetGroupToken(ctx, group interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGroupToken", reflect.TypeOf((*MockUserServer)(nil).GetGroupToken), ctx, group)
}

// GetGroups mocks base method.
func (m *MockUserServer) GetGroups(ctx context.Context, query map[string]string) *service_manage.BatchQueryResponse {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGroups", ctx, query)
	ret0, _ := ret[0].(*service_manage.BatchQueryResponse)
	return ret0
}

// GetGroups indicates an expected call of GetGroups.
func (mr *MockUserServerMockRecorder) GetGroups(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGroups", reflect.TypeOf((*MockUserServer)(nil).GetGroups), ctx, query)
}

// GetUserHelper mocks base method.
func (m *MockUserServer) GetUserHelper() auth.UserHelper {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserHelper")
	ret0, _ := ret[0].(auth.UserHelper)
	return ret0
}

// GetUserHelper indicates an expected call of GetUserHelper.
func (mr *MockUserServerMockRecorder) GetUserHelper() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserHelper", reflect.TypeOf((*MockUserServer)(nil).GetUserHelper))
}

// GetUserToken mocks base method.
func (m *MockUserServer) GetUserToken(ctx context.Context, user *security.User) *service_manage.Response {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserToken", ctx, user)
	ret0, _ := ret[0].(*service_manage.Response)
	return ret0
}

// GetUserToken indicates an expected call of GetUserToken.
func (mr *MockUserServerMockRecorder) GetUserToken(ctx, user interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserToken", reflect.TypeOf((*MockUserServer)(nil).GetUserToken), ctx, user)
}

// GetUsers mocks base method.
func (m *MockUserServer) GetUsers(ctx context.Context, query map[string]string) *service_manage.BatchQueryResponse {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUsers", ctx, query)
	ret0, _ := ret[0].(*service_manage.BatchQueryResponse)
	return ret0
}

// GetUsers indicates an expected call of GetUsers.
func (mr *MockUserServerMockRecorder) GetUsers(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUsers", reflect.TypeOf((*MockUserServer)(nil).GetUsers), ctx, query)
}

// Initialize mocks base method.
func (m *MockUserServer) Initialize(authOpt *auth.Config, storage store.Store, cacheMgn api.CacheManager) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Initialize", authOpt, storage, cacheMgn)
	ret0, _ := ret[0].(error)
	return ret0
}

// Initialize indicates an expected call of Initialize.
func (mr *MockUserServerMockRecorder) Initialize(authOpt, storage, cacheMgn interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Initialize", reflect.TypeOf((*MockUserServer)(nil).Initialize), authOpt, storage, cacheMgn)
}

// Login mocks base method.
func (m *MockUserServer) Login(req *security.LoginRequest) *service_manage.Response {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Login", req)
	ret0, _ := ret[0].(*service_manage.Response)
	return ret0
}

// Login indicates an expected call of Login.
func (mr *MockUserServerMockRecorder) Login(req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Login", reflect.TypeOf((*MockUserServer)(nil).Login), req)
}

// Name mocks base method.
func (m *MockUserServer) Name() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

// Name indicates an expected call of Name.
func (mr *MockUserServerMockRecorder) Name() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Name", reflect.TypeOf((*MockUserServer)(nil).Name))
}

// ResetGroupToken mocks base method.
func (m *MockUserServer) ResetGroupToken(ctx context.Context, group *security.UserGroup) *service_manage.Response {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResetGroupToken", ctx, group)
	ret0, _ := ret[0].(*service_manage.Response)
	return ret0
}

// ResetGroupToken indicates an expected call of ResetGroupToken.
func (mr *MockUserServerMockRecorder) ResetGroupToken(ctx, group interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResetGroupToken", reflect.TypeOf((*MockUserServer)(nil).ResetGroupToken), ctx, group)
}

// ResetUserToken mocks base method.
func (m *MockUserServer) ResetUserToken(ctx context.Context, user *security.User) *service_manage.Response {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResetUserToken", ctx, user)
	ret0, _ := ret[0].(*service_manage.Response)
	return ret0
}

// ResetUserToken indicates an expected call of ResetUserToken.
func (mr *MockUserServerMockRecorder) ResetUserToken(ctx, user interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResetUserToken", reflect.TypeOf((*MockUserServer)(nil).ResetUserToken), ctx, user)
}

// UpdateGroupToken mocks base method.
func (m *MockUserServer) UpdateGroupToken(ctx context.Context, group *security.UserGroup) *service_manage.Response {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateGroupToken", ctx, group)
	ret0, _ := ret[0].(*service_manage.Response)
	return ret0
}

// UpdateGroupToken indicates an expected call of UpdateGroupToken.
func (mr *MockUserServerMockRecorder) UpdateGroupToken(ctx, group interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateGroupToken", reflect.TypeOf((*MockUserServer)(nil).UpdateGroupToken), ctx, group)
}

// UpdateGroups mocks base method.
func (m *MockUserServer) UpdateGroups(ctx context.Context, groups []*security.ModifyUserGroup) *service_manage.BatchWriteResponse {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateGroups", ctx, groups)
	ret0, _ := ret[0].(*service_manage.BatchWriteResponse)
	return ret0
}

// UpdateGroups indicates an expected call of UpdateGroups.
func (mr *MockUserServerMockRecorder) UpdateGroups(ctx, groups interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateGroups", reflect.TypeOf((*MockUserServer)(nil).UpdateGroups), ctx, groups)
}

// UpdateUser mocks base method.
func (m *MockUserServer) UpdateUser(ctx context.Context, user *security.User) *service_manage.Response {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateUser", ctx, user)
	ret0, _ := ret[0].(*service_manage.Response)
	return ret0
}

// UpdateUser indicates an expected call of UpdateUser.
func (mr *MockUserServerMockRecorder) UpdateUser(ctx, user interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateUser", reflect.TypeOf((*MockUserServer)(nil).UpdateUser), ctx, user)
}

// UpdateUserPassword mocks base method.
func (m *MockUserServer) UpdateUserPassword(ctx context.Context, req *security.ModifyUserPassword) *service_manage.Response {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateUserPassword", ctx, req)
	ret0, _ := ret[0].(*service_manage.Response)
	return ret0
}

// UpdateUserPassword indicates an expected call of UpdateUserPassword.
func (mr *MockUserServerMockRecorder) UpdateUserPassword(ctx, req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateUserPassword", reflect.TypeOf((*MockUserServer)(nil).UpdateUserPassword), ctx, req)
}

// UpdateUserToken mocks base method.
func (m *MockUserServer) UpdateUserToken(ctx context.Context, user *security.User) *service_manage.Response {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateUserToken", ctx, user)
	ret0, _ := ret[0].(*service_manage.Response)
	return ret0
}

// UpdateUserToken indicates an expected call of UpdateUserToken.
func (mr *MockUserServerMockRecorder) UpdateUserToken(ctx, user interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateUserToken", reflect.TypeOf((*MockUserServer)(nil).UpdateUserToken), ctx, user)
}

// MockUserOperator is a mock of UserOperator interface.
type MockUserOperator struct {
	ctrl     *gomock.Controller
	recorder *MockUserOperatorMockRecorder
}

// MockUserOperatorMockRecorder is the mock recorder for MockUserOperator.
type MockUserOperatorMockRecorder struct {
	mock *MockUserOperator
}

// NewMockUserOperator creates a new mock instance.
func NewMockUserOperator(ctrl *gomock.Controller) *MockUserOperator {
	mock := &MockUserOperator{ctrl: ctrl}
	mock.recorder = &MockUserOperatorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUserOperator) EXPECT() *MockUserOperatorMockRecorder {
	return m.recorder
}

// CreateUsers mocks base method.
func (m *MockUserOperator) CreateUsers(ctx context.Context, users []*security.User) *service_manage.BatchWriteResponse {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateUsers", ctx, users)
	ret0, _ := ret[0].(*service_manage.BatchWriteResponse)
	return ret0
}

// CreateUsers indicates an expected call of CreateUsers.
func (mr *MockUserOperatorMockRecorder) CreateUsers(ctx, users interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateUsers", reflect.TypeOf((*MockUserOperator)(nil).CreateUsers), ctx, users)
}

// DeleteUsers mocks base method.
func (m *MockUserOperator) DeleteUsers(ctx context.Context, users []*security.User) *service_manage.BatchWriteResponse {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteUsers", ctx, users)
	ret0, _ := ret[0].(*service_manage.BatchWriteResponse)
	return ret0
}

// DeleteUsers indicates an expected call of DeleteUsers.
func (mr *MockUserOperatorMockRecorder) DeleteUsers(ctx, users interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteUsers", reflect.TypeOf((*MockUserOperator)(nil).DeleteUsers), ctx, users)
}

// GetUserToken mocks base method.
func (m *MockUserOperator) GetUserToken(ctx context.Context, user *security.User) *service_manage.Response {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserToken", ctx, user)
	ret0, _ := ret[0].(*service_manage.Response)
	return ret0
}

// GetUserToken indicates an expected call of GetUserToken.
func (mr *MockUserOperatorMockRecorder) GetUserToken(ctx, user interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserToken", reflect.TypeOf((*MockUserOperator)(nil).GetUserToken), ctx, user)
}

// GetUsers mocks base method.
func (m *MockUserOperator) GetUsers(ctx context.Context, query map[string]string) *service_manage.BatchQueryResponse {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUsers", ctx, query)
	ret0, _ := ret[0].(*service_manage.BatchQueryResponse)
	return ret0
}

// GetUsers indicates an expected call of GetUsers.
func (mr *MockUserOperatorMockRecorder) GetUsers(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUsers", reflect.TypeOf((*MockUserOperator)(nil).GetUsers), ctx, query)
}

// ResetUserToken mocks base method.
func (m *MockUserOperator) ResetUserToken(ctx context.Context, user *security.User) *service_manage.Response {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResetUserToken", ctx, user)
	ret0, _ := ret[0].(*service_manage.Response)
	return ret0
}

// ResetUserToken indicates an expected call of ResetUserToken.
func (mr *MockUserOperatorMockRecorder) ResetUserToken(ctx, user interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResetUserToken", reflect.TypeOf((*MockUserOperator)(nil).ResetUserToken), ctx, user)
}

// UpdateUser mocks base method.
func (m *MockUserOperator) UpdateUser(ctx context.Context, user *security.User) *service_manage.Response {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateUser", ctx, user)
	ret0, _ := ret[0].(*service_manage.Response)
	return ret0
}

// UpdateUser indicates an expected call of UpdateUser.
func (mr *MockUserOperatorMockRecorder) UpdateUser(ctx, user interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateUser", reflect.TypeOf((*MockUserOperator)(nil).UpdateUser), ctx, user)
}

// UpdateUserPassword mocks base method.
func (m *MockUserOperator) UpdateUserPassword(ctx context.Context, req *security.ModifyUserPassword) *service_manage.Response {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateUserPassword", ctx, req)
	ret0, _ := ret[0].(*service_manage.Response)
	return ret0
}

// UpdateUserPassword indicates an expected call of UpdateUserPassword.
func (mr *MockUserOperatorMockRecorder) UpdateUserPassword(ctx, req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateUserPassword", reflect.TypeOf((*MockUserOperator)(nil).UpdateUserPassword), ctx, req)
}

// UpdateUserToken mocks base method.
func (m *MockUserOperator) UpdateUserToken(ctx context.Context, user *security.User) *service_manage.Response {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateUserToken", ctx, user)
	ret0, _ := ret[0].(*service_manage.Response)
	return ret0
}

// UpdateUserToken indicates an expected call of UpdateUserToken.
func (mr *MockUserOperatorMockRecorder) UpdateUserToken(ctx, user interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateUserToken", reflect.TypeOf((*MockUserOperator)(nil).UpdateUserToken), ctx, user)
}

// MockGroupOperator is a mock of GroupOperator interface.
type MockGroupOperator struct {
	ctrl     *gomock.Controller
	recorder *MockGroupOperatorMockRecorder
}

// MockGroupOperatorMockRecorder is the mock recorder for MockGroupOperator.
type MockGroupOperatorMockRecorder struct {
	mock *MockGroupOperator
}

// NewMockGroupOperator creates a new mock instance.
func NewMockGroupOperator(ctrl *gomock.Controller) *MockGroupOperator {
	mock := &MockGroupOperator{ctrl: ctrl}
	mock.recorder = &MockGroupOperatorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockGroupOperator) EXPECT() *MockGroupOperatorMockRecorder {
	return m.recorder
}

// CreateGroup mocks base method.
func (m *MockGroupOperator) CreateGroup(ctx context.Context, group *security.UserGroup) *service_manage.Response {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateGroup", ctx, group)
	ret0, _ := ret[0].(*service_manage.Response)
	return ret0
}

// CreateGroup indicates an expected call of CreateGroup.
func (mr *MockGroupOperatorMockRecorder) CreateGroup(ctx, group interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateGroup", reflect.TypeOf((*MockGroupOperator)(nil).CreateGroup), ctx, group)
}

// DeleteGroups mocks base method.
func (m *MockGroupOperator) DeleteGroups(ctx context.Context, group []*security.UserGroup) *service_manage.BatchWriteResponse {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteGroups", ctx, group)
	ret0, _ := ret[0].(*service_manage.BatchWriteResponse)
	return ret0
}

// DeleteGroups indicates an expected call of DeleteGroups.
func (mr *MockGroupOperatorMockRecorder) DeleteGroups(ctx, group interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteGroups", reflect.TypeOf((*MockGroupOperator)(nil).DeleteGroups), ctx, group)
}

// GetGroup mocks base method.
func (m *MockGroupOperator) GetGroup(ctx context.Context, req *security.UserGroup) *service_manage.Response {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGroup", ctx, req)
	ret0, _ := ret[0].(*service_manage.Response)
	return ret0
}

// GetGroup indicates an expected call of GetGroup.
func (mr *MockGroupOperatorMockRecorder) GetGroup(ctx, req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGroup", reflect.TypeOf((*MockGroupOperator)(nil).GetGroup), ctx, req)
}

// GetGroupToken mocks base method.
func (m *MockGroupOperator) GetGroupToken(ctx context.Context, group *security.UserGroup) *service_manage.Response {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGroupToken", ctx, group)
	ret0, _ := ret[0].(*service_manage.Response)
	return ret0
}

// GetGroupToken indicates an expected call of GetGroupToken.
func (mr *MockGroupOperatorMockRecorder) GetGroupToken(ctx, group interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGroupToken", reflect.TypeOf((*MockGroupOperator)(nil).GetGroupToken), ctx, group)
}

// GetGroups mocks base method.
func (m *MockGroupOperator) GetGroups(ctx context.Context, query map[string]string) *service_manage.BatchQueryResponse {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGroups", ctx, query)
	ret0, _ := ret[0].(*service_manage.BatchQueryResponse)
	return ret0
}

// GetGroups indicates an expected call of GetGroups.
func (mr *MockGroupOperatorMockRecorder) GetGroups(ctx, query interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGroups", reflect.TypeOf((*MockGroupOperator)(nil).GetGroups), ctx, query)
}

// ResetGroupToken mocks base method.
func (m *MockGroupOperator) ResetGroupToken(ctx context.Context, group *security.UserGroup) *service_manage.Response {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResetGroupToken", ctx, group)
	ret0, _ := ret[0].(*service_manage.Response)
	return ret0
}

// ResetGroupToken indicates an expected call of ResetGroupToken.
func (mr *MockGroupOperatorMockRecorder) ResetGroupToken(ctx, group interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResetGroupToken", reflect.TypeOf((*MockGroupOperator)(nil).ResetGroupToken), ctx, group)
}

// UpdateGroupToken mocks base method.
func (m *MockGroupOperator) UpdateGroupToken(ctx context.Context, group *security.UserGroup) *service_manage.Response {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateGroupToken", ctx, group)
	ret0, _ := ret[0].(*service_manage.Response)
	return ret0
}

// UpdateGroupToken indicates an expected call of UpdateGroupToken.
func (mr *MockGroupOperatorMockRecorder) UpdateGroupToken(ctx, group interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateGroupToken", reflect.TypeOf((*MockGroupOperator)(nil).UpdateGroupToken), ctx, group)
}

// UpdateGroups mocks base method.
func (m *MockGroupOperator) UpdateGroups(ctx context.Context, groups []*security.ModifyUserGroup) *service_manage.BatchWriteResponse {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateGroups", ctx, groups)
	ret0, _ := ret[0].(*service_manage.BatchWriteResponse)
	return ret0
}

// UpdateGroups indicates an expected call of UpdateGroups.
func (mr *MockGroupOperatorMockRecorder) UpdateGroups(ctx, groups interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateGroups", reflect.TypeOf((*MockGroupOperator)(nil).UpdateGroups), ctx, groups)
}

// MockUserHelper is a mock of UserHelper interface.
type MockUserHelper struct {
	ctrl     *gomock.Controller
	recorder *MockUserHelperMockRecorder
}

// MockUserHelperMockRecorder is the mock recorder for MockUserHelper.
type MockUserHelperMockRecorder struct {
	mock *MockUserHelper
}

// NewMockUserHelper creates a new mock instance.
func NewMockUserHelper(ctrl *gomock.Controller) *MockUserHelper {
	mock := &MockUserHelper{ctrl: ctrl}
	mock.recorder = &MockUserHelperMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUserHelper) EXPECT() *MockUserHelperMockRecorder {
	return m.recorder
}

// CheckGroupsExist mocks base method.
func (m *MockUserHelper) CheckGroupsExist(ctx context.Context, groups []*security.UserGroup) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckGroupsExist", ctx, groups)
	ret0, _ := ret[0].(error)
	return ret0
}

// CheckGroupsExist indicates an expected call of CheckGroupsExist.
func (mr *MockUserHelperMockRecorder) CheckGroupsExist(ctx, groups interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckGroupsExist", reflect.TypeOf((*MockUserHelper)(nil).CheckGroupsExist), ctx, groups)
}

// CheckUserInGroup mocks base method.
func (m *MockUserHelper) CheckUserInGroup(ctx context.Context, group *security.UserGroup, user *security.User) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckUserInGroup", ctx, group, user)
	ret0, _ := ret[0].(bool)
	return ret0
}

// CheckUserInGroup indicates an expected call of CheckUserInGroup.
func (mr *MockUserHelperMockRecorder) CheckUserInGroup(ctx, group, user interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckUserInGroup", reflect.TypeOf((*MockUserHelper)(nil).CheckUserInGroup), ctx, group, user)
}

// CheckUsersExist mocks base method.
func (m *MockUserHelper) CheckUsersExist(ctx context.Context, users []*security.User) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CheckUsersExist", ctx, users)
	ret0, _ := ret[0].(error)
	return ret0
}

// CheckUsersExist indicates an expected call of CheckUsersExist.
func (mr *MockUserHelperMockRecorder) CheckUsersExist(ctx, users interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckUsersExist", reflect.TypeOf((*MockUserHelper)(nil).CheckUsersExist), ctx, users)
}

// GetGroup mocks base method.
func (m *MockUserHelper) GetGroup(ctx context.Context, req *security.UserGroup) *security.UserGroup {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGroup", ctx, req)
	ret0, _ := ret[0].(*security.UserGroup)
	return ret0
}

// GetGroup indicates an expected call of GetGroup.
func (mr *MockUserHelperMockRecorder) GetGroup(ctx, req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGroup", reflect.TypeOf((*MockUserHelper)(nil).GetGroup), ctx, req)
}

// GetUser mocks base method.
func (m *MockUserHelper) GetUser(ctx context.Context, user *security.User) *security.User {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUser", ctx, user)
	ret0, _ := ret[0].(*security.User)
	return ret0
}

// GetUser indicates an expected call of GetUser.
func (mr *MockUserHelperMockRecorder) GetUser(ctx, user interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUser", reflect.TypeOf((*MockUserHelper)(nil).GetUser), ctx, user)
}

// GetUserByID mocks base method.
func (m *MockUserHelper) GetUserByID(ctx context.Context, id string) *security.User {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserByID", ctx, id)
	ret0, _ := ret[0].(*security.User)
	return ret0
}

// GetUserByID indicates an expected call of GetUserByID.
func (mr *MockUserHelperMockRecorder) GetUserByID(ctx, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserByID", reflect.TypeOf((*MockUserHelper)(nil).GetUserByID), ctx, id)
}

// GetUserOwnGroup mocks base method.
func (m *MockUserHelper) GetUserOwnGroup(ctx context.Context, user *security.User) []*security.UserGroup {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserOwnGroup", ctx, user)
	ret0, _ := ret[0].([]*security.UserGroup)
	return ret0
}

// GetUserOwnGroup indicates an expected call of GetUserOwnGroup.
func (mr *MockUserHelperMockRecorder) GetUserOwnGroup(ctx, user interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserOwnGroup", reflect.TypeOf((*MockUserHelper)(nil).GetUserOwnGroup), ctx, user)
}
