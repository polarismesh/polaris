/**
 * Tencent is pleased to support the open source community by making Polaris available.
 *
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 *
 * Licensed under the BSD 3-Clause License (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://opensource.org/licenses/BSD-3-Clause
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package auth

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apisecurity "github.com/polarismesh/specification/source/go/api/v1/security"
	"google.golang.org/protobuf/types/known/wrapperspb"

	commontime "github.com/polarismesh/polaris/common/time"
	"github.com/polarismesh/polaris/common/utils"
)

var (
	// ErrorNoUser 没有找到对应的用户
	ErrorNoUser error = errors.New("no such user")

	// ErrorNoUserGroup 没有找到对应的用户组
	ErrorNoUserGroup error = errors.New("no such user group")
	// ErrorWrongUsernameOrPassword 用户或者密码错误
	ErrorWrongUsernameOrPassword error = errors.New("name or password is wrong")
	// ErrorTokenNotExist token 不存在
	ErrorTokenNotExist error = errors.New("token not exist")
	// ErrorTokenInvalid 非法的 token
	ErrorTokenInvalid error = errors.New("invalid token")
	// ErrorTokenDisabled token 已经被禁用
	ErrorTokenDisabled error = errors.New("token already disabled")
)

func ConvertToErrCode(err error) apimodel.Code {
	if errors.Is(err, ErrorTokenNotExist) {
		return apimodel.Code_TokenNotExisted
	}

	if errors.Is(err, ErrorTokenDisabled) {
		return apimodel.Code_TokenDisabled
	}

	return apimodel.Code_NotAllowedAccess
}

const (
	OperatorRoleKey      string = "operator_role"
	OperatorIDKey        string = "operator_id"
	OperatorOwnerKey     string = "operator_owner"
	OperatorLinkStrategy string = "operator_link_strategy"
	PrincipalKey         string = "principal"
	LinkUsersKey         string = "link_users"
	LinkGroupsKey        string = "link_groups"
	RemoveLinkUsersKey   string = "remove_link_users"
	RemoveLinkGroupsKey  string = "remove_link_groups"

	TokenDetailInfoKey string = "TokenInfo"
	TokenForUser       string = "uid"
	TokenForUserGroup  string = "groupid"

	ResourceAttachmentKey string = "resource_attachment"
)

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[PrincipalUser-1]
	_ = x[PrincipalGroup-2]
}

const _PrincipalType_name = "PrincipalUserPrincipalGroup"

var _PrincipalType_index = [...]uint8{0, 13, 27}

func (i PrincipalType) String() string {
	i -= 1
	if i < 0 || i >= PrincipalType(len(_PrincipalType_index)-1) {
		return "PrincipalType(" + strconv.FormatInt(int64(i+1), 10) + ")"
	}
	return _PrincipalType_name[_PrincipalType_index[i]:_PrincipalType_index[i+1]]
}

//go:generate stringer -type=PrincipalType
type PrincipalType int

const (
	PrincipalUser  PrincipalType = 1
	PrincipalGroup PrincipalType = 2
	PrincipalRole  PrincipalType = 3
)

// CheckPrincipalType 检查鉴权策略成员角色信息
func CheckPrincipalType(role int) error {
	switch PrincipalType(role) {
	case PrincipalUser:
		return nil
	case PrincipalGroup:
		return nil
	case PrincipalRole:
		return nil
	default:
		return errors.New("invalid principal type")
	}
}

var (
	// PrincipalNames principal name map
	PrincipalNames = map[PrincipalType]string{
		PrincipalUser:  "user",
		PrincipalGroup: "group",
		PrincipalRole:  "role",
	}
)

const (

	// DefaultStrategySuffix 默认策略的名称前缀
	DefaultStrategySuffix string = "的默认策略"
)

// BuildDefaultStrategyName 构建默认鉴权策略的名称信息
func BuildDefaultStrategyName(role PrincipalType, name string) string {
	if role == PrincipalUser {
		return fmt.Sprintf("%s%s%s", "(用户) ", name, DefaultStrategySuffix)
	}
	return fmt.Sprintf("%s%s%s", "(用户组) ", name, DefaultStrategySuffix)
}

// ResourceOperation 资源操作
type ResourceOperation int16

const (
	// Read 只读动作
	Read ResourceOperation = 10
	// Create 创建动作
	Create ResourceOperation = 20
	// Modify 修改动作
	Modify ResourceOperation = 30
	// Delete 删除动作
	Delete ResourceOperation = 40
)

// BzModule 模块标识
type BzModule int16

const (
	// UnknowModule 未知模块
	UnknowModule BzModule = iota
	// CoreModule 核心模块
	CoreModule
	// DiscoverModule 服务模块
	DiscoverModule
	// ConfigModule 配置模块
	ConfigModule
	// AuthModule 鉴权模块
	AuthModule
	// MaintainModule 运维操作模块
	MaintainModule
	// BootstrapModule 初始化模块
	BootstrapModule
)

// UserRoleType 用户角色类型
type UserRoleType int

const (
	UnknownUserRole    UserRoleType = -1
	AdminUserRole      UserRoleType = 0
	OwnerUserRole      UserRoleType = 20
	SubAccountUserRole UserRoleType = 50
)

var (
	UserRoleNames = map[UserRoleType]string{
		AdminUserRole:      "admin",
		OwnerUserRole:      "main",
		SubAccountUserRole: "sub",
	}
)

// ResourceEntry 资源最简单信息
type ResourceEntry struct {
	Type     apisecurity.ResourceType
	ID       string
	Owner    string
	Metadata map[string]string
}

// User 用户
type User struct {
	ID          string
	Name        string
	Password    string
	Owner       string
	Source      string
	Mobile      string
	Email       string
	Type        UserRoleType
	Metadata    map[string]string
	Token       string
	TokenEnable bool
	Valid       bool
	Comment     string
	CreateTime  time.Time
	ModifyTime  time.Time
}

func (u *User) GetToken() string {
	return u.Token
}

func (u *User) Disable() bool {
	return !u.TokenEnable
}

func (u *User) OwnerID() string {
	return u.Owner
}

func (u *User) SelfID() string {
	return u.ID
}

func (u *User) ToSpec() *apisecurity.User {
	if u == nil {
		return nil
	}
	return &apisecurity.User{
		Id:          wrapperspb.String(u.ID),
		Name:        wrapperspb.String(u.Name),
		Password:    wrapperspb.String(u.Password),
		Owner:       wrapperspb.String(u.Owner),
		Source:      wrapperspb.String(u.Source),
		AuthToken:   wrapperspb.String(u.Token),
		TokenEnable: wrapperspb.Bool(u.TokenEnable),
		Comment:     wrapperspb.String(u.Comment),
		UserType:    wrapperspb.String(fmt.Sprintf("%d", u.Type)),
	}
}

// UserGroupDetail 用户组详细（带用户列表）
type UserGroupDetail struct {
	*UserGroup

	// UserIds改为 map 的形式，加速查询
	UserIds map[string]struct{}
}

// ToUserIdSlice 将用户ID Map 专为 slice
func (ugd *UserGroupDetail) ToUserIdSlice() []string {
	uids := make([]string, 0, len(ugd.UserIds))
	for uid := range ugd.UserIds {
		uids = append(uids, uid)
	}
	return uids
}

func (ugd *UserGroupDetail) ListSpecUser() []*apisecurity.User {
	users := make([]*apisecurity.User, 0, len(ugd.UserIds))
	for i := range ugd.UserIds {
		users = append(users, &apisecurity.User{
			Id: wrapperspb.String(i),
		})
	}
	return users
}

// ToSpec 将用户ID Map 专为 slice
func (ugd *UserGroupDetail) ToSpec() *apisecurity.UserGroup {
	if ugd == nil {
		return nil
	}
	return &apisecurity.UserGroup{
		Id:          wrapperspb.String(ugd.ID),
		Name:        wrapperspb.String(ugd.Name),
		Owner:       wrapperspb.String(ugd.Owner),
		AuthToken:   wrapperspb.String(ugd.Token),
		TokenEnable: wrapperspb.Bool(ugd.TokenEnable),
		Comment:     wrapperspb.String(ugd.Comment),
		Ctime:       wrapperspb.String(commontime.Time2String(ugd.CreateTime)),
		Mtime:       wrapperspb.String(commontime.Time2String(ugd.ModifyTime)),
		Relation: &apisecurity.UserGroupRelation{
			GroupId: wrapperspb.String(ugd.ID),
			Users:   ugd.ListSpecUser(),
		},
		UserCount: wrapperspb.UInt32(uint32(len(ugd.UserIds))),
	}
}

// UserGroup 用户组
type UserGroup struct {
	ID          string
	Name        string
	Owner       string
	Token       string
	TokenEnable bool
	Metadata    map[string]string
	Valid       bool
	Comment     string
	Source      string
	CreateTime  time.Time
	ModifyTime  time.Time
}

func (u *UserGroup) GetToken() string {
	return u.Token
}

func (u *UserGroup) Disable() bool {
	return !u.TokenEnable
}

func (u *UserGroup) OwnerID() string {
	return u.Owner
}

func (u *UserGroup) SelfID() string {
	return u.ID
}

// ModifyUserGroup 用户组修改
type ModifyUserGroup struct {
	ID            string
	Owner         string
	Token         string
	TokenEnable   bool
	Comment       string
	Metadata      map[string]string
	AddUserIds    []string
	RemoveUserIds []string
}

// UserGroupRelation 用户-用户组关联关系具体信息
type UserGroupRelation struct {
	GroupID    string
	UserIds    []string
	CreateTime time.Time
	ModifyTime time.Time
}

type Condition struct {
	Key         string
	Value       string
	CompareFunc string
}

// StrategyDetail 鉴权策略详细
type StrategyDetail struct {
	ID   string
	Name string
	// Action: 只有 allow 以及 deny
	Action  string
	Comment string
	Default bool
	Owner   string
	// CalleeMethods 允许访问的服务端接口
	CalleeMethods []string
	Resources     []StrategyResource
	Conditions    []Condition
	Principals    []Principal
	Valid         bool
	Revision      string
	Metadata      map[string]string
	CreateTime    time.Time
	ModifyTime    time.Time
}

func (s *StrategyDetail) IsDeny() bool {
	return s.Action == apisecurity.AuthAction_DENY.String()
}

func NewPolicyDetailCache(d *StrategyDetail) *PolicyDetailCache {
	users := make(map[string]Principal, 0)
	groups := make(map[string]Principal, 0)

	for index := range d.Principals {
		principal := d.Principals[index]
		if principal.PrincipalType == PrincipalUser {
			users[principal.PrincipalID] = principal
		} else {
			groups[principal.PrincipalID] = principal
		}
	}

	resources := map[apisecurity.ResourceType]*utils.SyncSet[string]{}
	for index := range d.Resources {
		resource := d.Resources[index]
		resType := apisecurity.ResourceType(resource.ResType)
		if _, ok := resources[resType]; !ok {
			resources[resType] = utils.NewSyncSet[string]()
		}
		resources[resType].Add(resource.ResID)
	}

	return &PolicyDetailCache{
		StrategyDetail: d,
		UserPrincipal:  users,
		GroupPrincipal: groups,
		ResourceDict:   resources,
	}
}

// PolicyDetailCache 鉴权策略详细
type PolicyDetailCache struct {
	*StrategyDetail
	UserPrincipal  map[string]Principal
	GroupPrincipal map[string]Principal
	ResourceDict   map[apisecurity.ResourceType]*utils.SyncSet[string]
}

// ModifyStrategyDetail 修改鉴权策略详细
type ModifyStrategyDetail struct {
	ID               string
	Name             string
	Action           string
	Comment          string
	Metadata         map[string]string
	AddPrincipals    []Principal
	RemovePrincipals []Principal
	AddResources     []StrategyResource
	RemoveResources  []StrategyResource
	ModifyTime       time.Time
}

// Strategy 策略main信息
type Strategy struct {
	ID         string
	Name       string
	Principal  string
	Action     string
	Comment    string
	Owner      string
	Default    bool
	Valid      bool
	CreateTime time.Time
	ModifyTime time.Time
}

// StrategyResource 策略资源
type StrategyResource struct {
	StrategyID string
	ResType    int32
	ResID      string
}

// Principal 策略相关人
type Principal struct {
	StrategyID    string
	Name          string
	Owner         string
	PrincipalID   string
	PrincipalType PrincipalType
}

func (p Principal) String() string {
	return fmt.Sprintf("%s/%s", p.PrincipalType.String(), p.PrincipalID)
}

// ParseUserRole 从ctx中解析用户角色
func ParseUserRole(ctx context.Context) UserRoleType {
	if ctx == nil {
		return SubAccountUserRole
	}

	role, _ := ctx.Value(utils.ContextUserRoleIDKey).(UserRoleType)
	return role
}

type Role struct {
	ID         string
	Name       string
	Owner      string
	Source     string
	Type       string
	Metadata   map[string]string
	Valid      bool
	Comment    string
	CreateTime time.Time
	ModifyTime time.Time
	Users      []*User
	UserGroups []*UserGroup
}
