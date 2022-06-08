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

package utils

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/golang/protobuf/ptypes/wrappers"
	"go.uber.org/zap"

	"github.com/polarismesh/polaris-server/common/log"
	"github.com/polarismesh/polaris-server/common/model"
	"github.com/polarismesh/polaris-server/store"

	api "github.com/polarismesh/polaris-server/common/api/v1"
)

// some options config
const (
	// QueryDefaultOffset default query offset
	QueryDefaultOffset = 0
	// QueryDefaultLimit default query limit
	QueryDefaultLimit = 100
	// QueryMaxLimit default query max
	QueryMaxLimit = 100
	// MaxBatchSize max batch size
	MaxBatchSize = 100
	// MaxQuerySize max query size
	MaxQuerySize = 100

	// MaxMetadataLength metadata max length
	MaxMetadataLength = 64

	MaxBusinessLength   = 64
	MaxOwnersLength     = 1024
	MaxDepartmentLength = 1024
	MaxCommentLength    = 1024
	MaxNameLength       = 64

	// service表
	MaxDbServiceNameLength      = 128
	MaxDbServiceNamespaceLength = 64
	MaxDbServicePortsLength     = 8192
	MaxDbServiceBusinessLength  = 128
	MaxDbServiceDeptLength      = 1024
	MaxDbServiceCMDBLength      = 1024
	MaxDbServiceCommentLength   = 1024
	MaxDbServiceOwnerLength     = 1024
	MaxDbServiceToken           = 2048

	// instance表
	MaxDbInsHostLength     = 128
	MaxDbInsProtocolLength = 32
	MaxDbInsVersionLength  = 32
	MaxDbInsLogicSetLength = 128

	// circuitbreaker表
	MaxDbCircuitbreakerName       = 32
	MaxDbCircuitbreakerNamespace  = 64
	MaxDbCircuitbreakerBusiness   = 64
	MaxDbCircuitbreakerDepartment = 1024
	MaxDbCircuitbreakerComment    = 1024
	MaxDbCircuitbreakerOwner      = 1024
	MaxDbCircuitbreakerVersion    = 32

	// platform表
	MaxPlatformIDLength     = 32
	MaxPlatformNameLength   = 128
	MaxPlatformDomainLength = 1024
	MaxPlatformQPS          = 65535
)

// CheckResourceName 检查资源Name
func CheckResourceName(name *wrappers.StringValue) error {
	if name == nil {
		return errors.New("nil")
	}

	if name.GetValue() == "" {
		return errors.New("empty")
	}

	regStr := "^[0-9A-Za-z-./:_]+$"
	ok, err := regexp.MatchString(regStr, name.GetValue())
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("name contains invalid character")
	}

	return nil
}

// CheckResourceOwners 检查资源Owners
func CheckResourceOwners(owners *wrappers.StringValue) error {
	if owners == nil {
		return errors.New("nil")
	}

	if owners.GetValue() == "" {
		return errors.New("empty")
	}

	if utf8.RuneCountInString(owners.GetValue()) > MaxOwnersLength {
		return errors.New("owners too long")
	}

	return nil
}

// CheckInstanceHost 检查服务实例Host
func CheckInstanceHost(host *wrappers.StringValue) error {
	if host == nil {
		return errors.New("nil")
	}

	if host.GetValue() == "" {
		return errors.New("empty")
	}

	return nil
}

// CheckInstancePort 检查服务实例Port
func CheckInstancePort(port *wrappers.UInt32Value) error {
	if port == nil {
		return errors.New("nil")
	}

	if port.GetValue() < 0 {
		return errors.New("empty")
	}

	return nil
}

// CheckMetadata check metadata
// 检查metadata的个数 最大是64个
// key/value是否符合要求
func CheckMetadata(meta map[string]string) error {
	if meta == nil {
		return nil
	}

	if len(meta) > MaxMetadataLength {
		return errors.New("metadata is too long")
	}

	/*regStr := "^[0-9A-Za-z-._*]+$"
	   matchFunc := func(str string) error {
	  	 if str == "" {
	  		 return nil
	  	 }
	  	 ok, err := regexp.MatchString(regStr, str)
	  	 if err != nil {
	  		 log.Errorf("regexp match string(%s) err: %s", str, err.Error())
	  		 return err
	  	 }
	  	 if !ok {
	  		 log.Errorf("metadata string(%s) contains invalid character", str)
	  		 return errors.New("contain invalid character")
	  	 }
	  	 return nil
	   }
	   for key, value := range meta {
	  	 if err := matchFunc(key); err != nil {
	  		 return err
	  	 }
	  	 if err := matchFunc(value); err != nil {
	  		 return err
	  	 }
	   }*/

	return nil
}

// CheckQueryOffset 检查查询参数Offset
func CheckQueryOffset(offset []string) (int, error) {
	if len(offset) == 0 {
		return 0, nil
	}

	if len(offset) > 1 {
		return 0, errors.New("unique")
	}

	value, err := strconv.Atoi(offset[0])
	if err != nil {
		return 0, err
	}

	if value < 0 {
		return 0, errors.New("invalid")
	}

	return value, nil
}

// CheckQueryLimit 检查查询参数Limit
func CheckQueryLimit(limit []string) (int, error) {
	if len(limit) == 0 {
		return MaxQuerySize, nil
	}

	if len(limit) > 1 {
		return 0, errors.New("unique")
	}

	value, err := strconv.Atoi(limit[0])
	if err != nil {
		return 0, err
	}

	if value < 0 {
		return 0, errors.New("invalid")
	}

	if value > MaxQuerySize {
		value = MaxQuerySize
	}

	return value, nil
}

// CalculateInstanceID 计算实例ID
func CalculateInstanceID(namespace string, service string, vpcID string, host string, port uint32) (string, error) {
	h := sha1.New()
	var str string
	// 兼容带有vpcID的instance
	if vpcID == "" {
		str = fmt.Sprintf("%s##%s##%s##%d", namespace, service, host, port)
	} else {
		str = fmt.Sprintf("%s##%s##%s##%s##%d", namespace, service, vpcID, host, port)
	}

	if _, err := io.WriteString(h, str); err != nil {
		return "", err
	}

	out := hex.EncodeToString(h.Sum(nil))
	return out, nil
}

// CalculateRuleID 计算规则ID
func CalculateRuleID(name, namespace string) string {
	return name + "." + namespace
}

// ParseQueryOffset 格式化处理offset参数
func ParseQueryOffset(offset string) (uint32, error) {
	if offset == "" {
		return QueryDefaultOffset, nil
	}

	tmp, err := strconv.ParseUint(offset, 10, 32)
	if err != nil {
		log.Errorf("[Server][Query] attribute(offset:%s) is invalid, parse err: %s",
			offset, err.Error())
		return 0, err
	}

	return uint32(tmp), nil
}

// ParseQueryLimit 格式化处理limit参数
func ParseQueryLimit(limit string) (uint32, error) {
	if limit == "" {
		return QueryDefaultLimit, nil
	}

	tmp, err := strconv.ParseUint(limit, 10, 32)
	if err != nil {
		log.Errorf("[Server][Query] attribute(offset:%s) is invalid, parse err: %s",
			limit, err.Error())
		return 0, err
	}
	if tmp > QueryMaxLimit {
		tmp = QueryMaxLimit
	}

	return uint32(tmp), nil
}

// ParseOffsetAndLimit 统一格式化处理Offset和limit参数
func ParseOffsetAndLimit(query map[string]string) (uint32, uint32, error) {
	ofs, err := ParseQueryOffset(query["offset"])
	if err != nil {
		return 0, 0, err
	}
	delete(query, "offset")

	var lmt uint32
	lmt, err = ParseQueryLimit(query["limit"])
	if err != nil {
		return 0, 0, err
	}
	delete(query, "limit")

	return ofs, lmt, nil
}

// ParseInstanceArgs 解析服务实例的 ip 和 port 查询参数
func ParseInstanceArgs(query map[string]string) (*store.InstanceArgs, error) {
	if len(query) == 0 {
		return nil, nil
	}
	hosts, ok := query["host"]
	if !ok {
		return nil, fmt.Errorf("port parameter can not be used alone without host")
	}
	res := &store.InstanceArgs{}
	res.Hosts = strings.Split(hosts, ",")
	ports, ok := query["port"]
	if !ok {
		return res, nil
	}

	portSlices := strings.Split(ports, ",")
	for _, portStr := range portSlices {
		port, err := strconv.ParseUint(portStr, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("%s can not parse as uint, err is %s", portStr, err.Error())
		}
		res.Ports = append(res.Ports, uint32(port))
	}
	return res, nil
}

// ParseRequestID 从ctx中获取Request-ID
func ParseRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	rid, _ := ctx.Value(StringContext("request-id")).(string)
	return rid
}

func ParseClientAddress(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	rid, _ := ctx.Value(ContextClientAddress).(string)
	return rid
}


// ParseAuthToken 从ctx中获取token
func ParseAuthToken(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	token, _ := ctx.Value(ContextAuthTokenKey).(string)
	return token
}

// ParseIsOwner 从ctx中获取token
func ParseIsOwner(ctx context.Context) bool {
	if ctx == nil {
		return false
	}

	isOwner, _ := ctx.Value(ContextIsOwnerKey).(bool)
	return isOwner
}

// ParseUserRole 从ctx中解析用户角色
func ParseUserRole(ctx context.Context) model.UserRoleType {
	if ctx == nil {
		return model.SubAccountUserRole
	}

	role, _ := ctx.Value(ContextUserRoleIDKey).(model.UserRoleType)
	return role
}

// ParseUserID 从ctx中解析用户ID
func ParseUserID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	userID, _ := ctx.Value(ContextUserIDKey).(string)
	return userID
}

// ParseUserName 从ctx解析用户名称
func ParseUserName(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	userName, _ := ctx.Value(ContextUserNameKey).(string)
	return userName
}

// ParseOwnerID 从ctx解析Owner ID
func ParseOwnerID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	ownerID, _ := ctx.Value(ContextOwnerIDKey).(string)
	return ownerID
}

// ParseToken 从ctx中获取token
func ParseToken(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	token, _ := ctx.Value(StringContext("polaris-token")).(string)
	return token
}

// ParseOperator 从ctx中获取operator
func ParseOperator(ctx context.Context) string {
	defaultOperator := "Polaris"
	if ctx == nil {
		return defaultOperator
	}

	if operator, _ := ctx.Value(StringContext("operator")).(string); operator != "" {
		return operator
	}

	return defaultOperator
}

// ParsePlatformID 从ctx中获取Platform-Id
func ParsePlatformID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	pid, _ := ctx.Value(StringContext("platform-id")).(string)
	return pid
}

// ParsePlatformToken 从ctx中获取Platform-Token
func ParsePlatformToken(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	pToken, _ := ctx.Value(StringContext("platform-token")).(string)
	return pToken
}

// ZapRequestID 生成Request-ID的日志描述
func ZapRequestID(id string) zap.Field {
	return zap.String("request-id", id)
}

// ZapRequestIDByCtx 从ctx中获取Request-ID
func ZapRequestIDByCtx(ctx context.Context) zap.Field {
	return zap.String("request-id", ParseRequestID(ctx))
}

// ZapPlatformID 生成Platform-ID的日志描述
func ZapPlatformID(id string) zap.Field {
	return zap.String("platform-id", id)
}

// CheckDbStrFieldLen 检查name字段是否超过DB中对应字段的最大字符长度限制
func CheckDbStrFieldLen(param *wrappers.StringValue, dbLen int) error {
	if param.GetValue() != "" && utf8.RuneCountInString(param.GetValue()) > dbLen {
		errMsg := fmt.Sprintf("length of %s is over %d", param.GetValue(), dbLen)
		return errors.New(errMsg)
	}
	return nil
}

// CheckDbMetaDataFieldLen 检查metadata的K,V是否超过DB中对应字段的最大字符长度限制
func CheckDbMetaDataFieldLen(metaData map[string]string) error {
	for k, v := range metaData {
		if utf8.RuneCountInString(k) > 128 || utf8.RuneCountInString(v) > 4096 {
			errMsg := fmt.Sprintf("metadata:length of key(%s) or value(%s) is over size(key:128,value:4096)",
				k, v)
			return errors.New(errMsg)
		}
	}
	return nil
}

// CheckInstanceTetrad 根据服务实例四元组计算ID
func CheckInstanceTetrad(req *api.Instance) (string, *api.Response) {
	if err := CheckResourceName(req.GetService()); err != nil {
		return "", api.NewInstanceResponse(api.InvalidServiceName, req)
	}

	if err := CheckResourceName(req.GetNamespace()); err != nil {
		return "", api.NewInstanceResponse(api.InvalidNamespaceName, req)
	}

	if err := CheckInstanceHost(req.GetHost()); err != nil {
		return "", api.NewInstanceResponse(api.InvalidInstanceHost, req)
	}

	if err := CheckInstancePort(req.GetPort()); err != nil {
		return "", api.NewInstanceResponse(api.InvalidInstancePort, req)
	}

	var instID = req.GetId().GetValue()
	if len(instID) == 0 {
		id, err := CalculateInstanceID(
			req.GetNamespace().GetValue(),
			req.GetService().GetValue(),
			req.GetVpcId().GetValue(),
			req.GetHost().GetValue(),
			req.GetPort().GetValue(),
		)
		if err != nil {
			return "", api.NewInstanceResponse(api.ExecuteException, req)
		}
		instID = id
	}
	return instID, nil
}
