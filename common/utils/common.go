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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/golang/protobuf/ptypes/wrappers"
	apimodel "github.com/polarismesh/specification/source/go/api/v1/model"
	apiservice "github.com/polarismesh/specification/source/go/api/v1/service_manage"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/wrapperspb"

	api "github.com/polarismesh/polaris/common/api/v1"
	"github.com/polarismesh/polaris/common/log"
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

	// ratelimit表
	MaxDbRateLimitName = MaxRuleName

	MaxRuleName = 64

	MaxPlatformIDLength     = 32
	MaxPlatformNameLength   = 128
	MaxPlatformDomainLength = 1024
	MaxPlatformQPS          = 65535
)

var resourceNameRE = regexp.MustCompile("^[0-9A-Za-z-./:_]+$")

// CheckResourceName 检查资源Name
func CheckResourceName(name *wrappers.StringValue) error {
	if name == nil {
		return errors.New(NilErrString)
	}

	if name.GetValue() == "" {
		return errors.New(EmptyErrString)
	}

	if ok := resourceNameRE.MatchString(name.GetValue()); !ok {
		return errors.New("name contains invalid character")
	}

	return nil
}

// CheckResourceOwners 检查资源Owners
func CheckResourceOwners(owners *wrappers.StringValue) error {
	if owners == nil {
		return errors.New(NilErrString)
	}

	if owners.GetValue() == "" {
		return errors.New(EmptyErrString)
	}

	if utf8.RuneCountInString(owners.GetValue()) > MaxOwnersLength {
		return errors.New("owners too long")
	}

	return nil
}

// CheckInstanceHost 检查服务实例Host
func CheckInstanceHost(host *wrappers.StringValue) error {
	if host == nil {
		return errors.New(NilErrString)
	}

	if host.GetValue() == "" {
		return errors.New(EmptyErrString)
	}

	return nil
}

// CheckInstancePort 检查服务实例Port
func CheckInstancePort(port *wrappers.UInt32Value) error {
	if port == nil {
		return errors.New(NilErrString)
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

// ParseRequestID 从ctx中获取Request-ID
func ParseRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	rid, _ := ctx.Value(StringContext("request-id")).(string)
	return rid
}

// ParseClientAddress 从ctx中获取客户端地址
func ParseClientAddress(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	rid, _ := ctx.Value(ContextClientAddress).(string)
	return rid
}

// ParseClientIP .
func ParseClientIP(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	rid, _ := ctx.Value(ContextClientAddress).(string)
	if strings.Contains(rid, ":") {
		return strings.Split(rid, ":")[0]
	}
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
	if userName == "" {
		return ParseOperator(ctx)
	}
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

	if operator, _ := ctx.Value(ContextOperator).(string); operator != "" {
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

// RequestID 从ctx中获取Request-ID
func RequestID(ctx context.Context) zap.Field {
	return zap.String("request-id", ParseRequestID(ctx))
}

// ZapPlatformID 生成Platform-ID的日志描述
func ZapPlatformID(id string) zap.Field {
	return zap.String("platform-id", id)
}

// ZapInstanceID 生成instanceID的日志描述
func ZapInstanceID(id string) zap.Field {
	return zap.String("instance-id", id)
}

// ZapNamespace 生成namespace的日志描述
func ZapNamespace(namespace string) zap.Field {
	return zap.String("namesapce", namespace)
}

// ZapGroup 生成group的日志描述
func ZapGroup(group string) zap.Field {
	return zap.String("group", group)
}

// ZapFileName 生成fileName的日志描述
func ZapFileName(fileName string) zap.Field {
	return zap.String("file-name", fileName)
}

// ZapReleaseName 生成fileName的日志描述
func ZapReleaseName(fileName string) zap.Field {
	return zap.String("release-name", fileName)
}

// ZapVersion 生成 version 的日志描述
func ZapVersion(version uint64) zap.Field {
	return zap.Uint64("version", version)
}

// CheckDbStrFieldLen 检查name字段是否超过DB中对应字段的最大字符长度限制
func CheckDbStrFieldLen(param *wrappers.StringValue, dbLen int) error {
	return CheckDbRawStrFieldLen(param.GetValue(), dbLen)
}

// CheckDbRawStrFieldLen 检查name字段是否超过DB中对应字段的最大字符长度限制
func CheckDbRawStrFieldLen(param string, dbLen int) error {
	if param != "" && utf8.RuneCountInString(param) > dbLen {
		errMsg := fmt.Sprintf("length of %s is over %d", param, dbLen)
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
func CheckInstanceTetrad(req *apiservice.Instance) (string, *apiservice.Response) {
	if err := CheckResourceName(req.GetService()); err != nil {
		return "", api.NewInstanceResponse(apimodel.Code_InvalidServiceName, req)
	}

	if err := CheckResourceName(req.GetNamespace()); err != nil {
		return "", api.NewInstanceResponse(apimodel.Code_InvalidNamespaceName, req)
	}

	if err := CheckInstanceHost(req.GetHost()); err != nil {
		return "", api.NewInstanceResponse(apimodel.Code_InvalidInstanceHost, req)
	}

	if err := CheckInstancePort(req.GetPort()); err != nil {
		return "", api.NewInstanceResponse(apimodel.Code_InvalidInstancePort, req)
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
			return "", api.NewInstanceResponse(apimodel.Code_ExecuteException, req)
		}
		instID = id
	}
	return instID, nil
}

// ConvertStringValuesToSlice 转换StringValues为字符串切片
func ConvertStringValuesToSlice(vals []*wrapperspb.StringValue) []string {
	ret := make([]string, 0, 4)

	for index := range vals {
		id := vals[index]
		if strings.TrimSpace(id.GetValue()) == "" {
			continue
		}
		ret = append(ret, id.GetValue())
	}

	return ret
}

// CheckContractTetrad 根据服务实例四元组计算ID
func CheckContractTetrad(req *apiservice.ServiceContract) (string, *apiservice.Response) {
	str := fmt.Sprintf("%s##%s##%s##%s##%s", req.GetNamespace(), req.GetService(), req.GetName(),
		req.GetProtocol(), req.GetVersion())

	h := sha1.New()
	if _, err := io.WriteString(h, str); err != nil {
		return "", api.NewResponse(apimodel.Code_ExecuteException)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// BuildSha1Digest 构建SHA1摘要
func BuildSha1Digest(value string) (string, error) {
	if len(value) == 0 {
		return "", nil
	}
	h := sha1.New()
	if _, err := io.WriteString(h, value); err != nil {
		return "", err
	}
	out := hex.EncodeToString(h.Sum(nil))
	return out, nil
}

func CheckContractInterfaceTetrad(contractId string, source apiservice.InterfaceDescriptor_Source,
	req *apiservice.InterfaceDescriptor) (string, *apiservice.Response) {
	if contractId == "" {
		return "", api.NewResponseWithMsg(apimodel.Code_BadRequest, "invalid service_contract id")
	}
	if req.GetId() != "" {
		return req.GetId(), nil
	}
	if req.GetPath() == "" {
		return "", api.NewResponseWithMsg(apimodel.Code_BadRequest, "invalid service_contract interface path")
	}
	h := sha1.New()
	str := fmt.Sprintf("%s##%s##%s##%s##%d", contractId, req.GetMethod(), req.GetPath(), req.GetName(), source)

	if _, err := io.WriteString(h, str); err != nil {
		return "", api.NewResponseWithMsg(apimodel.Code_ExecuteException, err.Error())
	}
	out := hex.EncodeToString(h.Sum(nil))
	return out, nil
}

func CalculateContractID(namespace, service, name, protocol, version string) (string, error) {
	h := sha1.New()
	str := fmt.Sprintf("%s##%s##%s##%s##%s", namespace, service, name, protocol, version)

	if _, err := io.WriteString(h, str); err != nil {
		return "", err
	}

	out := hex.EncodeToString(h.Sum(nil))
	return out, nil
}

// ConvertMetadataToStringValue 将Metadata转换为可序列化字符串
func ConvertMetadataToStringValue(metadata map[string]string) (string, error) {
	if metadata == nil {
		return "", nil
	}
	v, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}
	return string(v), nil
}

// ConvertStringValueToMetadata 将字符串反序列为metadata
func ConvertStringValueToMetadata(str string) (map[string]string, error) {
	if str == "" {
		return nil, nil
	}
	v := make(map[string]string)
	err := json.Unmarshal([]byte(str), &v)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// NeedUpdateMetadata 判断是否出现了metadata的变更
func NeedUpdateMetadata(metadata map[string]string, inMetadata map[string]string) bool {
	if inMetadata == nil {
		return false
	}
	if len(metadata) != len(inMetadata) {
		return true
	}
	return !reflect.DeepEqual(metadata, inMetadata)
}
